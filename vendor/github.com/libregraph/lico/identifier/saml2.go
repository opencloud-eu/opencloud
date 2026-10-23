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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/crewjam/saml"
	"github.com/libregraph/oidc-go"
	"github.com/longsleep/rndm"
	"github.com/sirupsen/logrus"

	"github.com/libregraph/lico/identity"
	"github.com/libregraph/lico/identity/authorities"
	konnectoidc "github.com/libregraph/lico/oidc"

	"github.com/libregraph/lico/identity/authorities/samlext"
	"github.com/libregraph/lico/utils"
)

func (i *Identifier) writeSAML2Start(rw http.ResponseWriter, req *http.Request, authority *authorities.Details) {
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
		i.logger.WithFields(utils.ErrorAsFields(err)).Debugln("saml2 start error")
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
		i.logger.WithError(err).Errorln("identifier failed to process saml2 start")
		i.ErrorPage(rw, http.StatusInternalServerError, "", "saml2 start failed")
		return
	}

	sd := &StateData{
		State:    rndm.GenerateRandomString(32),
		RawQuery: req.URL.RawQuery,

		Ref: authority.ID,
	}

	uri, extra, err := authority.MakeRedirectAuthenticationRequestURL(sd.State)
	if err != nil {
		i.logger.WithError(err).Errorln("identifier failed to create authentication request: %w", err)
		i.ErrorPage(rw, http.StatusInternalServerError, "", "saml2 start failed")
		return
	}
	sd.Extra = extra

	// Set cookie which is consumed by the callback later.
	err = i.SetStateToStateCookie(req.Context(), rw, "saml2/acs", sd)
	if err != nil {
		i.logger.WithError(err).Debugln("identifier failed to set saml2 state cookie")
		i.ErrorPage(rw, http.StatusInternalServerError, "", "failed to set cookie")
		return
	}

	utils.WriteRedirect(rw, http.StatusFound, uri, nil, false)
}

