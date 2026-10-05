// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package authguest

import (
	"context"
	"errors"
	"fmt"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"

	"github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/jwt"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/pin"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/storage"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/token"
	revaevents "github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

var ErrExpired = errors.New("token expired")
var ErrAlreadyRedeemed = errors.New("token already redeemed")
var ErrShareNotFound = errors.New("share not found")
var ErrShareExpired = errors.New("share expired")
var ErrEventsNotConfigured = errors.New("event publisher not configured")

const guestLinkTokenTTL = 30 * time.Minute
const guestPinTTL = 30 * time.Minute

// RedeemError wraps a redeem failure together with the share id. The HTTP
// transport inspects ErrorType to choose a status code and message.
type RedeemError struct {
	ErrorType error
	ShareID   string
}

func (e *RedeemError) Error() string { return e.ErrorType.Error() }

// RenewError is an alias for RedeemError, used by the renew flow.
type RenewError = RedeemError

// RedeemResponse is the result of a successful token redemption.
type RedeemResponse struct {
	SessionToken string
	ShareID      string
}

// AuthGuest is the domain service used by the transport and event layers.
type AuthGuest interface {
	CreateToken(ctx context.Context, shareID string) (*token.Token, error)
	Redeem(ctx context.Context, tokenString string) (*RedeemResponse, error)
	Renew(ctx context.Context, shareID string) error
	CleanupShare(shareID string) error
}

var _ AuthGuest = (*AuthGuestService)(nil)

// AuthGuestService contains the business logic shared by auth-guest transport services.
type AuthGuestService struct {
	tokenSvc        *token.TokenService
	store           storage.Manager
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
	serviceAccount  config.ServiceAccount
	jwtService      *jwt.JwtService
	publisher       revaevents.Publisher
}

func NewAuthGuestService(tokenSvc *token.TokenService, store storage.Manager, opts ...Option) *AuthGuestService {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	return &AuthGuestService{
		tokenSvc:        tokenSvc,
		store:           store,
		gatewaySelector: o.GatewaySelector,
		serviceAccount:  o.ServiceAccount,
		jwtService:      o.JWT,
		publisher:       o.Publisher,
	}
}

func (s *AuthGuestService) CreateToken(ctx context.Context, shareID string) (*token.Token, error) {
	tok, err := s.tokenSvc.Generate(shareID)
	if err != nil {
		return nil, err
	}

	if err := s.store.Add(storage.Record{
		ShareID:     shareID,
		ShareIDHash: tok.ShareIDHash,
		SecretHash:  tok.SecretHash(),
		Expiry:      time.Now().Add(guestLinkTokenTTL),
		Redeemed:    false,
	}); err != nil {
		return nil, err
	}

	return tok, nil
}

// Redeem validates a token and its share and exchanges them for a session token
// and the share id.
func (s *AuthGuestService) Redeem(ctx context.Context, tokenString string) (*RedeemResponse, error) {
	rec, err := s.verifyToken(tokenString)
	if err != nil {
		return nil, err
	}

	if _, err := s.validateShare(ctx, rec.ShareID); err != nil {
		return nil, err
	}

	if err := s.store.Redeem(rec.ShareIDHash); err != nil {
		if errors.Is(err, storage.ErrAlreadyRedeemed) {
			return nil, &RedeemError{ErrorType: ErrAlreadyRedeemed, ShareID: rec.ShareID}
		}
		return nil, err
	}

	sessionToken, err := s.jwtService.Sign(rec.ShareID)
	if err != nil {
		return nil, err
	}

	return &RedeemResponse{SessionToken: sessionToken, ShareID: rec.ShareID}, nil
}

// Renew generates a new guest link token and PIN for the given share.
func (s *AuthGuestService) Renew(ctx context.Context, shareID string) error {
	if _, err := s.store.Get(token.Hash(shareID)); err != nil {
		return &RenewError{ErrorType: err, ShareID: shareID}
	}

	if s.publisher == nil {
		return ErrEventsNotConfigured
	}

	share, err := s.validateShare(ctx, shareID)
	if err != nil {
		return err
	}

	tok, err := s.tokenSvc.Generate(shareID)
	if err != nil {
		return err
	}

	pinStr, err := pin.Generate()
	if err != nil {
		return err
	}

	pinHash, err := pin.Hash(pinStr)
	if err != nil {
		return err
	}

	now := time.Now()

	if err := s.store.Replace(storage.Record{
		ShareID:     shareID,
		ShareIDHash: tok.ShareIDHash,
		SecretHash:  tok.SecretHash(),
		PinHash:     pinHash,
		Expiry:      now.Add(guestLinkTokenTTL),
		PinExpiry:   now.Add(guestPinTTL),
		Redeemed:    false,
	}); err != nil {
		return err
	}

	return revaevents.Publish(ctx, s.publisher, events.GuestTokenRenewed{
		ShareID:      share.GetId(),
		Sharer:       share.GetCreator(),
		GranteeEmail: share.GetGrantee().GetUserId().GetOpaqueId(),
		ItemID:       share.GetResourceId(),
		Token:        tok.String(),
		Pin:          pinStr,
		Timestamp:    now,
	})
}

// CleanupShare removes a share's token record from storage. Missing records are ignored.
func (s *AuthGuestService) CleanupShare(shareID string) error {
	shareIDHash := token.Hash(shareID)
	err := s.store.Remove(shareIDHash)
	if err != nil && err != storage.ErrNotFound {
		return err
	}

	return nil
}

// VerifyToken validates a token and returns its stored record.
//
// Until the secret has been verified, all failures are reported as
// token.ErrInvalidToken without a share id, so that a caller holding only part
// of a token learns neither the share id nor whether a record exists.
func (s *AuthGuestService) verifyToken(tokenString string) (*storage.Record, error) {
	tok, err := s.tokenSvc.Parse(tokenString)
	if err != nil {
		return nil, &RedeemError{ErrorType: token.ErrInvalidToken}
	}

	rec, err := s.store.Get(tok.ShareIDHash)
	switch {
	case errors.Is(err, storage.ErrNotFound), errors.Is(err, storage.ErrInvalidHash):
		return nil, &RedeemError{ErrorType: token.ErrInvalidToken}
	case err != nil:
		return nil, err
	}

	if err := s.tokenSvc.Verify(*tok, rec.SecretHash); err != nil {
		return nil, &RedeemError{ErrorType: token.ErrInvalidToken}
	}

	if !rec.Expiry.IsZero() && rec.Expiry.Before(time.Now()) {
		return nil, &RedeemError{ErrorType: ErrExpired, ShareID: rec.ShareID}
	}

	if rec.Redeemed {
		return nil, &RedeemError{ErrorType: ErrAlreadyRedeemed, ShareID: rec.ShareID}
	}

	return &rec, nil
}

// validateShare extracts the share information from the gateway and checks its existence and expiration.
func (s *AuthGuestService) validateShare(ctx context.Context, shareID string) (*collaboration.Share, error) {
	share, err := s.getShare(ctx, shareID)
	if err != nil {
		return nil, &RedeemError{ErrorType: err, ShareID: shareID}
	}

	if exp := utils.TSToTime(share.GetExpiration()); !exp.IsZero() && exp.Before(time.Now()) {
		return nil, &RedeemError{ErrorType: ErrShareExpired, ShareID: shareID}
	}

	return share, nil
}

// getShare fetches a share from the gateway.
func (s *AuthGuestService) getShare(ctx context.Context, shareID string) (*collaboration.Share, error) {
	gwc, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	ctx, err = utils.GetServiceUserContextWithContext(ctx, gwc, s.serviceAccount.ServiceAccountID, s.serviceAccount.ServiceAccountSecret)
	if err != nil {
		return nil, err
	}

	resp, err := gwc.GetShare(ctx, &collaboration.GetShareRequest{
		Ref: &collaboration.ShareReference{
			Spec: &collaboration.ShareReference_Id{
				Id: &collaboration.ShareId{
					OpaqueId: shareID,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	switch resp.GetStatus().GetCode() {
	case rpc.Code_CODE_OK:
	case rpc.Code_CODE_NOT_FOUND:
		return nil, ErrShareNotFound
	default:
		return nil, fmt.Errorf("could not get share %s: %s", shareID, resp.GetStatus().GetMessage())
	}

	share := resp.GetShare()
	if share == nil {
		return nil, ErrShareNotFound
	}

	return share, nil
}
