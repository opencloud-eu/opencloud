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
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	konnect "github.com/libregraph/lico"
	"github.com/libregraph/lico/identity/authorities"
	"github.com/libregraph/lico/utils"
)

// staticHandler serves the built web app assets. Everything under the assets
// folder carries a content hash in its name, so it can be cached indefinitely.
// The rest of the static folder is addressed by a stable name a deployment is
// meant to be able to replace, so it gets a lifetime short enough that a
// replacement is picked up without a rebuild, a restart or a cache clear.
func (i *Identifier) staticHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		addCommonResponseHeaders(rw.Header())
		if strings.Contains(req.URL.Path, "/static/assets/") {
			rw.Header().Set("Cache-Control", "max-age=3153600, public")
		} else {
			rw.Header().Set("Cache-Control", "max-age=300, public")
		}
		if strings.HasSuffix(req.URL.Path, "/") {
			// Do not serve folder-ish resources.
			i.ErrorPage(rw, http.StatusNotFound, "", "")
			return
		}
		handler.ServeHTTP(rw, req)
	})
}

func (i *Identifier) secureHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		var err error

		// TODO(longsleep): Add support for X-Forwareded-Host with trusted proxy.
		// NOTE: this does not protect from DNS rebinding. Protection for that
		// should be added at the frontend proxy.
		requiredHost := req.Host
		if host, port, splitErr := net.SplitHostPort(requiredHost); splitErr == nil {
			if port == "443" {
				// Ignore the port 443 as it is the default port and it is
				// usually not part of any of the urls. It might be in the
				// request for HTTP/3 requests.
				requiredHost = host
			}
		}

		// This follows https://www.owasp.org/index.php/Cross-Site_Request_Forgery_(CSRF)_Prevention_Cheat_Sheet
		for {
			if req.Header.Get("Kopano-Konnect-XSRF") != "1" {
				err = fmt.Errorf("missing xsrf header")
				break
			}

			origin := req.Header.Get("Origin")
			referer := req.Header.Get("Referer")

			// Require either Origin and Referer header.
			// NOTE(longsleep): Firefox does not send Origin header for POST
			// requests when on the same domain - this is fuck (tm). See
			// https://bugzilla.mozilla.org/show_bug.cgi?id=446344 for reference.
			if origin == "" && referer == "" {
				err = fmt.Errorf("missing origin or referer header")
				break
			}

			if origin != "" {
				originURL, urlParseErr := url.Parse(origin)
				if urlParseErr != nil {
					err = fmt.Errorf("invalid origin value: %v", urlParseErr)
					break
				}
				if originURL.Host != requiredHost {
					err = fmt.Errorf("origin does not match request URL")
					break
				}
			} else if referer != "" {
				refererURL, urlParseErr := url.Parse(referer)
				if urlParseErr != nil {
					err = fmt.Errorf("invalid referer value: %v", urlParseErr)
					break
				}
				if refererURL.Host != requiredHost {
					err = fmt.Errorf("referer does not match request URL")
					break
				}
			} else {
				i.logger.WithFields(logrus.Fields{
					"host":       requiredHost,
					"user-agent": req.UserAgent(),
				}).Warn("identifier HTTP request is insecure with no Origin and Referer")
			}

			handler.ServeHTTP(rw, req)
			return
		}

		if err != nil {
			i.logger.WithError(err).WithFields(logrus.Fields{
				"host":       requiredHost,
				"referer":    req.Referer(),
				"user-agent": req.UserAgent(),
				"origin":     req.Header.Get("Origin"),
			}).Warn("rejecting identifier HTTP request")
		}

		i.ErrorPage(rw, http.StatusBadRequest, "", "")
	})
}

