package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	goidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"golang.org/x/oauth2"
)

// OIDCClient used to mock the oidc client during tests
type OIDCClient interface {
	UserInfo(ctx context.Context, ts oauth2.TokenSource) (*UserInfo, error)
	VerifyAccessToken(ctx context.Context, token string) (RegClaimsWithSID, jwt.MapClaims, error)
	VerifyLogoutToken(ctx context.Context, token string) (*LogoutToken, error)
}

// KeySet is a set of public JSON Web Keys that can be used to validate the signature
// of JSON web tokens. This is expected to be backed by a remote key set through
// provider metadata discovery or an in-memory set of keys delivered out-of-band.
type KeySet interface {
	// VerifySignature parses the JSON web token, verifies the signature, and returns
	// the raw payload. Header and claim fields are validated by other parts of the
	// package. For example, the KeySet does not need to check values such as signature
	// algorithm, issuer, and audience since the IDTokenVerifier validates these values
	// independently.
	//
	// If VerifySignature makes HTTP requests to verify the token, it's expected to
	// use any HTTP client associated with the context through ClientContext.
	VerifySignature(ctx context.Context, jwt string) (payload []byte, err error)
}

type RegClaimsWithSID struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

type oidcClient struct {
	// Logger to use for logging, must be set
	Logger log.Logger

	issuer                  string
	provider                *ProviderMetadata
	providerLock            *sync.Mutex
	skipIssuerValidation    bool
	accessTokenVerifyMethod string
	remoteKeySet            KeySet
	algorithms              []string

	JWKSOptions config.JWKS
	JWKS        *keyfunc.JWKS
	jwksLock    *sync.Mutex

	httpClient *http.Client
}

// _supportedAlgorithms is a list of algorithms explicitly supported by this
// package. If a provider supports other algorithms, such as HS256 or none,
// those values won't be passed to the IDTokenVerifier.
var _supportedAlgorithms = map[string]bool{
	RS256: true,
	RS384: true,
	RS512: true,
	ES256: true,
	ES384: true,
	ES512: true,
	PS256: true,
	PS384: true,
	PS512: true,
}

// NewOIDCClient returns an OIDClient instance for the given issuer
func NewOIDCClient(opts ...Option) OIDCClient {
	options := newOptions(opts...)

	return &oidcClient{
		Logger:                  options.Logger,
		issuer:                  options.OIDCIssuer,
		httpClient:              options.HTTPClient,
		accessTokenVerifyMethod: options.AccessTokenVerifyMethod,
		JWKSOptions:             options.JWKSOptions, // TODO I don't like that we pass down config options ...
		JWKS:                    options.JWKS,
		providerLock:            &sync.Mutex{},
		jwksLock:                &sync.Mutex{},
		remoteKeySet:            options.KeySet,
		provider:                options.ProviderMetadata,
	}
}

func (c *oidcClient) lookupWellKnownOpenidConfiguration(ctx context.Context) error {
	c.providerLock.Lock()
	defer c.providerLock.Unlock()
	if c.provider == nil {
		wellKnown := strings.TrimSuffix(c.issuer, "/") + wellknownPath
		req, err := http.NewRequest("GET", wellKnown, nil)
		if err != nil {
			return err
		}
		resp, err := c.httpClient.Do(req.WithContext(ctx))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read response body: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("%s: %s", resp.Status, body)
		}

		var p ProviderMetadata
		err = unmarshalResp(resp, body, &p)
		if err != nil {
			return fmt.Errorf("oidc: failed to decode provider discovery object: %v", err)
		}

		if !c.skipIssuerValidation && p.Issuer != c.issuer {
			return fmt.Errorf("oidc: issuer did not match the issuer returned by provider, expected %q got %q", c.issuer, p.Issuer)
		}
		var algs []string
		for _, a := range p.IDTokenSigningAlgValuesSupported {
			if _supportedAlgorithms[a] {
				algs = append(algs, a)
			}
		}
		c.provider = &p
		c.algorithms = algs
		c.remoteKeySet = goidc.NewRemoteKeySet(goidc.ClientContext(ctx, c.httpClient), p.JwksURI)
	}
	return nil
}

