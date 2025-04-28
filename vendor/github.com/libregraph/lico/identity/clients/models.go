/*
 * Copyright 2017-2019 Kopano and its licensors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package clients

import (
	"context"
	"crypto"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/longsleep/rndm"
	"github.com/mendsley/gojwk"
	"golang.org/x/crypto/blake2b"
	_ "gopkg.in/yaml.v2" // Make sure we have yaml.
)

// Constat data used with dynamic stateless clients.
const (
	DynamicStatelessClientIDPrefix     = "dyn."
	DynamicStatelessClientStaticSaltV1 = "konnect-client-v1"
)

// RegistryData is the base structur of our client registry configuration file.
type RegistryData struct {
	Clients []*ClientRegistration `yaml:"clients,flow"`
}

// ClientRegistration defines a client with its properties.
type ClientRegistration struct {
	ID     string `yaml:"id" json:"-"`
	Secret string `yaml:"secret" json:"-"`

	Trusted       bool     `yaml:"trusted" json:"-"`
	TrustedScopes []string `yaml:"trusted_scopes" json:"-"`
	Insecure      bool     `yaml:"insecure" json:"-"`

	ImplicitScopes []string `yaml:"implicit_scopes" json:"-"`

	Dynamic         bool      `yaml:"-" json:"-"`
	IDIssuedAt      time.Time `yaml:"-" json:"-"`
	SecretExpiresAt time.Time `yaml:"-" json:"-"`

	Contacts        []string `yaml:"contacts,flow" json:"contacts,omitempty"`
	Name            string   `yaml:"name" json:"name,omitempty"`
	URI             string   `yaml:"uri"  json:"uri,omitempty"`
	GrantTypes      []string `yaml:"grant_types,flow" json:"grant_types,omitempty"`
	ApplicationType string   `yaml:"application_type"  json:"application_type,omitempty"`

	RedirectURIs []string `yaml:"redirect_uris,flow" json:"redirect_uris,omitempty"`
	Origins      []string `yaml:"origins,flow" json:"-"`

	JWKS *gojwk.Key `yaml:"jwks" json:"-"`

	RawIDTokenSignedResponseAlg    string `yaml:"id_token_signed_response_alg" json:"id_token_signed_response_alg,omitempty"`
	RawUserInfoSignedResponseAlg   string `yaml:"userinfo_signed_response_alg" json:"userinfo_signed_response_alg,omitempty"`
	RawRequestObjectSigningAlg     string `yaml:"request_object_signing_alg" json:"request_object_signing_alg,omitempty"`
	RawTokenEndpointAuthMethod     string `yaml:"token_endpoint_auth_method" json:"token_endpoint_auth_method,omitempty"`
	RawTokenEndpointAuthSigningAlg string `yaml:"token_endpoint_auth_signing_alg"  json:"token_endpoint_auth_signing_alg,omitempty"`

	PostLogoutRedirectURIs []string `yaml:"post_logout_redirect_uris,flow" json:"post_logout_redirect_uris,omitempty"`
}

// Validate validates the associated client registration data and returns error
// if the data is not valid.
func (cr *ClientRegistration) Validate() error {
	return nil
}

// Secure looks up the a matching key from the accociated client registration
// and returns its public key part as a secured client.
func (cr *ClientRegistration) Secure(rawKid interface{}) (*Secured, error) {
	var kid string
	var key crypto.PublicKey
	var err error

	switch len(cr.JWKS.Keys) {
	case 0:
		// breaks
	case 1:
		// Use the one and only, no matter what kid says.
		key, err = cr.JWKS.Keys[0].DecodePublicKey()
		if err != nil {
			return nil, err
		}
		kid = cr.JWKS.Keys[0].Kid
	default:
		// Find by kid.
		kid, _ = rawKid.(string)
		if kid == "" {
			kid = "default"
		}
		for _, k := range cr.JWKS.Keys {
			if kid == k.Kid {
				key, err = k.DecodePublicKey()
				if err != nil {
					return nil, err
				}
				break
			}
		}
	}

	if key == nil {
		return nil, fmt.Errorf("unknown kid")
	}

	return &Secured{
		ID:              cr.ID,
		DisplayName:     cr.Name,
		ApplicationType: cr.ApplicationType,

		Kid:       kid,
		PublicKey: key,

		TrustedScopes: cr.TrustedScopes,

		Registration: cr,
	}, nil
}

// SetDynamic modifieds the required data for the associated client registration
// so it becomes a dynamic client.
func (cr *ClientRegistration) SetDynamic(ctx context.Context, creator func(ctx context.Context, signingMethod jwt.SigningMethod, claims jwt.Claims) (string, error)) error {
	if creator == nil {
		return fmt.Errorf("no creator")
	}

	if cr.ID != "" {
		return fmt.Errorf("has ID already")
	}

	registry, ok := FromRegistryContext(ctx)
	if !ok {
		return fmt.Errorf("no registry")
	}

	// Initialize basic client registration data for dynamic client.
	cr.IDIssuedAt = time.Now()
	if registry.dynamicClientSecretDuration > 0 {
		cr.SecretExpiresAt = time.Now().Add(registry.dynamicClientSecretDuration)
	}
	cr.Dynamic = true

	sub, secret, err := cr.makeSecret(nil)
	if err != nil {
		return fmt.Errorf("failed to make dynamic client secret: %v", err)
	}

	// Stateless Dynamic Client Registration encodes all relevant data in the
	// client_id. See https://openid.net/specs/openid-connect-registration-1_0.html#StatelessRegistration
	// for more information. We use a JWT as client_id.
	claims := &RegistrationClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   sub,
			IssuedAt:  jwt.NewNumericDate(cr.IDIssuedAt),
			ExpiresAt: jwt.NewNumericDate(cr.SecretExpiresAt),
		},
		ClientRegistration: cr,
	}

	// Create signed stateless client ID by help of the provided creator function.
	id, err := creator(ctx, nil, claims)
	if err != nil {
		return nil
	}

	// Fill in ID and secret.
	cr.ID = DynamicStatelessClientIDPrefix + id
	cr.Secret = secret

	return nil
}

// ApplyImplicitScopes apples the associated registration's implicit scopes to
// the provided scopes map.
func (cr *ClientRegistration) ApplyImplicitScopes(scopes map[string]bool) error {
	for _, scope := range cr.ImplicitScopes {
		if scope != "" {
			scopes[scope] = true
		}
	}
	return nil
}

func (cr *ClientRegistration) makeSecret(secret []byte) (string, string, error) {
	// Create random secret. HMAC the client name with it to get the subject.
	if secret == nil {
		secret = rndm.GenerateRandomBytes(64)
	}

	hasher, err := blake2b.New512(secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to create hasher for dynamic client_id: %v", err)
	}
	hasher.Write([]byte(cr.Name))
	hasher.Write([]byte(" "))
	hasher.Write([]byte(DynamicStatelessClientStaticSaltV1))
	sub := base64.RawURLEncoding.EncodeToString(hasher.Sum(nil))

	return sub, base64.RawURLEncoding.EncodeToString(secret), nil
}

func (cr *ClientRegistration) validateSecret(clientSecret string) (bool, error) {
	if cr.Dynamic {
		if cr.Secret == "" {
			// Fail fast, since dynamic clients must have a secret.
			return false, fmt.Errorf("no secret in registration")
		}

		// Dynamic clients use hashed passwords.
		secret, err := base64.RawURLEncoding.DecodeString(clientSecret)
		if err != nil {
			return false, fmt.Errorf("failed to decode client secret: %v", err)
		}
		sub, _, err := cr.makeSecret(secret)
		if err != nil {
			return false, fmt.Errorf("failed to produce client secret for comparison: %v", err)
		}

		return subtle.ConstantTimeCompare([]byte(sub), []byte(cr.Secret)) == 1, nil
	}

	if cr.Secret != "" && subtle.ConstantTimeCompare([]byte(clientSecret), []byte(cr.Secret)) != 1 {
		return false, nil
	}
	return true, nil
}
