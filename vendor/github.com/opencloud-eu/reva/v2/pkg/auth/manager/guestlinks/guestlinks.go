// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

// Package guestlinks implements a Reva auth.Manager that authenticates
// OpenCloud guest-link sessions.
//
// It validates an OpenCloud guest-session JWT (issued by the OpenCloud
// proxy), re-validates the JWT's anchor collaborative share through the
// gateway GetShare API using a freshly minted service-account token, and
// returns the synthetic guest identity persisted as the share's grantee.
//
// See opencloud issue #3070 for the full design rationale.
package guestlinks

import (
	"context"
	"errors"
	"net/mail"
	"time"

	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-viper/mapstructure/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v2/pkg/auth"
	"github.com/opencloud-eu/reva/v2/pkg/auth/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/share"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

const (
	jwtLeeway        = 30 * time.Second
	maxShareIDLength = 512
)

func init() {
	registry.Register("guestlinks", New)
}

// guestClaims are the required custom JWT claims of a guest-session token.
type guestClaims struct {
	// the attribute in the JWT is called permissionId (for consistency with the
	// graph API), for us it really is the "shareId"
	ShareID string `json:"permissionId"`
	jwt.RegisteredClaims
}

// config holds the guestlinks auth manager configuration.
type config struct {
	GatewayAddr          string `mapstructure:"gateway_addr"`
	JWTSecret            string `mapstructure:"jwt_secret"`
	ServiceAccountID     string `mapstructure:"service_account_id"`
	ServiceAccountSecret string `mapstructure:"service_account_secret"`
}

func (c *config) validate() error {
	switch {
	case c.GatewayAddr == "":
		return errors.New("guestlinks: gateway_addr must not be empty")
	case c.JWTSecret == "":
		return errors.New("guestlinks: jwt_secret must not be empty")
	case c.ServiceAccountID == "":
		return errors.New("guestlinks: service_account_id must not be empty")
	case c.ServiceAccountSecret == "":
		return errors.New("guestlinks: service_account_secret must not be empty")
	}
	return nil
}

type manager struct {
	c   *config
	log *zerolog.Logger
}

func parseConfig(m map[string]any) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		return nil, errors.New("guestlinks: error decoding conf: " + err.Error())
	}
	return c, nil
}

// New returns a new guestlinks auth.Manager.
func New(m map[string]any, log *zerolog.Logger) (auth.Manager, error) {
	mgr := &manager{log: log}
	if err := mgr.Configure(m); err != nil {
		return nil, err
	}
	return mgr, nil
}

// Configure parses and validates the manager configuration.
func (m *manager) Configure(ml map[string]any) error {
	c, err := parseConfig(ml)
	if err != nil {
		return err
	}
	if err := c.validate(); err != nil {
		return err
	}
	m.c = c
	return nil
}

// Authenticate implements auth.Manager.
//
// clientID must be empty as the guestlinks credential is carried entirely in
// clientSecret, which must is the raw guest-session JWT.
func (m *manager) Authenticate(ctx context.Context, clientID, clientSecret string) (*userpb.User, map[string]*authpb.Scope, error) {
	if clientID != "" {
		m.logOutcome("invalid", "non-empty client_id")
		return nil, nil, errtypes.InvalidCredentials("non-empty client_id")
	}

	shareID, err := m.validateToken(clientSecret)
	if err != nil {
		if _, ok := errors.AsType[*errSessionExpired](err); ok {
			m.logOutcome("expired", "token expired")
		} else {
			m.logOutcome("invalid", "token validation failed")
		}
		return nil, nil, err
	}

	foundShare, err := m.lookupShare(ctx, shareID)
	if err != nil {
		return nil, nil, err
	}

	u, err := guestUserFromShare(foundShare)
	if err != nil {
		m.logOutcome("share-invalid", "share grantee invalid")
		return nil, nil, errtypes.InvalidCredentials("share grantee invalid")
	}

	sc, err := scope.AddOwnerScope(nil)
	if err != nil {
		m.logOutcome("internal", "error building scope")
		return nil, nil, errtypes.InternalError("guestlinks: error building token scope: " + err.Error())
	}

	m.logOutcome("success", "")
	return u, sc, nil
}

// validateToken validates the JWT contained in raw and returns the
// validated share_id.
//
// If the *only* validation failure is expiry, the returned error is a
// *errSessionExpired (still carrying the validated share_id), per the
// expiry-only classification rules. Any other validation failure is
// returned as a generic errtypes.InvalidCredentials with no detail.
func (m *manager) validateToken(raw string) (shareID string, err error) {
	if raw == "" {
		return "", errtypes.InvalidCredentials("empty token")
	}

	claims := &guestClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithoutClaimsValidation(),
	)
	_, parseErr := parser.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok || t.Method.Alg() != "HS256" {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(m.c.JWTSecret), nil
	})
	if parseErr != nil {
		return "", errtypes.InvalidCredentials("signature/algorithm/parse error")
	}

	if claims.ShareID == "" || len(claims.ShareID) > maxShareIDLength {
		return "", errtypes.InvalidCredentials("missing/malformed share_id")
	}
	if claims.IssuedAt == nil || claims.ExpiresAt == nil {
		return "", errtypes.InvalidCredentials("missing iat/exp")
	}

	now := time.Now()

	// iat in the future (beyond leeway) is structurally invalid.
	if claims.IssuedAt.After(now.Add(jwtLeeway)) {
		return "", errtypes.InvalidCredentials("invalid iat")
	}

	// Everything but expiry has been validated at this point. Now check
	// expiry last, so we can classify an expiry-only failure.
	if now.After(claims.ExpiresAt.Add(jwtLeeway)) {
		return claims.ShareID, newSessionExpiredError(claims.ShareID)
	}

	return claims.ShareID, nil
}

