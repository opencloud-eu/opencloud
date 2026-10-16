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

package samlext

import (
	"crypto"

	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/crewjam/saml"
)

// IdpLogoutRequest is used by IdentityProvider to handle a single logout request.
type IdpLogoutRequest struct {
	HTTPRequest *http.Request

	Binding       string
	RequestBuffer []byte
	Request       *saml.LogoutRequest
	Now           time.Time

	RelayState string
	SigAlg     *string
	Signature  []byte
}

func NewIdpLogoutRequest(r *http.Request) (*IdpLogoutRequest, error) {
	req := &IdpLogoutRequest{
		HTTPRequest: r,
		Now:         saml.TimeNow(),
	}

	switch r.Method {
	case http.MethodGet:
		req.Binding = saml.HTTPRedirectBinding

		compressedRequest, err := base64.StdEncoding.DecodeString(r.URL.Query().Get("SAMLRequest"))
		if err != nil {
			return nil, fmt.Errorf("cannot decode request: %w", err)
		}
		req.RequestBuffer, err = inflateMessage(compressedRequest)
		if err != nil {
			return nil, fmt.Errorf("cannot decompress request: %w", err)
		}
		req.RelayState = r.URL.Query().Get("RelayState")

		sigAlgRaw := r.URL.Query().Get("SigAlg")
		if sigAlgRaw != "" {
			req.SigAlg = &sigAlgRaw

			signature, err := base64.StdEncoding.DecodeString(r.URL.Query().Get("Signature"))
			if err != nil {
				return nil, fmt.Errorf("cannot decode signature: %w", err)
			}
			req.Signature = signature
		}

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			return nil, err
		}

		req.Binding = saml.HTTPPostBinding

		var err error
		req.RequestBuffer, err = base64.StdEncoding.DecodeString(r.PostForm.Get("SAMLRequest"))
		if err != nil {
			return nil, err
		}
		req.RelayState = r.PostForm.Get("RelayState")

		return nil, fmt.Errorf("parsing logout request from POST is not implemented")
	default:
		return nil, fmt.Errorf("method not allowed")
	}
	return req, nil
}

// Validate checks that the authentication request is valid and assigns
// the LogoutRequest and Metadata properties. Returns a non-nil error if the
// request is not valid.
func (req *IdpLogoutRequest) Validate() error {
	request := &saml.LogoutRequest{}
	if err := xml.Unmarshal(req.RequestBuffer, request); err != nil {
		return err
	}
	req.Request = request

	if req.Request.IssueInstant.Add(saml.MaxIssueDelay).Before(req.Now) {
		return fmt.Errorf("request expired at %s", req.Request.IssueInstant.Add(saml.MaxIssueDelay))
	}
	if req.Request.Version != "2.0" {
		return fmt.Errorf("expected SAML request version 2.0 got %v", req.Request.Version)
	}

	return nil
}

// VerifySignature verifies the associated IdpLogoutRequest data with the
// associated Signature using the provided public key.
func (req *IdpLogoutRequest) VerifySignature(pubKey crypto.PublicKey) error {
	return VerifySignedHTTPRedirectQuery("SAMLRequest", req.HTTPRequest.URL.RawQuery, *req.SigAlg, req.Signature, pubKey)
}
