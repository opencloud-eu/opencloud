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

package authorities

import (
	"crypto"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/golang-jwt/jwt/v5"
	"github.com/libregraph/oidc-go"
)

// Details hold immutable information about external authorities identified by ID.
type Details struct {
	ID            string
	Name          string
	AuthorityType string

	ClientID     string
	ClientSecret string

	Trusted  bool
	Insecure bool

	Scopes              []string
	ResponseType        string
	ResponseMode        string
	CodeChallengeMethod string

	EndSessionEnabled bool

	registration AuthorityRegistration

	ready bool

	validationKeys map[string]crypto.PublicKey
}

// IsReady returns wether or not the associated registration entry was ready
// at time of creation of the associated details.
func (d *Details) IsReady() bool {
	return d.ready
}

// IdentityClaimValue returns the identity claim value from the provided data.
func (d *Details) IdentityClaimValue(claims interface{}) (string, map[string]interface{}, error) {
	return d.registration.IdentityClaimValue(claims)
}

// MakeRedirectAuthenticationRequestURL returns the authentication request
// URL which can be used to initiate authentication with the associated
// authority. It takes a state as parameter and in addition to the URL it also
// returns a mapping of extra state data and potentially an error.
func (d *Details) MakeRedirectAuthenticationRequestURL(state string) (*url.URL, map[string]interface{}, error) {
	return d.registration.MakeRedirectAuthenticationRequestURL(state)
}

// MakeRedirectEndSessionRequestURL returns the end session request URL which
// can be used to initiate end session with the associated authority. It takes
// a state as paraeter and in addition to the URL it also returns a mappting
// of extra state data and potentially an error.
func (d *Details) MakeRedirectEndSessionRequestURL(ref interface{}, state string) (*url.URL, map[string]interface{}, error) {
	return d.registration.MakeRedirectEndSessionRequestURL(ref, state)
}

// MakeRedirectEndSessionResponseURL returns the end session response URL which
// can be used to redirect back the response for an incoming end session request.
// It takes the authority specific request and a state, returning the destination
// url, additional state mapping and potential error.
func (d *Details) MakeRedirectEndSessionResponseURL(req interface{}, state string) (*url.URL, map[string]interface{}, error) {
	return d.registration.MakeRedirectEndSessionResponseURL(req, state)
}

// ParseStateResponse takes an incoming request, a state and optional extra data
// and returns the parsed authority specific response data for that request or
// error.
func (d *Details) ParseStateResponse(req *http.Request, state string, extra map[string]interface{}) (interface{}, error) {
	return d.registration.ParseStateResponse(req, state, extra)
}

// JWTKeyfunc returns a key func to validate JWTs with the keys of the associated
// authority registration.
func (d *Details) JWTKeyfunc() jwt.Keyfunc {
	return d.validateJWT
}

func (d *Details) validateJWT(token *jwt.Token) (interface{}, error) {
	rawAlg, ok := token.Header[oidc.JWTHeaderAlg]
	if !ok {
		return nil, errors.New("no alg header")
	}
	alg, ok := rawAlg.(string)
	if !ok {
		return nil, errors.New("invalid alg value")
	}
	switch jwt.GetSigningMethod(alg).(type) {
	case *jwt.SigningMethodRSA:
	case *jwt.SigningMethodECDSA:
	case *jwt.SigningMethodRSAPSS:
	default:
		return nil, fmt.Errorf("unexpected alg value")
	}
	rawKid, ok := token.Header[oidc.JWTHeaderKeyID]
	if !ok {
		return nil, fmt.Errorf("no kid header")
	}
	kid, ok := rawKid.(string)
	if !ok {
		return nil, fmt.Errorf("invalid kid value")
	}

	if key, ok := d.validationKeys[kid]; ok {
		return key, nil
	}

	return nil, errors.New("no key available")
}

func (d *Details) Metadata() interface{} {
	return d.registration.Metadata()
}
