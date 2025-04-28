/*
 * Copyright 2017-2020 Kopano and its licensors
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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/libregraph/oidc-go"
	"github.com/longsleep/rndm"
	"golang.org/x/oauth2"

	"github.com/libregraph/lico/identity/authorities"
	konnectoidc "github.com/libregraph/lico/oidc"
	"github.com/libregraph/lico/oidc/payload"
	"github.com/libregraph/lico/utils"
)

func (i *Identifier) writeOAuth2Start(rw http.ResponseWriter, req *http.Request, authority *authorities.Details) {
	var err error

	if authority == nil {
		err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2TemporarilyUnavailable, "no authority")
	} else if !authority.IsReady() {
		err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2TemporarilyUnavailable, "authority not ready")
	}

	switch typedErr := err.(type) {
	case nil:
		// breaks
	case *konnectoidc.OAuth2Error:
		// Redirect back, with error.
		i.logger.WithFields(utils.ErrorAsFields(err)).Debugln("oauth2 start error")
		// NOTE(longsleep): Pass along error ID but not the description to avoid
		// leaking potentially internal information to our RP.
		uri, _ := url.Parse(i.authorizationEndpointURI.String())
		query, _ := url.ParseQuery(req.URL.RawQuery)
		query.Del("flow")
		query.Set("error", typedErr.ErrorID)
		query.Set("error_description", "identifier failed to authenticate")
		uri.RawQuery = query.Encode()
		utils.WriteRedirect(rw, http.StatusFound, uri, nil, false)
		return
	default:
		i.logger.WithError(err).Errorln("identifier failed to process oauth2 start")
		i.ErrorPage(rw, http.StatusInternalServerError, "", "oauth2 start failed")
		return
	}

	sd := &StateData{
		State:    rndm.GenerateRandomString(32),
		RawQuery: req.URL.RawQuery,

		ClientID: authority.ClientID,
		Ref:      authority.ID,
	}

	// Construct URL to redirect client to external OAuth2 authorize endpoints.
	uri, extra, err := authority.MakeRedirectAuthenticationRequestURL(sd.State)
	if err != nil {
		i.logger.WithError(err).Errorln("identifier failed to create authentication request: %w", err)
		i.ErrorPage(rw, http.StatusInternalServerError, "", "oauth2 start failed")
		return
	}
	if extra != nil {
		sd.Extra = extra
	} else {
		sd.Extra = make(map[string]interface{})
	}

	query := uri.Query()
	query.Add("client_id", authority.ClientID)
	if authority.ResponseType != "" {
		query.Add("response_type", authority.ResponseType)
	}
	if authority.ResponseMode != "" {
		query.Add("response_mode", authority.ResponseMode)
	}
	query.Add("scope", strings.Join(authority.Scopes, " "))
	query.Add("redirect_uri", i.oauth2CbEndpointURI.String())
	query.Add("nonce", rndm.GenerateRandomString(32))
	if authority.CodeChallengeMethod != "" {
		codeVerifier := rndm.GenerateRandomString(32)
		sd.Extra["code_verifier"] = codeVerifier
		codeChallenge := ""
		if codeChallenge, err = oidc.MakeCodeChallenge(authority.CodeChallengeMethod, codeVerifier); err == nil {
			query.Add("code_challenge", codeChallenge)
			query.Add("code_challenge_method", authority.CodeChallengeMethod)
		} else {
			i.logger.WithError(err).Debugln("identifier failed to create oauth2 code challenge")
			i.ErrorPage(rw, http.StatusInternalServerError, "", "failed to create code challenge")
			return
		}
	}
	if display := req.Form.Get("display"); display != "" {
		query.Add("display", display)
	}
	if prompt := req.Form.Get("prompt"); prompt != "" && prompt != oidc.PromptConsent {
		// Pass along all prompt values, except consent to external provider and
		// handle consent as needed ourselves.
		query.Add("prompt", prompt)
	}
	if maxAge := req.Form.Get("max_age"); maxAge != "" {
		query.Add("max_age", maxAge)
	}
	if uiLocales := req.Form.Get("ui_locales"); uiLocales != "" {
		query.Add("ui_locales", uiLocales)
	}
	if acrValues := req.Form.Get("acr_values"); acrValues != "" {
		query.Add("acr_values", acrValues)
	}
	if claimsLocales := req.Form.Get("claims_locales"); claimsLocales != "" {
		query.Add("claims_locales", claimsLocales)
	}

	// Set cookie which is consumed by the callback later.
	err = i.SetStateToStateCookie(req.Context(), rw, "oauth2/cb", sd)
	if err != nil {
		i.logger.WithError(err).Debugln("identifier failed to set oauth2 state cookie")
		i.ErrorPage(rw, http.StatusInternalServerError, "", "failed to set cookie")
		return
	}

	uri.RawQuery = query.Encode()
	utils.WriteRedirect(rw, http.StatusFound, uri, nil, false)
}

func (i *Identifier) writeOAuth2Cb(rw http.ResponseWriter, req *http.Request) {
	// Callbacks from authorization or end session. Validate as specified at
	// https://tools.ietf.org/html/rfc6749#section-4.1.2 and https://tools.ietf.org/html/rfc6749#section-10.12.
	var err error
	var sd *StateData
	var user *IdentifiedUser
	var userInfoClaims jwt.MapClaims
	var authority *authorities.Details

	for {
		sd, err = i.GetStateFromStateCookie(req.Context(), rw, req, "oauth2/cb", req.Form.Get("state"))
		if err != nil {
			err = fmt.Errorf("failed to decode oauth2 cb state: %w", err)
			break
		}
		if sd == nil {
			err = errors.New("state not found")
			break
		}

		// Load authority with client_id in state.
		authority, _ = i.authorities.Lookup(req.Context(), sd.Ref)
		if authority == nil {
			i.logger.WithField("client_id", sd.ClientID).Debugln("identifier failed to find authority in oauth2 cb")
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidRequest, "unknown client_id")
			break
		}

		if authority.AuthorityType != authorities.AuthorityTypeOIDC {
			err = errors.New("unknown authority type")
			break
		}

		// Check incoming state type.
		var done bool
		done, err = func() (bool, error) {
			switch sd.Mode {
			case StateModeEndSession:
				// Special mode. When in end session, take value from state and
				// redirect to it. This completes end session callback.
				uri, _ := url.Parse(sd.RawQuery)
				if uri == nil {
					return false, konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidRequest, "no uri in state")
				}
				if sd.State != "" {
					query := uri.Query()
					query.Set("state", sd.State)
					uri.RawQuery = query.Encode()
				}
				utils.WriteRedirect(rw, http.StatusFound, uri, nil, false)

				return true, nil
			default:
				// Continue further.
			}

			return false, nil
		}()
		if err != nil {
			break
		}
		if done {
			// Already done, nothing further so return.
			return
		}

		if authority.ResponseType == oidc.ResponseTypeCode ||
			authority.ResponseType == oidc.ResponseTypeCodeIDToken ||
			authority.ResponseType == oidc.ResponseTypeCodeIDTokenToken {
			// Exchange code for ID token.
			md := authority.Metadata().(*oidc.WellKnown)
			config := &oauth2.Config{
				ClientID:     authority.ClientID,
				ClientSecret: authority.ClientSecret,

				RedirectURL: i.oauth2CbEndpointURI.String(),

				Endpoint: oauth2.Endpoint{
					TokenURL: md.TokenEndpoint,
				},

				Scopes: authority.Scopes,
			}
			var httpClient *http.Client
			if authority.Insecure {
				httpClient = utils.InsecureHTTPClient
			} else {
				httpClient = utils.DefaultHTTPClient
			}
			t, exchangeErr := config.Exchange(
				context.WithValue(req.Context(), oauth2.HTTPClient, httpClient),
				req.Form.Get("code"),
				oauth2.SetAuthURLParam("code_verifier",
					sd.Extra["code_verifier"].(string)),
			)
			if exchangeErr != nil {
				err = fmt.Errorf("failed to exchange code for token: %w", exchangeErr)
				break
			}
			// Inject found data into request for later parse.
			req.Form.Set("access_token", t.AccessToken)
			req.Form.Set("token_type", t.TokenType)
			req.Form.Set("refresh_token", t.RefreshToken)
			if v, ok := t.Extra("expires_in").(string); ok {
				req.Form.Set("expires_in", v)
			}
			if v, ok := t.Extra("id_token").(string); ok {
				req.Form.Set("id_token", v)
			}
			// Fetch userinfo.
			uiReq, requestErr := http.NewRequest(http.MethodGet, md.UserInfoEndpoint, http.NoBody)
			if requestErr != nil {
				err = fmt.Errorf("failed to create userinfo request: %w", requestErr)
				break
			}
			t.SetAuthHeader(uiReq)
			uiResp, responseErr := httpClient.Do(uiReq)
			if responseErr != nil {
				err = fmt.Errorf("failed to get userinfo: %w", responseErr)
				break
			}
			// Decode userinfo as JSON, directly into the claims set.
			if decodeErr := json.NewDecoder(uiResp.Body).Decode(&userInfoClaims); decodeErr != nil {
				err = fmt.Errorf("failed to decode userinfo response: %w", decodeErr)
				uiResp.Body.Close()
				break
			}
			uiResp.Body.Close()
		}

		// Parse incoming state response.
		var authenticationSuccess *payload.AuthenticationSuccess
		if authenticationSuccessRaw, parseErr := authority.ParseStateResponse(req, sd.State, sd.Extra); parseErr == nil {
			authenticationSuccess = authenticationSuccessRaw.(*payload.AuthenticationSuccess)
		} else {
			err = parseErr
			break
		}

		// Parse and validate IDToken.
		idToken, idTokenParseErr := jwt.ParseWithClaims(authenticationSuccess.IDToken, userInfoClaims, authority.JWTKeyfunc())
		if idTokenParseErr != nil {
			if authority.Insecure {
				i.logger.WithField("client_id", sd.ClientID).WithError(idTokenParseErr).Warnln("identifier ignoring validation error for insecure authority")
			} else {
				i.logger.WithError(idTokenParseErr).Debugln("identifier failed to validate oauth2 cb id token")
				err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2ServerError, "authority response validation failed")
				break
			}
		}
		if claims, _ := idToken.Claims.(jwt.MapClaims); claims == nil {
			err = errors.New("invalid id token claims")
			break
		}

		// Lookup username and user.
		un, extra, claimsErr := authority.IdentityClaimValue(idToken)
		if claimsErr != nil {
			i.logger.WithError(claimsErr).Debugln("identifier failed to get username from oauth2 cb id token claims")
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InsufficientScope, "identity claim not found")
			break
		}

		username := &un

		// TODO(longsleep): This flow currently does not provide a hello
		// context, means that downwards a backend might fail to resolve the
		// user when it requires additional information for multiple backend
		// routing.
		user, err = i.resolveUser(req.Context(), *username)
		if err != nil {
			i.logger.WithError(err).WithField("username", *username).Debugln("identifier failed to resolve oauth2 cb user with backend")
			// TODO(longsleep): Break on validation error.
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2AccessDenied, "failed to resolve user")
			break
		}
		if user == nil || user.Subject() == "" {
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2AccessDenied, "no such user")
			break
		}

		var logonRef string
		if rawIDToken, ok := extra["RawIDToken"]; ok {
			logonRef = rawIDToken.(string)
		}
		if logonRef != "" {
			user.logonRef = &logonRef
		}

		// Get user meta data.
		// TODO(longsleep): This is an additional request to the backend. This
		// should be avoided. Best would be if the backend would return everything
		// in one shot (TODO in core).
		err = i.updateUser(req.Context(), user, authority)
		if err != nil {
			i.logger.WithError(err).Debugln("identifier failed to update user data in oauth2 cb request")
		}

		// Set logon time.
		user.logonAt = time.Now()

		err = i.SetUserToLogonCookie(req.Context(), rw, user)
		if err != nil {
			i.logger.WithError(err).Errorln("identifier failed to serialize logon ticket in oauth2 cb")
			i.ErrorPage(rw, http.StatusInternalServerError, "", "failed to serialize logon ticket")
			return
		}

		break
	}

	if sd == nil {
		i.logger.WithError(err).Debugln("identifier oauth2 cb without state")
		i.ErrorPage(rw, http.StatusBadRequest, "", "state not found")
		return
	}

	uri, _ := url.Parse(i.authorizationEndpointURI.String())
	query, _ := url.ParseQuery(sd.RawQuery)
	query.Del("flow")
	query.Set("identifier", MustBeSignedIn)
	if query.Get("prompt") == oidc.PromptSelectAccount {
		// Remove select_acount prompt for our secondary indentifier, it was
		// already processed by the external provider.
		query.Del("prompt")
	}

	switch typedErr := err.(type) {
	case nil:
		// breaks
	case *konnectoidc.OAuth2Error:
		// Pass along OAuth2 error.
		i.logger.WithFields(utils.ErrorAsFields(err)).Debugln("oauth2 cb error")
		// NOTE(longsleep): Pass along error ID but not the description to avoid
		// leaking potetially internal information to our RP.
		query.Set("error", typedErr.ErrorID)
		query.Set("error_description", "identifier failed to authenticate")
		//breaks
	default:
		i.logger.WithError(err).Errorln("identifier failed to process oauth2 cb")
		i.ErrorPage(rw, http.StatusInternalServerError, "", "oauth2 cb failed")
		return
	}

	uri.RawQuery = query.Encode()
	utils.WriteRedirect(rw, http.StatusFound, uri, nil, false)
}
