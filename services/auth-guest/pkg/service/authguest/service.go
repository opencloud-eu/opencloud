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
var ErrPinInvalid = errors.New("pin invalid")
var ErrPinExpired = errors.New("pin expired")
var ErrPermissionMismatch = errors.New("permissionId does not match the provided credentials")

const guestLinkTokenTTL = 30 * time.Minute
const guestPinTTL = 30 * time.Minute

// GuestError wraps an auth-guest failure together with the share id. The HTTP
// transport inspects ErrorType to choose a status code and message.
type GuestError struct {
	ErrorType error
	ShareID   string
}

func (e *GuestError) Error() string { return e.ErrorType.Error() }

// SessionResponse is the result of a successful authentication: a session token
// and the share id it grants access to.
type SessionResponse struct {
	SessionToken string
	ShareID      string
}

// AuthGuest is the domain service used by the transport and event layers.
type AuthGuest interface {
	CreateToken(ctx context.Context, shareID string) (*token.Token, error)
	Redeem(ctx context.Context, tokenString string) (*SessionResponse, error)
	Renew(ctx context.Context, shareID, tokenString, sessionToken string) error
	VerifyPin(ctx context.Context, shareID, pinValue string) (*SessionResponse, error)
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
func (s *AuthGuestService) Redeem(ctx context.Context, tokenString string) (*SessionResponse, error) {
	rec, err := s.verifyToken(tokenString)
	if err != nil {
		return nil, err
	}

	if _, err := s.validateShare(ctx, rec.ShareID); err != nil {
		return nil, err
	}

	if _, err := s.store.UpdateFrom(*rec, func(r *storage.Record) error {
		if r.Redeemed {
			return ErrAlreadyRedeemed
		}
		r.Redeemed = true
		return nil
	}); err != nil {
		if errors.Is(err, storage.ErrConflict) {
			err = ErrAlreadyRedeemed
		}
		return nil, &GuestError{ErrorType: err, ShareID: rec.ShareID}
	}

	sessionToken, err := s.jwtService.Sign(rec.ShareID)
	if err != nil {
		return nil, err
	}

	return &SessionResponse{SessionToken: sessionToken, ShareID: rec.ShareID}, nil
}

// Renew generates a new guest link token and PIN for shareID, authorized by the
// previous link token or a (possibly expired) session token.
func (s *AuthGuestService) Renew(ctx context.Context, shareID, tokenString, sessionToken string) error {
	if s.publisher == nil {
		return ErrEventsNotConfigured
	}

	if err := s.verifyRenewCredentials(shareID, tokenString, sessionToken); err != nil {
		return err
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

	current, err := s.store.Get(token.Hash(shareID))
	if err != nil {
		return &GuestError{ErrorType: err, ShareID: shareID}
	}

	updated, err := s.store.UpdateFrom(*current, func(rec *storage.Record) error {
		rec.SecretHash = tok.SecretHash()
		rec.PinHash = pinHash
		rec.Expiry = now.Add(guestLinkTokenTTL)
		rec.PinExpiry = now.Add(guestPinTTL)
		rec.Redeemed = false
		return nil
	})
	if err != nil {
		if errors.Is(err, storage.ErrConflict) {
			err = ErrAlreadyRedeemed
		}
		return &GuestError{ErrorType: err, ShareID: shareID}
	}

	if err := revaevents.Publish(ctx, s.publisher, events.GuestTokenRenewed{
		ShareID:      share.GetId(),
		Sharer:       share.GetCreator(),
		GranteeEmail: share.GetGrantee().GetUserId().GetOpaqueId(),
		ItemID:       share.GetResourceId(),
		Token:        tok.String(),
		Pin:          pinStr,
		Timestamp:    now,
	}); err != nil {
		if _, rbErr := s.store.UpdateFrom(*updated, func(rec *storage.Record) error {
			*rec = *current
			return nil
		}); rbErr != nil {
			return fmt.Errorf("publishing guest token renewed event failed: %w (rollback failed: %v)", err, rbErr)
		}
		return err
	}

	return nil
}

// VerifyPin validates a PIN and its share and exchanges them for a session token
// and the share id.
func (s *AuthGuestService) VerifyPin(ctx context.Context, shareID, pinValue string) (*SessionResponse, error) {
	if _, err := s.validateShare(ctx, shareID); err != nil {
		return nil, err
	}

	err := s.store.Update(token.Hash(shareID), func(rec *storage.Record) error {
		if rec.PinHash == "" {
			return ErrPinInvalid
		}

		if !rec.PinExpiry.IsZero() && rec.PinExpiry.Before(time.Now()) {
			return ErrPinExpired
		}

		ok, err := pin.Verify(pinValue, rec.PinHash)
		if err != nil || !ok {
			return ErrPinInvalid
		}

		rec.PinHash = ""
		rec.PinExpiry = time.Time{}

		return nil
	})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			// TODO: should a missing record map to 404 tokenNotFound instead of 401 pinInvalid?
			err = ErrPinInvalid
		}
		return nil, &GuestError{ErrorType: err, ShareID: shareID}
	}

	sessionToken, err := s.jwtService.Sign(shareID)
	if err != nil {
		return nil, err
	}

	return &SessionResponse{SessionToken: sessionToken, ShareID: shareID}, nil
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

// verifyRenewCredentials checks that the previous link token or a (possibly
// expired) session token belongs to shareID.
func (s *AuthGuestService) verifyRenewCredentials(shareID, tokenString, sessionToken string) error {
	switch {
	case tokenString != "":
		return s.verifyRenewToken(shareID, tokenString)
	case sessionToken != "":
		if s.jwtService == nil {
			return &GuestError{ErrorType: jwt.ErrInvalidSession, ShareID: shareID}
		}
		sessionShareID, err := s.jwtService.Verify(sessionToken)
		if err != nil {
			return &GuestError{ErrorType: err, ShareID: shareID}
		}
		if sessionShareID != shareID {
			return &GuestError{ErrorType: ErrPermissionMismatch, ShareID: shareID}
		}
		return nil
	default:
		return &GuestError{ErrorType: jwt.ErrInvalidSession, ShareID: shareID}
	}
}

// verifyRenewToken checks that the link token belongs to shareID and matches
// the stored record, ignoring its expiry and redeemed state.
func (s *AuthGuestService) verifyRenewToken(shareID, tokenString string) error {
	tok, err := s.tokenSvc.Parse(tokenString)
	if err != nil {
		return &GuestError{ErrorType: err, ShareID: shareID}
	}

	if tok.ShareIDHash != token.Hash(shareID) {
		return &GuestError{ErrorType: ErrPermissionMismatch, ShareID: shareID}
	}

	rec, err := s.store.Get(tok.ShareIDHash)
	if err != nil {
		return &GuestError{ErrorType: err, ShareID: shareID}
	}

	if err := s.tokenSvc.Verify(*tok, rec.SecretHash); err != nil {
		return &GuestError{ErrorType: err, ShareID: shareID}
	}

	return nil
}

// VerifyToken validates a token and returns its stored record.
func (s *AuthGuestService) verifyToken(tokenString string) (*storage.Record, error) {
	tok, err := s.tokenSvc.Parse(tokenString)
	if err != nil {
		return nil, &GuestError{ErrorType: err}
	}

	rec, err := s.store.Get(tok.ShareIDHash)
	if err != nil {
		return nil, &GuestError{ErrorType: err}
	}

	if err := s.tokenSvc.Verify(*tok, rec.SecretHash); err != nil {
		return nil, &GuestError{ErrorType: err, ShareID: rec.ShareID}
	}

	if !rec.Expiry.IsZero() && rec.Expiry.Before(time.Now()) {
		return nil, &GuestError{ErrorType: ErrExpired, ShareID: rec.ShareID}
	}

	if rec.Redeemed {
		return nil, &GuestError{ErrorType: ErrAlreadyRedeemed, ShareID: rec.ShareID}
	}

	return rec, nil
}

// validateShare extracts the share information from the gateway and checks its existence and expiration.
func (s *AuthGuestService) validateShare(ctx context.Context, shareID string) (*collaboration.Share, error) {
	share, err := s.getShare(ctx, shareID)
	if err != nil {
		return nil, &GuestError{ErrorType: err, ShareID: shareID}
	}

	if exp := utils.TSToTime(share.GetExpiration()); !exp.IsZero() && exp.Before(time.Now()) {
		return nil, &GuestError{ErrorType: ErrShareExpired, ShareID: shareID}
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