// Reads and re-validates the share associated with the authentication token
// using the service account.
func (m *manager) lookupShare(ctx context.Context, shareID string) (*collaboration.Share, error) {
	gwc, err := pool.GetGatewayServiceClient(m.c.GatewayAddr)
	if err != nil {
		m.logOutcome("unavailable", "error getting gateway client")
		return nil, errtypes.Unavailable("guestlinks: error getting gateway client: " + err.Error())
	}

	saCtx, err := utils.GetServiceUserContextWithContext(ctx, gwc, m.c.ServiceAccountID, m.c.ServiceAccountSecret)
	if err != nil {
		m.logOutcome("unavailable", "error minting service account token")
		return nil, errtypes.Unavailable("guestlinks: error authenticating service account: " + err.Error())
	}

	getShareRes, err := gwc.GetShare(saCtx, &collaboration.GetShareRequest{
		Ref: &collaboration.ShareReference{
			Spec: &collaboration.ShareReference_Id{
				Id: &collaboration.ShareId{OpaqueId: shareID},
			},
		},
	})
	if err != nil {
		m.logOutcome("unavailable", "error calling GetShare")
		return nil, errtypes.Unavailable("guestlinks: error calling GetShare: " + err.Error())
	}

	switch getShareRes.GetStatus().GetCode() {
	case rpc.Code_CODE_OK:
		// fall through
	case rpc.Code_CODE_NOT_FOUND, rpc.Code_CODE_PERMISSION_DENIED, rpc.Code_CODE_UNAUTHENTICATED:
		m.logOutcome("share-invalid", "share not found/inaccessible")
		return nil, errtypes.InvalidCredentials("share not found/inaccessible")
	case rpc.Code_CODE_UNAVAILABLE:
		m.logOutcome("unavailable", "share provider unavailable")
		return nil, errtypes.Unavailable("guestlinks: share provider unavailable")
	default:
		m.logOutcome("internal", "unexpected GetShare status")
		return nil, errtypes.InternalError("guestlinks: unexpected GetShare status: " + getShareRes.GetStatus().GetCode().String())
	}

	foundShare := getShareRes.GetShare()
	if foundShare == nil {
		m.logOutcome("share-invalid", "nil share")
		return nil, errtypes.InvalidCredentials("nil share")
	}
	if share.IsExpired(foundShare) {
		m.logOutcome("share-invalid", "share expired")
		return nil, errtypes.InvalidCredentials("share expired")
	}

	return foundShare, nil
}

// guestUserFromShare builds the synthetic guest user from the persisted
// share grantee. It never trusts JWT claims for identity data.
func guestUserFromShare(s *collaboration.Share) (*userpb.User, error) {
	grantee := s.GetGrantee()
	if grantee.GetType() != storageprovider.GranteeType_GRANTEE_TYPE_USER {
		return nil, errors.New("guestlinks: grantee is not a user")
	}

	uid := grantee.GetUserId()
	if uid == nil || uid.GetOpaqueId() == "" {
		return nil, errors.New("guestlinks: incomplete grantee user id")
	}
	if uid.GetType() != userpb.UserType_USER_TYPE_GUEST {
		return nil, errors.New("guestlinks: grantee is not a guest user")
	}

	addr, err := mail.ParseAddress(uid.GetOpaqueId())
	if err != nil || addr.Name != "" || addr.Address != uid.GetOpaqueId() {
		return nil, errors.New("guestlinks: grantee opaque id is not a bare email address")
	}
	email := uid.GetOpaqueId()

	u := &userpb.User{
		Id: &userpb.UserId{
			Idp:                uid.GetIdp(),
			OpaqueId:           uid.GetOpaqueId(),
			Type:               userpb.UserType_USER_TYPE_GUEST,
			TenantId:           uid.GetTenantId(),
			ExternalIdentities: uid.GetExternalIdentities(),
		},
		Username:    email,
		DisplayName: email,
	}

	// TODO: OpenCloud usually, attaches a user role to the token, how can we do that here?

	return u, nil
}

func (m *manager) logOutcome(outcome, detail string) {
	if m.log == nil {
		return
	}
	ev := m.log.Debug()
	if outcome != "success" {
		ev = m.log.Info()
	}
	ev.Str("outcome", outcome).Str("detail", detail).Msg("guestlinks: authenticate outcome")
}
