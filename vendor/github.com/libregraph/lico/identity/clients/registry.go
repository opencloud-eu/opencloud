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
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/libregraph/oidc-go"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Registry implements the registry for registered clients.
type Registry struct {
	mutex sync.RWMutex

	trustedURI *url.URL
	clients    map[string]*ClientRegistration

	allowDynamicClientRegistration bool
	dynamicClientSecretDuration    time.Duration

	StatelessCreator   func(ctx context.Context, signingMethod jwt.SigningMethod, claims jwt.Claims) (string, error)
	StatelessValidator func(token *jwt.Token) (interface{}, error)

	logger logrus.FieldLogger
}

// contextKey is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type contextKey int

// claimsKey is the key for claims in contexts. It is
// unexported; clients use konnect.NewClaimsContext and
// connect.FromClaimsContext instead of using this key directly.
var registryKey contextKey

// NewRegistry created a new client Registry with the provided parameters.
func NewRegistry(ctx context.Context, trustedURI *url.URL, registrationConfFilepath string, allowDynamicClientRegistration bool, dynamicClientSecretDuration time.Duration, logger logrus.FieldLogger) (*Registry, error) {
	registryData := &RegistryData{}

	if registrationConfFilepath != "" {
		logger.Debugf("parsing identifier registration conf from %v", registrationConfFilepath)
		registryFile, err := ioutil.ReadFile(registrationConfFilepath)
		if err != nil {
			return nil, err
		}

		err = yaml.Unmarshal(registryFile, registryData)
		if err != nil {
			return nil, err
		}
	}

	r := &Registry{
		trustedURI: trustedURI,
		clients:    make(map[string]*ClientRegistration),

		allowDynamicClientRegistration: allowDynamicClientRegistration,
		dynamicClientSecretDuration:    dynamicClientSecretDuration,

		logger: logger,
	}

	for _, client := range registryData.Clients {
		validateErr := client.Validate()
		registerErr := r.Register(client)
		fields := logrus.Fields{
			"client_id":          client.ID,
			"with_client_secret": client.Secret != "",
			"trusted":            client.Trusted,
			"insecure":           client.Insecure,
			"application_type":   client.ApplicationType,
			"redirect_uris":      client.RedirectURIs,
			"origins":            client.Origins,
		}

		if validateErr != nil {
			logger.WithError(validateErr).WithFields(fields).Warnln("skipped registration of invalid client entry")
			continue
		}
		if registerErr != nil {
			logger.WithError(registerErr).WithFields(fields).Warnln("skipped registration of invalid client")
			continue
		}
		logger.WithFields(fields).Debugln("registered client")
	}

	return r, nil
}

// NewRegistryContext returns a new Context that carries value provided Registry.
func NewRegistryContext(ctx context.Context, r *Registry) context.Context {
	return context.WithValue(ctx, registryKey, r)
}

// FromRegistryContext returns the Registry value stored in ctx, if any.
func FromRegistryContext(ctx context.Context) (*Registry, bool) {
	r, ok := ctx.Value(registryKey).(*Registry)
	return r, ok
}

