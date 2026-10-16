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
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/guard"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/jwt"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/pin"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/storage"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/token"
	revaevents "github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

const (
	guestLinkTokenTTL = 30 * time.Minute
	renewMaxAttempts  = 3
)

// SessionResponse is the result of a successful authentication: a session token,
// its expiry and the share id it grants access to.
type SessionResponse struct {
	SessionToken string
	ExpiresAt    time.Time
	ShareID      string
}

// AuthGuest is the domain service used by the transport and event layers.
type AuthGuest interface {
	CreateToken(ctx context.Context, shareID string) (*token.Token, error)
	VerifyToken(ctx context.Context, tokenString string) (*SessionResponse, error)
	Renew(ctx context.Context, shareID, sessionToken string) (string, error)
	VerifyPin(ctx context.Context, shareID, pinValue, nonce string) (*SessionResponse, error)
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
	pinAndNonceTTL  time.Duration
	guards          guards
}

// guards bundles the rate limiters and the nonce protecting the guest
// endpoints. Their policies are fixed at construction from the service config.
type guards struct {
	backoffLimiter guard.BackoffLimiter
	quotaLimiter   guard.QuotaLimiter
	nonce          guard.Nonce
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
		pinAndNonceTTL:  o.PinAndNonceTTL,
		guards: guards{
			backoffLimiter: guard.NewBackoffLimiter(o.BackoffLimiter),
			quotaLimiter:   guard.NewQuotaLimiter(o.QuotaLimiter),
			nonce:          guard.NewNonce(o.PinAndNonceTTL),
		},
	}
}

func (s *AuthGuestService) CreateToken(ctx context.Context, shareID string) (*token.Token, error) {
	tok, err := s.tokenSvc.Generate(shareID)
	if err != nil {
		return nil, err
	}

	rec := storage.NewRecord(shareID, tok.ShareIDHash, tok.SecretHash(), time.Now().Add(guestLinkTokenTTL))
	if err := s.store.Add(rec); err != nil {
		return nil, err
	}

	return tok, nil
}

// VerifyToken validates a token and its share and exchanges them for a session
// token and the share id.
func (s *AuthGuestService) VerifyToken(ctx context.Context, tokenString string) (*SessionResponse, error) {
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

	sessionToken, expiresAt, err := s.jwtService.Sign(rec.ShareID)
	if err != nil {
		return nil, err
	}

	return &SessionResponse{SessionToken: sessionToken, ExpiresAt: expiresAt, ShareID: rec.ShareID}, nil
}

// Renew renews the guest link token for shareID. When the request is
// authorized by a valid, unexpired session token, a new PIN and nonce are
// also issued; otherwise the token is only rotated (rate limited) and no
// PIN or nonce is issued.
func (s *AuthGuestService) Renew(ctx context.Context, shareID, sessionToken string) (string, error) {
	if s.publisher == nil {
		return "", ErrEventsNotConfigured
	}

	hash := token.Hash(shareID)
	now := time.Now()

	current, err := s.store.Get(hash)
	if err != nil {
		return "", &GuestError{ErrorType: err, ShareID: shareID}
	}

	validSession := false
	if sessionToken != "" {
		sessionShareID, err := s.jwtService.Verify(sessionToken)
		validSession = err == nil && sessionShareID == shareID
	}

	if !validSession {
		// Reject an active cooldown before doing the (remote) share validation.
		if wait := s.guards.quotaLimiter.RetryAfter(&current.QuotaLimiterState, now); wait > 0 {
			return "", &GuestError{ErrorType: ErrTooManyAttempts, ShareID: shareID, RetryAfter: wait}
		}
	}

	share, err := s.validateShare(ctx, shareID)
	if err != nil {
		return "", err
	}

	tok, err := s.tokenSvc.Generate(shareID)
	if err != nil {
		return "", err
	}

	pinStr, err := pin.Generate()
	if err != nil {
		return "", err
	}

	pinHash, err := pin.Hash(pinStr)
	if err != nil {
		return "", err
	}

	nonce, seen, updated, err := s.rotateGuestToken(hash, tok, pinHash, now, validSession)
	if err != nil {
		var ge *GuestError
		if !errors.As(err, &ge) {
			return "", &GuestError{ErrorType: err, ShareID: shareID}
		}
		return "", ge
	}

	ev := events.GuestTokenRenewed{
		ShareID:      share.GetId(),
		Sharer:       share.GetCreator(),
		GranteeEmail: share.GetGrantee().GetUserId().GetOpaqueId(),
		ItemID:       share.GetResourceId(),
		Token:        tok.String(),
		Timestamp:    now,
	}

	if validSession {
		ev.Pin = pinStr
	}

	if err := revaevents.Publish(ctx, s.publisher, ev); err != nil {
		if _, rbErr := s.store.UpdateFrom(updated, func(rec *storage.Record) error {
			*rec = seen
			return nil
		}); rbErr != nil {
			return "", fmt.Errorf("publishing guest token renewed event failed: %w (rollback failed: %v)", err, rbErr)
		}
		return "", err
	}

	return nonce, nil
}

