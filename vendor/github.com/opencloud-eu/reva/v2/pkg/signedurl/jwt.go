package signedurl

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTSignedURL implements the Signer and Verifier interfaces using JWT for signing URLs.
type JWTSignedURL struct {
	JWTOptions
}

type claims struct {
	TargetURL string `json:"target_url"`
	jwt.RegisteredClaims
}

// JWTOption defines a single option function.
type JWTOption func(o *JWTOptions)

// JWTOptions defines the available options for this package.
type JWTOptions struct {
	secret     string // Secret key used for signing and verifying JWTs
	queryParam string // Name of the query parameter for the signature
}

func NewJWTSignedURL(opts ...JWTOption) (*JWTSignedURL, error) {
	opt := JWTOptions{}
	for _, o := range opts {
		o(&opt)
	}

	if opt.secret == "" {
		return nil, ErrInvalidKey
	}

	if opt.queryParam == "" {
		opt.queryParam = "oc-jwt-sig"
	}

	return &JWTSignedURL{opt}, nil
}

func WithSecret(secret string) JWTOption {
	return func(o *JWTOptions) {
		o.secret = secret
	}
}

func WithQueryParam(queryParam string) JWTOption {
	return func(o *JWTOptions) {
		o.queryParam = queryParam
	}
}

// Sign signs a URL using JWT with a specified time-to-live (ttl).
func (j *JWTSignedURL) Sign(unsignedURL, subject string, ttl time.Duration) (string, error) {
	// Re-encode the Query parameters to ensure they are "normalized" (Values.Encode() does return them alphabetically ordered).
	u, err := url.Parse(unsignedURL)
	if err != nil {
		return "", NewSignedURLError(err, "failed to parse url")
	}
	query := u.Query()
	u.RawQuery = query.Encode()
	c := claims{
		TargetURL: u.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			Issuer:    "reva",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   subject,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	signedToken, err := token.SignedString([]byte(j.secret))
	if err != nil {
		return "", fmt.Errorf("signing failed: %w", err)
	}
	query.Set(j.queryParam, signedToken)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

// Verify verifies a signed URL using a JWT. Returns the subject of the JWT if verification is successful.
func (j *JWTSignedURL) Verify(signedURL string) (string, error) {
	u, err := url.Parse(signedURL)
	if err != nil {
		return "", NewSignatureVerificationError(fmt.Errorf("could not parse URL: %w", err))
	}
	query := u.Query()
	tokenString := query.Get(j.queryParam)
	if tokenString == "" {
		return "", NewSignatureVerificationError(errors.New("no signature in url"))
	}
	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(token *jwt.Token) (any, error) { return []byte(j.secret), nil })
	if err != nil {
		return "", NewSignatureVerificationError(err)
	}
	c, ok := token.Claims.(*claims)
	if !ok {
		return "", NewSignatureVerificationError(errors.New("invalid JWT claims"))
	}

	query.Del(j.queryParam)
	u.RawQuery = query.Encode()

	if c.TargetURL != u.String() {
		return "", NewSignatureVerificationError(errors.New("url mismatch"))
	}

	return c.Subject, nil
}