func (c *oidcClient) getKeyfunc() *keyfunc.JWKS {
	c.jwksLock.Lock()
	defer c.jwksLock.Unlock()
	if c.JWKS == nil {
		var err error
		c.Logger.Debug().Str("jwks", c.provider.JwksURI).Msg("discovered jwks endpoint")
		options := keyfunc.Options{
			Client: c.httpClient,
			RefreshErrorHandler: func(err error) {
				c.Logger.Error().Err(err).Msg("There was an error with the jwt.Keyfunc")
			},
			RefreshInterval:   time.Minute * time.Duration(c.JWKSOptions.RefreshInterval),
			RefreshRateLimit:  time.Second * time.Duration(c.JWKSOptions.RefreshRateLimit),
			RefreshTimeout:    time.Second * time.Duration(c.JWKSOptions.RefreshTimeout),
			RefreshUnknownKID: c.JWKSOptions.RefreshUnknownKID,
		}
		c.JWKS, err = keyfunc.Get(c.provider.JwksURI, options)
		if err != nil {
			c.JWKS = nil
			c.Logger.Error().Err(err).Msg("Failed to create JWKS from resource at the given URL.")
			return nil
		}
	}
	return c.JWKS
}

type stringAsBool bool

// Claims unmarshals the raw JSON string into a bool.
func (sb *stringAsBool) UnmarshalJSON(b []byte) error {
	v, err := strconv.ParseBool(string(b))
	if err != nil {
		return err
	}
	*sb = stringAsBool(v)
	return nil
}

// UserInfo represents the OpenID Connect userinfo claims.
type UserInfo struct {
	Subject       string `json:"sub"`
	Profile       string `json:"profile"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`

	claims []byte
}

type userInfoRaw struct {
	Subject string `json:"sub"`
	Profile string `json:"profile"`
	Email   string `json:"email"`
	// Handle providers that return email_verified as a string
	// https://forums.aws.amazon.com/thread.jspa?messageID=949441&#949441 and
	// https://discuss.elastic.co/t/openid-error-after-authenticating-against-aws-cognito/206018/11
	EmailVerified stringAsBool `json:"email_verified"`
}

// Claims unmarshals the raw JSON object claims into the provided object.
func (u *UserInfo) Claims(v any) error {
	if u.claims == nil {
		return errors.New("oidc: claims not set")
	}
	return json.Unmarshal(u.claims, v)
}

// UserInfo retrieves the userinfo from a Token
func (c *oidcClient) UserInfo(ctx context.Context, tokenSource oauth2.TokenSource) (*UserInfo, error) {
	if err := c.lookupWellKnownOpenidConfiguration(ctx); err != nil {
		return nil, err
	}

	if c.provider.UserinfoEndpoint == "" {
		return nil, errors.New("oidc: user info endpoint is not supported by this provider")
	}

	req, err := http.NewRequest("GET", c.provider.UserinfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("oidc: create GET request: %v", err)
	}

	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("oidc: get access token: %v", err)
	}
	token.SetAuthHeader(req)

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", resp.Status, body)
	}

	ct := resp.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(ct)
	if err == nil && mediaType == "application/jwt" {
		payload, err := c.remoteKeySet.VerifySignature(goidc.ClientContext(ctx, c.httpClient), string(body))
		if err != nil {
			return nil, fmt.Errorf("oidc: invalid userinfo jwt signature %v", err)
		}
		body = payload
	}

	var userInfo userInfoRaw
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("oidc: failed to decode userinfo: %v", err)
	}
	return &UserInfo{
		Subject:       userInfo.Subject,
		Profile:       userInfo.Profile,
		Email:         userInfo.Email,
		EmailVerified: bool(userInfo.EmailVerified),
		claims:        body,
	}, nil
}

func (c *oidcClient) VerifyAccessToken(ctx context.Context, token string) (RegClaimsWithSID, jwt.MapClaims, error) {
	if err := c.lookupWellKnownOpenidConfiguration(ctx); err != nil {
		return RegClaimsWithSID{}, jwt.MapClaims{}, err
	}
	switch c.accessTokenVerifyMethod {
	case config.AccessTokenVerificationJWT:
		return c.verifyAccessTokenJWT(token)
	case config.AccessTokenVerificationNone:
		c.Logger.Debug().Msg("Access Token verification disabled")
		return RegClaimsWithSID{}, jwt.MapClaims{}, nil
	default:
		c.Logger.Error().Str("access_token_verify_method", c.accessTokenVerifyMethod).Msg("Unknown Access Token verification setting")
		return RegClaimsWithSID{}, jwt.MapClaims{}, errors.New("unknown Access Token Verification method")
	}
}