// VerifyPin validates a PIN and the browser-bound nonce and exchanges
// them for a session token and the share id.
func (s *AuthGuestService) VerifyPin(ctx context.Context, shareID, pinValue, nonce string) (*SessionResponse, error) {
	hash := token.Hash(shareID)

	if err := s.precheckPin(shareID, nonce); err != nil {
		return nil, err
	}

	if _, err := s.validateShare(ctx, shareID); err != nil {
		return nil, err
	}

	// verifyPinAttempt reports a wrong PIN as ok=false with a nil error, so the
	// failed-attempt state is persisted by Update; real errors abort the write.
	var denied bool
	err := s.store.Update(hash, func(rec *storage.Record) error {
		ok, err := s.verifyPinAttempt(rec, pinValue, nonce)
		if err != nil {
			return err
		}
		denied = !ok
		return nil
	})
	if err != nil {
		var ge *GuestError
		if !errors.As(err, &ge) {
			return nil, &GuestError{ErrorType: err, ShareID: shareID}
		}
		return nil, ge
	}
	if denied {
		return nil, &GuestError{ErrorType: ErrPinInvalid, ShareID: shareID}
	}

	sessionToken, expiresAt, err := s.jwtService.Sign(shareID)
	if err != nil {
		return nil, err
	}

	return &SessionResponse{SessionToken: sessionToken, ExpiresAt: expiresAt, ShareID: shareID}, nil
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

// rotateGuestToken replaces the guest token's secret and expiry under the
// record lock. When sessionValid, the PIN and nonce are replaced as well;
// otherwise the quota limiter state is recorded. When the record is modified
// concurrently it re-reads
// and re-applies the rotation, up to renewMaxAttempts. It returns the issued
// nonce, the record state seen before the successful update (for rollback) and
// the updated record.
func (s *AuthGuestService) rotateGuestToken(hash string, tok *token.Token, pinHash string, now time.Time, sessionValid bool) (string, storage.Record, storage.Record, error) {
	var nonce string
	for attempt := 1; ; attempt++ {
		rec, err := s.store.Get(hash)
		if err != nil {
			return "", storage.Record{}, storage.Record{}, err
		}

		updated, err := s.store.UpdateFrom(*rec, func(record *storage.Record) error {
			if !sessionValid {
				if wait := s.guards.quotaLimiter.RetryAfter(&record.QuotaLimiterState, now); wait > 0 {
					return &GuestError{ErrorType: ErrTooManyAttempts, ShareID: record.ShareID, RetryAfter: wait}
				}
				s.guards.quotaLimiter.Record(&record.QuotaLimiterState, now)
			} else {
				n, err := s.guards.nonce.Issue(&record.NonceState, now)
				if err != nil {
					return err
				}
				nonce = n
				record.BackoffLimiterState = guard.BackoffLimiterState{}

				record.PinHash = pinHash
				record.PinExpiry = now.Add(s.pinAndNonceTTL)
			}

			record.SecretHash = tok.SecretHash()
			record.Expiry = now.Add(guestLinkTokenTTL)

			record.Redeemed = false
			return nil
		})
		if err == nil {
			return nonce, *rec, *updated, nil
		}
		if !errors.Is(err, storage.ErrConflict) || attempt >= renewMaxAttempts {
			return "", storage.Record{}, storage.Record{}, err
		}
	}
}

// checkPinRequest verifies the browser-bound nonce and then the PIN
// failed-attempt backoff, returning the error to surface to the client. The
// nonce is checked first so a missing or mismatched nonce is not hidden behind
// rate limiting.
func (s *AuthGuestService) checkPinRequest(rec *storage.Record, nonce string, now time.Time) *GuestError {
	if err := s.guards.nonce.Verify(&rec.NonceState, nonce, now); err != nil {
		return &GuestError{ErrorType: nonceError(err), ShareID: rec.ShareID}
	}
	if wait := s.guards.backoffLimiter.RetryAfter(&rec.BackoffLimiterState, now); wait > 0 {
		return &GuestError{ErrorType: ErrTooManyAttempts, ShareID: rec.ShareID, RetryAfter: wait}
	}
	return nil
}

// precheckPin cheaply rejects a bad or blocked request before the (remote)
// share validation. The same checks run again under the record lock in
// verifyPinAttempt, which is what actually consumes the attempt.
func (s *AuthGuestService) precheckPin(shareID, nonce string) error {
	rec, err := s.store.Get(token.Hash(shareID))
	if err != nil {
		return &GuestError{ErrorType: err, ShareID: shareID}
	}

	ge := s.checkPinRequest(rec, nonce, time.Now())
	if ge != nil {
		return ge
	}
	return nil
}

// verifyPinAttempt verifies the PIN under the caller's lock and updates the
// record: a success clears the credential, a wrong PIN records the attempt and
// its backoff. Gate and expiry failures leave the record untouched and return an
// error.
func (s *AuthGuestService) verifyPinAttempt(rec *storage.Record, pinValue, nonce string) (ok bool, err error) {
	now := time.Now()

	// The pre-check surfaced Retry-After already; this authoritative re-check
	// under the lock returns the GuestError so the wait still reaches the
	// transport.
	ge := s.checkPinRequest(rec, nonce, now)
	if ge != nil {
		return false, ge
	}

	switch {
	case rec.PinHash == "":
		return false, ErrPinInvalid
	case !rec.PinExpiry.IsZero() && rec.PinExpiry.Before(now):
		return false, ErrPinExpired
	}

	match, _ := pin.Verify(pinValue, rec.PinHash)
	if !match && !s.guards.backoffLimiter.Record(&rec.BackoffLimiterState, now) {
		// Wrong PIN, but attempts remain: keep the credential and its nonce.
		return false, nil
	}

	// Either the PIN was correct (credential consumed) or the attempts are
	// exhausted (credential invalidated); both spend the single-use nonce.
	rec.PinHash = ""
	rec.PinExpiry = time.Time{}
	s.guards.nonce.Clear(&rec.NonceState)
	s.guards.backoffLimiter.Reset(&rec.BackoffLimiterState)
	return match, nil
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