func (i *Identifier) writeSAML2AssertionConsumerService(rw http.ResponseWriter, req *http.Request) {
	var err error
	var sd *StateData
	var user *IdentifiedUser
	var authority *authorities.Details

	for {
		sd, err = i.GetStateFromStateCookie(req.Context(), rw, req, "saml2/acs", req.Form.Get("RelayState"))
		if err != nil {
			err = fmt.Errorf("failed to decode saml2 acs state: %v", err)
			break
		}
		if sd == nil {
			err = errors.New("state not found")
			break
		}

		// Load authority with client_id in state.
		authority, _ = i.authorities.Lookup(req.Context(), sd.Ref)
		if authority == nil {
			i.logger.Debugln("identifier failed to find authority in saml2 acs")
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidRequest, "unknown client_id")
			break
		}

		if authority.AuthorityType != authorities.AuthorityTypeSAML2 {
			err = errors.New("unknown authority type")
			break
		}

		// Parse incoming state response.
		var assertion *saml.Assertion
		if assertionRaw, parseErr := authority.ParseStateResponse(req, sd.State, sd.Extra); parseErr == nil {
			assertion = assertionRaw.(*saml.Assertion)
		} else {
			err = parseErr
			break
		}

		// Lookup username and user.
		un, claims, claimsErr := authority.IdentityClaimValue(assertion)
		if claimsErr != nil {
			i.logger.WithError(claimsErr).Debugln("identifier failed to get username from saml2 acs assertion")
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
			i.logger.WithError(err).WithField("username", *username).Debugln("identifier failed to resolve saml2 acs user with backend")
			// TODO(longsleep): Break on validation error.
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2AccessDenied, "failed to resolve user")
			break
		}
		if user == nil || user.Subject() == "" {
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2AccessDenied, "no such user")
			break
		}

		// Apply additional authority claims.
		if sessionNotOnOrAfter, ok := claims["SessionNotOnOrAfter"]; ok {
			user.expiresAfter = sessionNotOnOrAfter.(*time.Time)
		}
		var logonRef string
		if nameIDTransient, ok := claims["TransientNameID"]; ok {
			logonRef = "transient:" + nameIDTransient.(string)
		} else if nameIDPersistent, ok := claims["PersistentNameID"]; ok {
			logonRef = "persistent:" + nameIDPersistent.(string)
		} else if nameIDUnspecified, ok := claims["UnspecifiedNameID"]; ok {
			logonRef = "unspecified:" + nameIDUnspecified.(string)
		}
		if logonRef != "" {
			user.logonRef = &logonRef
		}
		if authority.Trusted {
			// Use external authority session, if the external authority is
			// trusted. It is tagged, since it is not a ref of our backend and
			// must never be sent to one.
			if sessionIndexString, ok := claims["SessionIndex"]; ok {
				user.sessionRef = identity.GetExternalSessionRef(sessionIndexString.(string))
			}
		}

		// Get user meta data.
		// TODO(longsleep): This is an additional request to the backend. This
		// should be avoided. Best would be if the backend would return everything
		// in one shot (TODO in core).
		err = i.updateUser(req.Context(), user, authority)
		if err != nil {
			i.logger.WithError(err).Debugln("identifier failed to get user data in saml2 acs request")
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2AccessDenied, "failed to get user data")
			break
		}

		// Set logon time.
		user.logonAt = time.Now()

		err = i.SetUserToLogonCookie(req.Context(), rw, user)
		if err != nil {
			i.logger.WithError(err).Errorln("identifier failed to serialize logon ticket in saml2 acs")
			i.ErrorPage(rw, http.StatusInternalServerError, "", "failed to serialize logon ticket")
			return
		}

		break
	}

	if sd == nil {
		i.logger.WithError(err).Debugln("identifier saml2 acs without state")
		i.ErrorPage(rw, http.StatusBadRequest, "", "state not found")
		return
	}

	uri, _ := url.Parse(i.authorizationEndpointURI.String())
	query, _ := url.ParseQuery(sd.RawQuery)
	query.Del("flow")
	query.Set("identifier", MustBeSignedIn)
	query.Set("prompt", oidc.PromptNone)

	switch typedErr := err.(type) {
	case nil:
		// breaks
	case *saml.InvalidResponseError:
		i.logger.WithError(err).WithFields(logrus.Fields{
			"reason": typedErr.PrivateErr,
		}).Debugf("saml2 acs invalid response")
		query.Set("error", oidc.ErrorCodeOAuth2AccessDenied)
		query.Set("error_description", "identifier received invalid response")
		// breaks
	case *konnectoidc.OAuth2Error:
		// Pass along OAuth2 error.
		i.logger.WithFields(utils.ErrorAsFields(err)).Debugln("saml2 acs error")
		// NOTE(longsleep): Pass along error ID but not the description to avoid
		// leaking potetially internal information to our RP.
		query.Set("error", typedErr.ErrorID)
		query.Set("error_description", "identifier failed to authenticate")
		//breaks
	default:
		i.logger.WithError(err).Errorln("identifier failed to process saml2 acs")
		i.ErrorPage(rw, http.StatusInternalServerError, "", "saml2 acs failed")
		return
	}

	uri.RawQuery = query.Encode()
	utils.WriteRedirect(rw, http.StatusFound, uri, nil, false)
}