// verifyAccessTokenJWT tries to parse and verify the access token as a JWT.
func (c *oidcClient) verifyAccessTokenJWT(token string) (RegClaimsWithSID, jwt.MapClaims, error) {
	var claims RegClaimsWithSID
	mapClaims := jwt.MapClaims{}
	jwks := c.getKeyfunc()
	if jwks == nil {
		return claims, mapClaims, errors.New("error initializing jwks keyfunc")
	}

	issuer := c.issuer
	if c.provider.AccessTokenIssuer != "" {
		// AD FS .well-known/openid-configuration has an optional `access_token_issuer` which takes precedence over `issuer`
		// See https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-oidce/586de7dd-3385-47c7-93a2-935d9e90441c
		issuer = c.provider.AccessTokenIssuer
	}

	_, err := jwt.ParseWithClaims(token, &claims, jwks.Keyfunc, jwt.WithIssuer(issuer))
	if err != nil {
		return claims, mapClaims, err
	}
	_, _, err = new(jwt.Parser).ParseUnverified(token, mapClaims)
	// TODO: decode mapClaims to sth readable
	c.Logger.Debug().Interface("access token", &claims).Msg("parsed access token")
	if err != nil {
		c.Logger.Info().Err(err).Msg("Failed to parse/verify the access token.")
		return claims, mapClaims, err
	}

	return claims, mapClaims, nil
}

func (c *oidcClient) VerifyLogoutToken(ctx context.Context, rawToken string) (*LogoutToken, error) {
	var claims LogoutToken
	if err := c.lookupWellKnownOpenidConfiguration(ctx); err != nil {
		return nil, err
	}
	jwks := c.getKeyfunc()
	if jwks == nil {
		return nil, errors.New("error initializing jwks keyfunc")
	}

	// From the backchannel-logout spec: Like ID Tokens, selection of the
	// algorithm used is governed by the id_token_signing_alg_values_supported
	// Discovery parameter and the id_token_signed_response_alg Registration
	// parameter when they are used; otherwise, the value SHOULD be the default
	// of RS256
	supportedSigAlgs := c.algorithms
	if len(supportedSigAlgs) == 0 {
		supportedSigAlgs = []string{RS256}
	}

	_, err := jwt.ParseWithClaims(rawToken, &claims, jwks.Keyfunc, jwt.WithValidMethods(supportedSigAlgs), jwt.WithIssuer(c.issuer))
	if err != nil {
		c.Logger.Debug().Err(err).Msg("Failed to parse logout token")
		return nil, err
	}
	// Basic token validation has happened in ParseWithClaims (signature,
	// issuer, audience, ...). Now for some logout token specific checks.
	// 1. Verify that the Logout Token contains a sub Claim, a sid Claim, or both.
	if claims.Subject == "" && claims.SessionId == "" {
		return nil, fmt.Errorf("oidc: logout token must contain either sub or sid and MAY contain both")
	}
	// 2. Verify that the Logout Token contains an events Claim whose value is JSON object containing the member name http://schemas.openid.net/event/backchannel-logout.
	if claims.Events.Event == nil {
		return nil, fmt.Errorf("oidc: logout token must contain logout event")
	}
	// 3. Verify that the Logout Token does not contain a nonce Claim.
	if claims.Nonce != nil {
		return nil, fmt.Errorf("oidc: nonce on logout token MUST NOT be present")
	}

	return &claims, nil
}

func unmarshalResp(r *http.Response, body []byte, v any) error {
	err := json.Unmarshal(body, &v)
	if err == nil {
		return nil
	}
	ct := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(ct)
	if err == nil && mediaType == "application/json" {
		return fmt.Errorf("got Content-Type = application/json, but could not unmarshal as JSON: %v", err)
	}
	return fmt.Errorf("expected Content-Type = application/json, got %q: %v", ct, err)
}