func (i *Identifier) handleIdentifier(rw http.ResponseWriter, req *http.Request) {
	addCommonResponseHeaders(rw.Header())
	addNoCacheResponseHeaders(rw.Header())

	err := req.ParseForm()
	if err != nil {
		i.logger.WithError(err).Debugln("identifier failed to decode request")
		i.ErrorPage(rw, http.StatusBadRequest, "", "failed to decode request")
		return
	}

	switch req.Form.Get("flow") {
	case FlowOIDC, FlowOAuth, "":
		if req.Form.Get("identifier") != MustBeSignedIn {
			//  Check if there is a default authority, if so use that.
			authority := i.authorities.Default(req.Context())
			if authority != nil {
				switch authority.AuthorityType {
				case authorities.AuthorityTypeOIDC:
					i.writeOAuth2Start(rw, req, authority)
				case authorities.AuthorityTypeSAML2:
					i.writeSAML2Start(rw, req, authority)
				default:
					i.ErrorPage(rw, http.StatusNotImplemented, "", "unknown authority type")
				}
				return
			}
		}
	}

	// Show default.
	i.writeWebappIndexHTML(rw, req)
}

func (i *Identifier) handleLogon(rw http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var r LogonRequest
	err := decoder.Decode(&r)
	if err != nil {
		i.logger.WithError(err).Debugln("identifier failed to decode logon request")
		i.ErrorPage(rw, http.StatusBadRequest, "", "failed to decode request JSON")
		return
	}

	var user *IdentifiedUser
	response := &LogonResponse{
		State: r.State,
	}

	addNoCacheResponseHeaders(rw.Header())

	record := NewRecord(req, i.Config.Config)

	if r.Hello != nil {
		err = r.Hello.parse()
		if err != nil {
			i.logger.WithError(err).Debugln("identifier failed to parse logon request hello")
			i.ErrorPage(rw, http.StatusBadRequest, "", "failed to parse request values")
			return
		}
		record.HelloRequest = r.Hello
	}

	req = req.WithContext(NewRecordContext(konnect.NewRequestContext(req.Context(), req), record))

	// Params is an array like this [$username, $password, $mode], defining a
	// extensible way to extend login modes over time. The minimal length of
	// the params array is 1 with only [$username]. Second field is the password
	// but its interpretation depends on the third field ($mode). The rest of the
	// fields are mode specific.
	params := r.Params
	for {
		paramSize := len(params)
		if paramSize == 0 {
			i.ErrorPage(rw, http.StatusBadRequest, "", "params required")
			return
		}

		if paramSize >= 3 && params[1] == "" && params[2] == ModeLogonUsernameEmptyPasswordCookie {
			// Special mode to allow when same user is logged in via cookie. This
			// is used in the select account page logon flow with empty password.
			identifiedUser, cookieErr := i.GetUserFromLogonCookie(req.Context(), req, 0, true)
			if cookieErr != nil {
				i.logger.WithError(cookieErr).Debugln("identifier failed to decode logon cookie in logon request")
			}
			if identifiedUser != nil {
				if identifiedUser.Username() == params[0] {
					user = identifiedUser
					break
				}
			}
		}

		audience := ""
		if r.Hello != nil {
			audience = r.Hello.ClientID
		}

		if paramSize < 3 {
			// Unsupported logon mode.
			break
		}
		if params[1] == "" {
			// Empty password, stop here - never allowed in any mode.
			break
		}

		switch params[2] {
		case ModeLogonUsernamePassword:
			// Username and password validation mode.
			logonedUser, logonErr := i.logonUser(req.Context(), audience, params[0], params[1])
			if logonErr != nil {
				i.logger.WithError(logonErr).Errorln("identifier failed to logon with backend")
				i.ErrorPage(rw, http.StatusInternalServerError, "", "failed to logon")
				return
			}
			user = logonedUser

		default:
			i.logger.WithField("mode", params[2]).Debugln("identifier unknown logon mode")
		}

		break
	}

	if user == nil || user.Subject() == "" {
		rw.Header().Set("Kopano-Konnect-State", response.State)
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	// Get user meta data.
	// TODO(longsleep): This is an additional request to the backend. This
	// should be avoided. Best would be if the backend would return everything
	// in one shot (TODO in core).
	err = i.updateUser(req.Context(), user, nil)
	if err != nil {
		i.logger.WithError(err).Debugln("identifier failed to update user data in logon request")
	}

	// Set logon time.
	user.logonAt = time.Now()

	if r.Hello != nil {
		hello, errHello := i.writeHelloResponse(rw, req, r.Hello, user)
		if errHello != nil {
			i.logger.WithError(errHello).Debugln("rejecting identifier logon request")
			i.ErrorPage(rw, http.StatusBadRequest, "", errHello.Error())
			return
		}
		if !hello.Success {
			rw.Header().Set("Kopano-Konnect-State", response.State)
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		response.Hello = hello
	}

	err = i.SetUserToLogonCookie(req.Context(), rw, user)
	if err != nil {
		i.logger.WithError(err).Errorln("failed to serialize logon ticket")
		i.ErrorPage(rw, http.StatusInternalServerError, "", "failed to serialize logon ticket")
		return
	}

	response.Success = true

	err = utils.WriteJSON(rw, http.StatusOK, response, "")
	if err != nil {
		i.logger.WithError(err).Errorln("logon request failed writing response")
	}
}

func (i *Identifier) handleLogoff(rw http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var r StateRequest
	err := decoder.Decode(&r)
	if err != nil {
		i.logger.WithError(err).Debugln("identifier failed to decode logoff request")
		i.ErrorPage(rw, http.StatusBadRequest, "", "failed to decode request JSON")
		return
	}

	addNoCacheResponseHeaders(rw.Header())

	ctx := req.Context()
	u, err := i.GetUserFromLogonCookie(ctx, req, 0, false)
	if err != nil {
		i.logger.WithError(err).Warnln("identifier logoff failed to get logon from ticket")
	}
	err = i.UnsetLogonCookie(ctx, u, rw)
	if err != nil {
		i.logger.WithError(err).Errorln("identifier failed to set logoff ticket")
		i.ErrorPage(rw, http.StatusInternalServerError, "", "failed to set logoff ticket")
		return
	}

	response := &StateResponse{
		State:   r.State,
		Success: true,
	}

	err = utils.WriteJSON(rw, http.StatusOK, response, "")
	if err != nil {
		i.logger.WithError(err).Errorln("logoff request failed writing response")
	}
}

func (i *Identifier) handleConsent(rw http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var r ConsentRequest
	err := decoder.Decode(&r)
	if err != nil {
		i.logger.WithError(err).Debugln("identifier failed to decode consent request")
		i.ErrorPage(rw, http.StatusBadRequest, "", "failed to decode request JSON")
		return
	}

	addNoCacheResponseHeaders(rw.Header())

	consent := &Consent{
		Allow: r.Allow,
	}
	if r.Allow {
		consent.RawScope = r.RawScope
	}

	err = i.SetConsentToConsentCookie(req.Context(), rw, &r, consent)
	if err != nil {
		i.logger.WithError(err).Errorln("failed to serialize consent ticket")
		i.ErrorPage(rw, http.StatusInternalServerError, "", "failed to serialize consent ticket")
		return
	}

	if !r.Allow {
		rw.Header().Set("Kopano-Konnect-State", r.State)
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	response := &StateResponse{
		State:   r.State,
		Success: true,
	}

	err = utils.WriteJSON(rw, http.StatusOK, response, "")
	if err != nil {
		i.logger.WithError(err).Errorln("logoff request failed writing response")
	}
}

func (i *Identifier) handleHello(rw http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var r HelloRequest
	err := decoder.Decode(&r)
	if err != nil {
		i.logger.WithError(err).Debugln("identifier failed to decode hello request")
		i.ErrorPage(rw, http.StatusBadRequest, "", "failed to decode request JSON")
		return
	}
	err = r.parse()
	if err != nil {
		i.logger.WithError(err).Debugln("identifier failed to parse hello request")
		i.ErrorPage(rw, http.StatusBadRequest, "", "failed to parse request values")
		return
	}

	addNoCacheResponseHeaders(rw.Header())

	response, err := i.writeHelloResponse(rw, req, &r, nil)
	if err != nil {
		i.logger.WithError(err).Debugln("rejecting identifier hello request")
		i.ErrorPage(rw, http.StatusBadRequest, "", err.Error())
		return
	}

	err = utils.WriteJSON(rw, http.StatusOK, response, "")
	if err != nil {
		i.logger.WithError(err).Errorln("hello request failed writing response")
	}
}

func (i *Identifier) handleTrampolin(rw http.ResponseWriter, req *http.Request) {
	if !strings.HasSuffix(req.URL.Path, ".js") {
		err := req.ParseForm()
		if err != nil {
			i.logger.WithError(err).Debugln("identifier failed to decode trampolin request")
			i.ErrorPage(rw, http.StatusBadRequest, "", "failed to decode request parameters")
			return
		}

		sd, err := i.GetStateFromStateCookie(req.Context(), rw, req, "trampolin", req.Form.Get("state"))
		if err != nil {
			i.ErrorPage(rw, http.StatusBadRequest, "", err.Error())
			return
		}
		if sd == nil || sd.Trampolin == nil {
			i.ErrorPage(rw, http.StatusBadRequest, "", "no state")
			return
		}

		scope := sd.Trampolin.Scope
		uri, _ := url.Parse(sd.Trampolin.URI)
		sd.Trampolin = nil

		err = i.SetStateToStateCookie(req.Context(), rw, scope, sd)
		if err != nil {
			i.logger.WithError(err).Errorln("failed to write trampolin state cookie")
			i.ErrorPage(rw, http.StatusInternalServerError, "", "failed to write trampolin state cookie")
			return
		}

		i.writeTrampolinHTML(rw, req, uri)
	} else {
		i.writeTrampolinScript(rw, req)
	}
}

func (i *Identifier) handleOAuth2Start(rw http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		i.logger.WithError(err).Debugln("identifier failed to decode oauth2 start request")
		i.ErrorPage(rw, http.StatusBadRequest, "", "failed to decode request parameters")
		return
	}

	var authority *authorities.Details
	if authorityID := req.Form.Get("authority_id"); authorityID != "" {
		authority, _ = i.authorities.Lookup(req.Context(), authorityID)
	}

	i.writeOAuth2Start(rw, req, authority)
}

func (i *Identifier) handleOAuth2Cb(rw http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		i.logger.WithError(err).Debugln("identifier failed to decode oauth2 cb request")
		i.ErrorPage(rw, http.StatusBadRequest, "", "failed to decode request parameters")
		return
	}

	i.writeOAuth2Cb(rw, req)
}

func (i *Identifier) handleSAML2Metadata(rw http.ResponseWriter, req *http.Request) {
	authorityDetails := i.authorities.Default(req.Context())
	if authorityDetails == nil || authorityDetails.AuthorityType != authorities.AuthorityTypeSAML2 {
		i.ErrorPage(rw, http.StatusNotFound, "", "saml not configured")
		return
	}

	metadata := authorityDetails.Metadata()
	if metadata == nil {
		i.ErrorPage(rw, http.StatusNotFound, "", "saml has no meta data")
		return
	}

	buf, _ := xml.MarshalIndent(metadata, "", "  ")
	rw.Header().Set("Content-Type", "application/samlmetadata+xml")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(xml.Header))
	rw.Write(buf)
}

func (i *Identifier) handleSAML2AssertionConsumerService(rw http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		i.logger.WithError(err).Debugln("identifier failed to decode saml2 acs request")
		i.ErrorPage(rw, http.StatusBadRequest, "", "failed to decode request parameters")
		return
	}

	i.writeSAML2AssertionConsumerService(rw, req)
}

func (i *Identifier) handleSAML2SingleLogoutService(rw http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		i.logger.WithError(err).Debugln("identifier failed to decode saml2 slo request")
		i.ErrorPage(rw, http.StatusBadRequest, "", "failed to decode request parameters")
		return
	}

	if _, ok := req.Form["SAMLRequest"]; ok {
		i.writeSAMLSingleLogoutServiceRequest(rw, req)
	} else if _, ok := req.Form["SAMLResponse"]; ok {
		i.writeSAMLSingleLogoutServiceResponse(rw, req)
	} else {
		i.ErrorPage(rw, http.StatusBadRequest, "", "neither SAMLRequest nor SAMLResponse parameter found")
	}
}