func (i *Identifier) writeSAMLSingleLogoutServiceRequest(rw http.ResponseWriter, req *http.Request) {
	lor, err := samlext.NewIdpLogoutRequest(req)
	if err != nil {
		i.logger.WithError(err).Debugln("identifier failed to process saml2 slo request")
		i.ErrorPage(rw, http.StatusBadRequest, "", "failed to parse request")
		return
	}

	err = lor.Validate()
	if err != nil {
		i.logger.WithError(err).Debugln("identifier saml2 slo request validation failed")
		i.ErrorPage(rw, http.StatusBadRequest, "", "slo request validation failed")
		return
	}

	// In http://docs.oasis-open.org/security/saml/v2.0/saml-bindings-2.0-os.pdf §3.4.5.2
	// we get a description of the Destination attribute:
	//
	//   If the message is signed, the Destination XML attribute in the root SAML
	//   element of the protocol message MUST contain the URL to which the sender
	//   has instructed the user agent to deliver the message. The recipient MUST
	//   then verify that the value matches the location at which the message has
	//   been received.
	//
	// We require the destination be correct either (a) if signing is enabled or
	// (b) if it was provided.
	mustHaveDestination := lor.SigAlg != nil
	mustHaveDestination = mustHaveDestination || lor.Request.Destination != ""
	if mustHaveDestination {
		uri, _ := i.absoluteURLForRoute("saml2/slo")
		if lor.Request.Destination != uri.String() {
			i.logger.WithField("destination", lor.Request.Destination).Debugln("identifier saml2 slo request with wrong desitation")
			i.ErrorPage(rw, http.StatusBadRequest, "", "slo request destination wrong")
			return
		}
	}

	// Find matching authority.
	authority, found := i.authorities.Find(req.Context(), func(authority authorities.AuthorityRegistration) bool {
		if authority.AuthorityType() != authorities.AuthorityTypeSAML2 {
			return false
		}
		if lor.Request.Issuer.Value == authority.Issuer() {
			return true
		}
		return false
	})
	if !found {
		i.logger.WithField("issuer", lor.Request.Issuer.Value).Debugln("identifier saml2 slo request from unknown issuer")
		i.ErrorPage(rw, http.StatusBadRequest, "", "slo request issuer unknown")
		return
	}

	authorityDetails := authority.Authority()
	if lor.SigAlg == nil {
		// Never consider trusted if not signed.
		authorityDetails.Trusted = false
	}

	if authorityDetails.AuthorityType != authorities.AuthorityTypeSAML2 {
		i.logger.WithField("issuer", lor.Request.Issuer.Value).Debugln("identifier saml2 slo request for unknown authority type")
		i.ErrorPage(rw, http.StatusBadRequest, "", "slo request issuer authority type unknown")
		return
	}

	// Validate.
	validated, err := authority.ValidateIdpEndSessionRequest(lor, lor.RelayState)
	if err != nil {
		i.logger.WithError(err).WithField("issuer", authority.Issuer()).Debugln("identifier saml2 slo request authority validation failed")
		i.ErrorPage(rw, http.StatusBadRequest, "", "slo request authority validation failed")
		return
	}
	if !validated && authorityDetails.Trusted {
		// Never consider unvalidated logout requests as trusted.
		authorityDetails.Trusted = false
	}

	user, _ := i.GetUserFromLogonCookie(req.Context(), req, 0, false)
	if user != nil {
		// Compare the signed in SAML SessionIndex with the one provided in the
		// LogoutRequest.
		if sessionRef := user.SessionRef(); sessionRef != nil {
			if sessionIndex, isExternal := identity.ParseExternalSessionRef(*sessionRef); isExternal {
				if lor.Request.SessionIndex == nil {
					i.logger.Debugln("identifier saml2 slo request without session index")
					i.ErrorPage(rw, http.StatusBadRequest, "", "slo request missing session index")
					return
				}
				if lor.Request.SessionIndex.Value != sessionIndex {
					i.logger.Debugln("identifier saml2 slo request for other session index")
					i.ErrorPage(rw, http.StatusBadRequest, "", "slo request session index mismatch")
					return
				}
			}
		}

		if authorityDetails != nil && authorityDetails.Trusted {
			// Directly clear identifier session when a trusted authority requests it.
			err = i.UnsetLogonCookie(req.Context(), user, rw)
			if err != nil {
				i.logger.WithError(err).Errorln("identifier saml2 slo failed to unset logon cookie")
				i.ErrorPage(rw, http.StatusInternalServerError, "", "saml2 slo logout failed")
				return
			}
		}
	} else {
		// Ignore when not signed in, for end session.
	}

	if authorityDetails == nil || !authorityDetails.Trusted {
		// Handle directly by redirecting to our logout confirm url for untrusted
		// registies or when no URL was set.
		uri, _ := i.absoluteURLForRoute("goodbye")
		query := &url.Values{}

		uri.RawQuery = query.Encode()
		utils.WriteRedirect(rw, http.StatusFound, uri, nil, false)
		return
	}

	uri, _, err := authorityDetails.MakeRedirectEndSessionResponseURL(lor.Request, lor.RelayState)
	if err != nil {
		i.logger.WithError(err).Errorln("failed to make saml2 slo redirect request url")
		i.ErrorPage(rw, http.StatusInternalServerError, "", "saml2 slo failed")
		return
	}
	if uri == nil {
		i.logger.Warnln("saml2 slo reached dead end, no post logout redirect uri available")
		// Fall back to logout confirm url.
		uri, _ = i.absoluteURLForRoute("goodbye")
	}

	utils.WriteRedirect(rw, http.StatusFound, uri, nil, false)
}