// Register validates the provided client registration and adds the client
// to the accociated registry if valid. Returns error otherwise.
func (r *Registry) Register(client *ClientRegistration) error {
	if client.ID == "" {
		return errors.New("invalid client_id")
	}

	if !client.Insecure && len(client.RedirectURIs) == 0 {
		return errors.New("no redirect_uris")
	}

	switch client.ApplicationType {
	case "":
		client.ApplicationType = oidc.ApplicationTypeWeb
		fallthrough
	case oidc.ApplicationTypeWeb:
		for _, urlString := range client.RedirectURIs {
			// http://openid.net/specs/openid-connect-registration-1_0.html#ClientMetadata
			parsed, _ := url.Parse(urlString)
			if parsed == nil || parsed.Host == "" {
				return fmt.Errorf("invalid redirect_uri %v - invalid or no hostname", urlString)
			} else if !client.Insecure && parsed.Scheme != "https" {
				return fmt.Errorf("invalid redirect_uri %v - make sure to use https when application_type is web", parsed)
			} else if IsLocalNativeHTTPURI(parsed) {
				return fmt.Errorf("invalid redirect_uri %v - host must not be localhost when application_type is web", parsed)
			}

			if len(client.Origins) == 0 {
				// Auto add first redirect scheme and host as Origin if no
				// origin is explicitly configured.
				client.Origins = append(client.Origins, parsed.Scheme+"://"+parsed.Host)
			}
		}
		if !client.Insecure && len(client.Origins) == 0 {
			return errors.New("no origins - origin is required when application_type is web")
		}
		// breaks

	case oidc.ApplicationTypeNative:
		// breaks
		for _, urlString := range client.RedirectURIs {
			// http://openid.net/specs/openid-connect-registration-1_0.html#ClientMetadata
			parsed, _ := url.Parse(urlString)
			if parsed == nil || parsed.Host == "" {
				return fmt.Errorf("invalid redirect_uri %v - invalid uri or no hostname", urlString)
			} else if parsed.Scheme == "https" {
				return fmt.Errorf("invalid redirect_uri %v - scheme must not be https when application_type is native", parsed)
			} else if parsed.Scheme == "http" && !IsLocalNativeHTTPURI(parsed) {
				return fmt.Errorf("invalid redirect_uri %v = http host must be localhost when application_type is native", parsed)
			}
		}
		// breaks

	default:
		return fmt.Errorf("unknown application_type: %v", client.ApplicationType)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.clients[client.ID] = client
	return nil
}

// Validate checks if the provided client registration data complies to the
// provided parameters and returns error when it does not.
func (r *Registry) Validate(client *ClientRegistration, clientSecret string, redirectURIString string, originURIString string, withoutSecret bool) error {
	if client.ApplicationType == oidc.ApplicationTypeWeb {
		if originURIString != "" && (!client.Insecure || len(client.Origins) > 0) {
			// Compare originURI if it was given.
			originOK := false
			for _, urlString := range client.Origins {
				if urlString == originURIString {
					originOK = true
					break
				}
			}
			if !originOK {
				return fmt.Errorf("invalid origin: %v", originURIString)
			}
		}
	}

	if redirectURIString != "" && (!client.Insecure || len(client.RedirectURIs) > 0) {
		// Make sure to validate the redirect URI unless client is marked insecure
		// and has no configured redirect URIs.
		redirectURIOK := false
		for _, registeredURIString := range client.RedirectURIs {
			if client.ApplicationType == oidc.ApplicationTypeNative {
				registeredURI, _ := url.Parse(registeredURIString)
				if IsLocalNativeHTTPURI(registeredURI) {
					redirectURI, err := url.Parse(redirectURIString)
					if err != nil {
						break
					}
					if IsLocalNativeHTTPURI(redirectURI) {
						if registeredURI.Path == "" || redirectURI.Path == registeredURI.Path {
							redirectURIOK = true
							break
						}
					}
					continue
				}
			}
			if registeredURIString == redirectURIString {
				redirectURIOK = true
				break
			}
		}
		if !redirectURIOK {
			return fmt.Errorf("invalid redirect_uri: %v", redirectURIString)
		}
	}

	if !withoutSecret {
		if valid, err := client.validateSecret(clientSecret); !valid {
			return fmt.Errorf("invalid client_secret: %v", err)
		}
	}

	return nil
}

// Lookup returns and validates the clients Detail information for the provided
// parameters from the accociated registry.
func (r *Registry) Lookup(ctx context.Context, clientID string, clientSecret string, redirectURI *url.URL, originURIString string, withoutSecret bool) (*Details, error) {
	var err error
	var trusted bool
	var dynamic bool
	var displayName string

	if clientID == "" {
		return nil, fmt.Errorf("invalid client_id")
	}

	originURI, _ := url.Parse(originURIString)

	// Implicit trust for web clients running and redirecting to the same origin
	// as the issuer (ourselves).
	if r.trustedURI != nil {
		for {
			if r.trustedURI.Scheme != redirectURI.Scheme || r.trustedURI.Host != redirectURI.Host {
				break
			}
			if originURI.Scheme != "" && (r.trustedURI.Scheme != originURI.Scheme || r.trustedURI.Host != originURI.Host) {
				break
			}
			trusted = true
			break
		}
	}

	// Lookup client registration.
	r.mutex.RLock()
	registration, _ := r.clients[clientID]
	r.mutex.RUnlock()

	if registration == nil && strings.HasPrefix(clientID, DynamicStatelessClientIDPrefix) {
		trusted = false
		dynamic = true
	}

	// Lookup dynamic clients when it makes sense.
	if dynamic && registration == nil {
		registration, _ = r.getDynamicClient(clientID)
	}

	if registration != nil {
		redirectURIBase := &url.URL{
			Scheme: redirectURI.Scheme,
			Host:   redirectURI.Host,
			Path:   redirectURI.Path,
		}
		err = r.Validate(registration, clientSecret, redirectURIBase.String(), originURIString, withoutSecret)
		displayName = registration.Name
		trusted = registration.Trusted
	} else {
		if trusted {
			// Always let in implicitly trusted clients.
			err = nil
		} else {
			err = fmt.Errorf("unknown client_id: %v", clientID)
		}
	}

	if err != nil {
		return nil, err
	}

	redirecURIString := redirectURI.String()

	r.logger.WithFields(logrus.Fields{
		"trusted":      trusted,
		"client_id":    clientID,
		"redirect_uri": redirecURIString,
		"known":        registration != nil,
	}).Debugln("identifier client lookup")

	return &Details{
		ID:          clientID,
		RedirectURI: redirecURIString,
		DisplayName: displayName,
		Trusted:     trusted,

		Registration: registration,
	}, nil
}

// Get returns the registered clients registration for the provided client ID.
func (r *Registry) Get(ctx context.Context, clientID string) (*ClientRegistration, bool) {
	// Lookup client registration.
	r.mutex.RLock()
	registration, ok := r.clients[clientID]
	r.mutex.RUnlock()
	if ok {
		return registration, true
	}

	return r.getDynamicClient(clientID)
}

func (r *Registry) getDynamicClient(clientID string) (*ClientRegistration, bool) {
	var registration *ClientRegistration

	if len(clientID) >= len(DynamicStatelessClientIDPrefix) {
		tokenString := clientID[len(DynamicStatelessClientIDPrefix):]
		if token, err := jwt.ParseWithClaims(tokenString, &RegistrationClaims{}, func(token *jwt.Token) (interface{}, error) {
			if r.StatelessValidator == nil {
				return nil, fmt.Errorf("no validator for dynamic client ids")
			}
			return r.StatelessValidator(token)
		}); err == nil {
			if claims, ok := token.Claims.(*RegistrationClaims); ok && token.Valid {
				// TODO(longsleep): Add secure client secret.
				registration = claims.ClientRegistration
				registration.ID = clientID
				registration.Secret = claims.RegisteredClaims.Subject
				registration.Dynamic = true
			}
		}
	}

	return registration, registration != nil
}
