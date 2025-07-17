package middleware

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/userroles"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/signedurl"
	microstore "go-micro.dev/v4/store"
	"golang.org/x/crypto/pbkdf2"
)

const (
	_paramOCSignature  = "OC-Signature"
	_paramOCCredential = "OC-Credential" // #nosec G101
	_paramOCDate       = "OC-Date"
	_paramOCExpires    = "OC-Expires"
	_paramOCVerb       = "OC-Verb"
	_paramOCAlgo       = "OC-Algo"
	_paramOCJWTSig     = "oc-jwt-sig"
)

var (
	_requiredParams = [...]string{
		_paramOCSignature,
		_paramOCCredential,
		_paramOCDate,
		_paramOCExpires,
		_paramOCVerb,
	}
)

// SignedURLAuthenticator is the authenticator responsible for authenticating signed URL requests.
type SignedURLAuthenticator struct {
	Logger             log.Logger
	PreSignedURLConfig config.PreSignedURL
	UserProvider       backend.UserBackend
	UserRoleAssigner   userroles.UserRoleAssigner
	Store              microstore.Store
	Now                func() time.Time
	URLVerifier        signedurl.Verifier
}

func (m SignedURLAuthenticator) shouldServeLegacy(req *http.Request) bool {
	if !m.PreSignedURLConfig.Enabled {
		return false
	}
	return req.URL.Query().Get(_paramOCSignature) != ""
}

func (m SignedURLAuthenticator) shouldServe(req *http.Request) bool {
	if m.URLVerifier == nil {
		return false
	}
	return req.URL.Query().Get(_paramOCJWTSig) != ""
}

func (m SignedURLAuthenticator) validate(req *http.Request) (err error) {
	query := req.URL.Query()

	if err := m.allRequiredParametersArePresent(query); err != nil {
		return err
	}

	if err := m.requestMethodMatches(req.Method, query); err != nil {
		return err
	}

	if err := m.requestMethodIsAllowed(req.Method); err != nil {
		return err
	}

	if err = m.urlIsExpired(query); err != nil {
		return err
	}

	if err := m.signatureIsValid(req); err != nil {
		return err
	}

	return nil
}

func (m SignedURLAuthenticator) allRequiredParametersArePresent(query url.Values) (err error) {
	// check if required query parameters exist in given request query parameters
	// OC-Signature - the computed signature - server will verify the request upon this REQUIRED
	// OC-Credential - defines the user scope (shall we use the opencloud user id here - this might leak internal data ....) REQUIRED
	// OC-Date - defined the date the url was signed (ISO 8601 UTC) REQUIRED
	// OC-Expires - defines the expiry interval in seconds (between 1 and 604800 = 7 days) REQUIRED
	// TODO OC-Verb - defines for which http verb the request is valid - defaults to GET OPTIONAL
	for _, p := range _requiredParams {
		if query.Get(p) == "" {
			return fmt.Errorf("required %s parameter not found", p)
		}
	}

	return nil
}

func (m SignedURLAuthenticator) requestMethodMatches(meth string, query url.Values) (err error) {
	// check if given url query parameter OC-Verb matches given request method
	if !strings.EqualFold(meth, query.Get(_paramOCVerb)) {
		return errors.New("required OC-Verb parameter did not match request method")
	}

	return nil
}

func (m SignedURLAuthenticator) requestMethodIsAllowed(meth string) (err error) {
	//  check if given request method is allowed
	methodIsAllowed := false
	for _, am := range m.PreSignedURLConfig.AllowedHTTPMethods {
		if strings.EqualFold(meth, am) {
			methodIsAllowed = true
			break
		}
	}

	if !methodIsAllowed {
		return errors.New("request method is not listed in PreSignedURLConfig AllowedHTTPMethods")
	}

	return nil
}

func (m SignedURLAuthenticator) urlIsExpired(query url.Values) (err error) {
	// check if url is expired by checking if given date (OC-Date) + expires in seconds (OC-Expires) is after now
	validFrom, err := time.Parse(time.RFC3339, query.Get(_paramOCDate))
	if err != nil {
		return err
	}

	requestExpiry, err := time.ParseDuration(query.Get(_paramOCExpires) + "s")
	if err != nil {
		return err
	}

	validTo := validFrom.Add(requestExpiry)
	if !(m.Now().Before(validTo)) {
		return errors.New("URL is expired")
	}

	return nil
}

func (m SignedURLAuthenticator) signatureIsValid(req *http.Request) (err error) {
	c := revactx.ContextMustGetUser(req.Context())
	signingKey, err := m.Store.Read(c.Id.OpaqueId)
	if err != nil {
		m.Logger.Error().Err(err).Msg("could not retrieve signing key")
		return err
	}
	if len(signingKey[0].Value) == 0 {
		m.Logger.Error().Err(err).Msg("signing key empty")
		return err
	}
	u := m.buildUrlToSign(req)
	computedSignature := m.createSignature(u, signingKey[0].Value)
	signatureInURL := req.URL.Query().Get(_paramOCSignature)
	if computedSignature == signatureInURL {
		return nil
	}

	// try a workaround for https://github.com/owncloud/ocis/issues/10180
	// Some reverse proxies might replace $ with %24 in the URL leading to a mismatch in the signature
	u = strings.Replace(u, "$", "%24", 1)
	computedSignature = m.createSignature(u, signingKey[0].Value)
	signatureInURL = req.URL.Query().Get(_paramOCSignature)
	if computedSignature == signatureInURL {
		return nil
	}

	return fmt.Errorf("signature mismatch: expected %s != actual %s", computedSignature, signatureInURL)
}

