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
	"bytes"
	"fmt"
	"net/http"

	"github.com/libregraph/oidc-go"
	"github.com/longsleep/rndm"

	"github.com/libregraph/lico/identifier/meta"
	"github.com/libregraph/lico/identifier/meta/scopes"
)

func (i *Identifier) writeWebappIndexHTML(rw http.ResponseWriter, req *http.Request) {
	nonce := rndm.GenerateRandomString(32)

	// FIXME(longsleep): Set a secure CSP. Right now we need `data:` for images
	// since it is used. Since `data:` URLs possibly could allow xss, a better
	// way should be found for our early loading inline SVG stuff.
	rw.Header().Set("Content-Security-Policy", fmt.Sprintf("default-src 'self'; img-src 'self' data:; font-src 'self' data:; script-src 'self' 'nonce-%s'; style-src 'self' 'nonce-%s'; base-uri 'none'; frame-ancestors 'none';", nonce, nonce))

	// Write index with random nonce to response.
	index := bytes.ReplaceAll(i.webappIndexHTML, []byte("__CSP_NONCE__"), []byte(nonce))
	rw.Write(index)
}

func (i Identifier) writeHelloResponse(rw http.ResponseWriter, req *http.Request, r *HelloRequest, identifiedUser *IdentifiedUser) (*HelloResponse, error) {
	var err error
	response := &HelloResponse{
		State: r.State,
		Branding: &meta.Branding{
			BannerLogo:        i.defaultBannerLogo,
			UsernameHintText:  i.Config.DefaultUsernameHintText,
			SignInPageText:    i.Config.DefaultSignInPageText,
			SignInPageLogoURI: i.Config.DefaultSignInPageLogoURI,
			Locales:           i.Config.UILocales,
		},
	}

handleHelloLoop:
	for {
		// Check prompt value.
		switch {
		case r.Prompts[oidc.PromptNone] == true:
			// Never show sign-in, directly return error.
			return nil, fmt.Errorf("prompt none requested")
		case r.Prompts[oidc.PromptLogin] == true:
			// Ignore all potential sources, when prompt login was requested.
			if identifiedUser != nil {
				response.Username = identifiedUser.Username()
				response.DisplayName = identifiedUser.Name()
				if response.Username != "" {
					response.Success = true
				}
			}
			break handleHelloLoop
		default:
			// Let all other prompt values pass.
		}

		if identifiedUser == nil {
			// Check if logged in via cookie.
			identifiedUser, err = i.GetUserFromLogonCookie(req.Context(), req, r.MaxAge, true)
			if err != nil {
				i.logger.WithError(err).Debugln("identifier failed to decode logon cookie in hello")
			}
		}

		if identifiedUser != nil {
			response.Username = identifiedUser.Username()
			response.DisplayName = identifiedUser.Name()
			if response.Username != "" {
				response.Success = true
				break
			}
		}

		break
	}

	if !response.Success {
		return response, nil
	}

	switch r.Flow {
	case FlowOAuth:
		fallthrough
	case FlowConsent:
		fallthrough
	case FlowOIDC:
		// TODO(longsleep): Add something to validate the parameters.
		clientDetails, err := i.clients.Lookup(req.Context(), r.ClientID, "", r.RedirectURI, "", true)
		if err != nil {
			return nil, err
		}

		promptConsent := false

		// Check prompt value.
		switch {
		case r.Prompts[oidc.PromptConsent] == true:
			promptConsent = true
		default:
			// Let all other prompt values pass.
		}

		// If not trusted, always force consent.
		if !clientDetails.Trusted {
			promptConsent = true
		}

		if promptConsent {
			// TODO(longsleep): Filter scopes to scopes we know about and all.
			response.Next = FlowConsent
			response.Scopes = r.Scopes
			response.ClientDetails = clientDetails
			response.Meta = &meta.Meta{
				Scopes: scopes.NewScopesFromIDs(r.Scopes, i.meta.Scopes),
			}
		}

		// Add authorize endpoint URI as continue URI.
		response.ContinueURI = i.authorizationEndpointURI.String()
		response.Flow = r.Flow
	}

	return response, nil
}
