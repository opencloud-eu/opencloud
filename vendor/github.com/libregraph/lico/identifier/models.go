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

package identifier

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/libregraph/lico/identifier/meta"
	"github.com/libregraph/lico/identity/clients"
)

// A LogonRequest is the request data as sent to the logon endpoint
type LogonRequest struct {
	State string `json:"state"`

	Params []string      `json:"params"`
	Hello  *HelloRequest `json:"hello"`
}

// A LogonResponse holds a response as sent by the logon endpoint.
type LogonResponse struct {
	Success bool   `json:"success"`
	State   string `json:"state"`

	Hello *HelloResponse `json:"hello"`
}

// A HelloRequest is the request data as send to the hello endpoint.
type HelloRequest struct {
	State          string `json:"state"`
	Flow           string `json:"flow"`
	RawScope       string `json:"scope"`
	RawPrompt      string `json:"prompt"`
	ClientID       string `json:"client_id"`
	RawRedirectURI string `json:"redirect_uri"`
	RawIDTokenHint string `json:"id_token_hint"`
	RawMaxAge      string `json:"max_age"`

	Scopes      map[string]bool `json:"-"`
	Prompts     map[string]bool `json:"-"`
	RedirectURI *url.URL        `json:"-"`
	IDTokenHint *jwt.Token      `json:"-"`
	MaxAge      time.Duration   `json:"-"`

	//TODO(longsleep): Add support to pass request parameters as JWT as
	// specified in http://openid.net/specs/openid-connect-core-1_0.html#JWTRequests
}

func (hr *HelloRequest) parse() error {
	hr.Scopes = make(map[string]bool)
	hr.Prompts = make(map[string]bool)

	hr.RedirectURI, _ = url.Parse(hr.RawRedirectURI)

	if hr.RawScope != "" {
		for _, scope := range strings.Split(hr.RawScope, " ") {
			hr.Scopes[scope] = true
		}
	}
	if hr.RawPrompt != "" {
		for _, prompt := range strings.Split(hr.RawPrompt, " ") {
			hr.Prompts[prompt] = true
		}
	}
	if hr.RawMaxAge != "" {
		maxAgeInt, err := strconv.ParseInt(hr.RawMaxAge, 10, 64)
		if err != nil {
			return err
		}
		hr.MaxAge = time.Duration(maxAgeInt) * time.Second
	}

	return nil
}

// A HelloResponse holds a response as sent by the hello endpoint.
type HelloResponse struct {
	State       string `json:"state"`
	Flow        string `json:"flow"`
	Success     bool   `json:"success"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"displayName,omitempty"`

	Next          string           `json:"next,omitempty"`
	ContinueURI   string           `json:"continue_uri,omitempty"`
	Scopes        map[string]bool  `json:"scopes,omitempty"`
	ClientDetails *clients.Details `json:"client,omitempty"`
	Meta          *meta.Meta       `json:"meta,omitempty"`
	Branding      *meta.Branding   `json:"branding,omitempty"`
}

// A StateRequest is a general request with a state.
type StateRequest struct {
	State string
}

// A StateResponse hilds a response as reply to a StateRequest.
type StateResponse struct {
	Success bool   `json:"success"`
	State   string `json:"state"`
}

// StateData contains data bound to a state.
type StateData struct {
	State string `json:"state"`
	Mode  string `json:"mode,omitempty"`

	RawQuery string `json:"raw_query,omitempty"`

	ClientID string `json:"client_id"`
	Ref      string `json:"ref,omitempty"`

	Extra map[string]interface{} `json:"extra,omitempty"`

	Trampolin *TrampolinData `json:"trampolin,omitempty"`
}

type TrampolinData struct {
	URI   string `json:"uri"`
	Scope string `json:"scope"`
}

// A ConsentRequest is the request data as sent to the consent endpoint.
type ConsentRequest struct {
	State          string `json:"state"`
	Allow          bool   `json:"allow"`
	RawScope       string `json:"scope"`
	ClientID       string `json:"client_id"`
	RawRedirectURI string `json:"redirect_uri"`
	Ref            string `json:"ref"`
	Nonce          string `json:"flow_nonce"`
}

// Consent is the data received and sent to allow or cancel consent flows.
type Consent struct {
	Allow    bool   `json:"allow"`
	RawScope string `json:"scope"`
}

// Scopes returns the associated consents approved scopes filtered by the
//provided requested scopes and the full unfiltered approved scopes table.
func (c *Consent) Scopes(requestedScopes map[string]bool) (map[string]bool, map[string]bool) {
	scopes := make(map[string]bool)
	if c.RawScope != "" {
		for _, scope := range strings.Split(c.RawScope, " ") {
			scopes[scope] = true
		}
	}

	approved := make(map[string]bool)
	for n, v := range requestedScopes {
		if ok, _ := scopes[n]; ok && v {
			approved[n] = true
		}
	}

	return approved, scopes
}