func (m SignedURLAuthenticator) buildUrlToSign(req *http.Request) string {
	q := req.URL.Query()

	// only params required for signing
	signParameters := make(url.Values)
	signParameters.Add(_paramOCCredential, q.Get(_paramOCCredential))
	signParameters.Add(_paramOCDate, q.Get(_paramOCDate))
	signParameters.Add(_paramOCExpires, q.Get(_paramOCExpires))
	signParameters.Add(_paramOCVerb, q.Get(_paramOCVerb))

	// remaining query params
	q.Del(_paramOCAlgo)
	q.Del(_paramOCCredential)
	q.Del(_paramOCDate)
	q.Del(_paramOCExpires)
	q.Del(_paramOCSignature)
	q.Del(_paramOCVerb)

	url := *req.URL
	if len(q) == 0 {
		url.RawQuery = signParameters.Encode()
	} else {
		url.RawQuery = strings.Join([]string{q.Encode(), signParameters.Encode()}, "&")
	}
	u := url.String()
	if !url.IsAbs() {
		u = "https://" + req.Host + u // TODO where do we get the scheme
	}
	return u
}

func (m SignedURLAuthenticator) createSignature(url string, signingKey []byte) string {
	// the oc10 signature check: $hash = \hash_pbkdf2("sha512", $url, $signingKey, 10000, 64, false);
	// - sets the length of the output string to 64
	// - sets raw output to false ->  if raw_output is FALSE length corresponds to twice the byte-length of the derived key (as every byte of the key is returned as two hexits).
	// TODO change to length 128 in oc10?
	// fo golangs pbkdf2.Key we need to use 32 because it will be encoded into 64 hexits later
	hash := pbkdf2.Key([]byte(url), signingKey, 10000, 32, sha512.New)
	return hex.EncodeToString(hash)
}

// Authenticate implements the authenticator interface to authenticate requests via signed URL auth.
func (m SignedURLAuthenticator) Authenticate(r *http.Request) (*http.Request, bool) {
	switch {
	case m.shouldServeLegacy(r):
		return m.authenticateLegacy(r)
	case m.shouldServe(r):
		return m.authenticate(r)
	}
	return nil, false
}

func (m SignedURLAuthenticator) authenticate(r *http.Request) (*http.Request, bool) {
	u := r.URL.String()
	if !r.URL.IsAbs() {
		u = "https://" + r.Host + u
	}

	userid, err := m.URLVerifier.Verify(u)
	if err != nil {
		m.Logger.Error().
			Err(err).
			Str("authenticator", "signed_url_jwt").
			Str("path", r.URL.Path).
			Str("url", u).
			Msg("Could not verify JWT signature")
		return nil, false
	}
	user, _, err := m.UserProvider.GetUserByClaims(r.Context(), "userid", userid)
	if err != nil {
		m.Logger.Error().
			Err(err).
			Str("authenticator", "signed_url_jwt").
			Str("path", r.URL.Path).
			Msg("Could not get user by claim")
		return nil, false
	}
	user, err = m.UserRoleAssigner.ApplyUserRole(r.Context(), user)
	if err != nil {
		m.Logger.Error().
			Err(err).
			Str("authenticator", "signed_url").
			Str("path", r.URL.Path).
			Msg("Could not get user by claim")
		return nil, false
	}
	ctx := revactx.ContextSetUser(r.Context(), user)
	r = r.WithContext(ctx)
	m.Logger.Debug().
		Str("authenticator", "signed_url").
		Str("path", r.URL.Path).
		Msg("successfully authenticated request")
	return r, true
}

// authenticateLegacy is a helper function to authenticate requests that use the legacy
// client side signed URLs
func (m SignedURLAuthenticator) authenticateLegacy(r *http.Request) (*http.Request, bool) {
	user, _, err := m.UserProvider.GetUserByClaims(r.Context(), "username", r.URL.Query().Get(_paramOCCredential))
	if err != nil {
		m.Logger.Error().
			Err(err).
			Str("authenticator", "signed_url").
			Str("path", r.URL.Path).
			Msg("Could not get user by claim")
		return nil, false
	}

	user, err = m.UserRoleAssigner.ApplyUserRole(r.Context(), user)
	if err != nil {
		m.Logger.Error().
			Err(err).
			Str("authenticator", "signed_url").
			Str("path", r.URL.Path).
			Msg("Could not get user by claim")
		return nil, false
	}

	ctx := revactx.ContextSetUser(r.Context(), user)

	r = r.WithContext(ctx)

	if err := m.validate(r); err != nil {
		m.Logger.Error().
			Err(err).
			Str("authenticator", "signed_url").
			Str("path", r.URL.Path).
			Str("url", r.URL.String()).
			Msg("Could not get user by claim")
		return nil, false
	}

	m.Logger.Debug().
		Str("authenticator", "signed_url").
		Str("path", r.URL.Path).
		Msg("successfully authenticated request")
	return r, true
}