func (i *Identifier) writeSAMLSingleLogoutServiceResponse(rw http.ResponseWriter, req *http.Request) {
	lor, err := samlext.NewIdpLogoutResponse(req)
	if err != nil {
		i.logger.WithError(err).Debugln("identifier failed to process saml2 slo response")
		i.ErrorPage(rw, http.StatusBadRequest, "", "failed to parse response")
		return
	}

	err = lor.Validate()
	if err != nil {
		i.logger.WithError(err).Debugln("identifier saml2 slo response validation failed")
		i.ErrorPage(rw, http.StatusBadRequest, "", "response validation failed")
		return
	}

	sd, err := i.GetStateFromStateCookie(req.Context(), rw, req, "_/saml2/slo", lor.RelayState)
	if err != nil {
		i.logger.WithError(err).Debugln("identifier saml2 slo response failed to load state")
		i.ErrorPage(rw, http.StatusBadRequest, "", "response state invalid")
		return
	}
	if sd == nil {
		i.logger.WithError(err).Debugln("identifier saml2 slo response failed as state is missing")
		i.ErrorPage(rw, http.StatusBadRequest, "", "response state missing")
		return
	}

	authority, found := i.authorities.Get(req.Context(), sd.Ref)
	if !found {
		i.ErrorPage(rw, http.StatusBadRequest, "", "no authority")
		return
	}

	authorityDetails := authority.Authority()
	if lor.SigAlg == nil {
		// Never consider trusted if not signed.
		authorityDetails.Trusted = false
	}

	if authorityDetails.AuthorityType != authorities.AuthorityTypeSAML2 {
		i.logger.WithField("issuer", authority.Issuer()).Debugln("identifier saml2 slo response for unknown authority type")
		i.ErrorPage(rw, http.StatusBadRequest, "", "slo response issuer authority type unknown")
		return
	}

	// Validate.
	validated, err := authority.ValidateIdpEndSessionResponse(lor, lor.RelayState)
	if err != nil {
		i.logger.WithError(err).WithField("issuer", authority.Issuer()).Debugln("identifier saml2 slo response authority validation failed")
		i.ErrorPage(rw, http.StatusBadRequest, "", "slo response authority validation failed")
		return
	}
	if !validated && authorityDetails.Trusted {
		// Never consider unvalidated logout responses as trusted.
		authorityDetails.Trusted = false
	}

	if lor.Response.Status.StatusCode.Value != saml.StatusSuccess {
		i.logger.WithField("status", lor.Response.Status.StatusCode).Debugln("saml2 slo response without success status")
	}

	// Extract destination URI from state data (its put into the RawQuery field).
	uri, err := url.Parse(sd.RawQuery)
	if err != nil {
		i.logger.WithError(err).Errorln("failed to parse slo response redirect url from state data")
		i.ErrorPage(rw, http.StatusInternalServerError, "", "saml2 slo response failed")
		return
	}
	if uri == nil || uri.String() == "" {
		i.logger.Warnln("saml2 slo reached dead end, no post logout redirect uri available")
		// Fall back to our signed out url or goodbye route.
		if i.Config.SignedOutEndpointURI != nil {
			uri = i.Config.SignedOutEndpointURI
		} else {
			uri, _ = i.absoluteURLForRoute("goodbye")
		}
	}
	if sd.State != "" {
		query := uri.Query()
		query.Set("state", sd.State)
		uri.RawQuery = query.Encode()
	}

	utils.WriteRedirect(rw, http.StatusFound, uri, nil, false)
}
