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

// IdpLogoutResponse is used by IdentityProvider to handle a single logout
// response callbacks.
type IdpLogoutResponse struct {
	HTTPRequest *http.Request

	Binding        string
	ResponseBuffer []byte
	Response       *saml.LogoutResponse
	Now            time.Time

	RelayState string
	SigAlg     *string
	Signature  []byte
}

func NewIdpLogoutResponse(r *http.Request) (*IdpLogoutResponse, error) {
	res := &IdpLogoutResponse{
		HTTPRequest: r,
		Now:         saml.TimeNow(),
	}

	switch r.Method {
	case http.MethodGet:
		res.Binding = saml.HTTPRedirectBinding

		compressedResponse, err := base64.StdEncoding.DecodeString(r.URL.Query().Get("SAMLResponse"))
		if err != nil {
			return nil, fmt.Errorf("cannot decode response: %w", err)
		}
		res.ResponseBuffer, err = inflateMessage(compressedResponse)
		if err != nil {
			return nil, fmt.Errorf("cannot decompress response: %w", err)
		}
		res.RelayState = r.URL.Query().Get("RelayState")

		sigAlgRaw := r.URL.Query().Get("SigAlg")
		if sigAlgRaw != "" {
			res.SigAlg = &sigAlgRaw

			signature, err := base64.StdEncoding.DecodeString(r.URL.Query().Get("Signature"))
			if err != nil {
				return nil, fmt.Errorf("cannot decode signature: %w", err)
			}
			res.Signature = signature
		}

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			return nil, err
		}

		res.Binding = saml.HTTPPostBinding

		var err error
		res.ResponseBuffer, err = base64.StdEncoding.DecodeString(r.PostForm.Get("SAMLResponse"))
		if err != nil {
			return nil, err
		}
		res.RelayState = r.PostForm.Get("RelayState")

		return nil, fmt.Errorf("parsing logout response from POST is not implemented")
	default:
		return nil, fmt.Errorf("method not allowed")
	}
	return res, nil
}

// Validate checks that the associated response is valid and assigns
// the LogoutResponse and Metadata properties. Returns a non-nil error if the
// request is not valid.
func (res *IdpLogoutResponse) Validate() error {
	response := &saml.LogoutResponse{}
	if err := xml.Unmarshal(res.ResponseBuffer, response); err != nil {
		return err
	}
	res.Response = response

	if res.Response.IssueInstant.Add(saml.MaxIssueDelay).Before(res.Now) {
		return fmt.Errorf("response expired at %s", res.Response.IssueInstant.Add(saml.MaxIssueDelay))
	}
	if res.Response.Version != "2.0" {
		return fmt.Errorf("expected SAML response version 2.0 got %v", res.Response.Version)
	}

	return nil
}

// VerifySignature verifies the associated IdpLogoutResponse data with the
// associated Signature using the provided public key.
func (res *IdpLogoutResponse) VerifySignature(pubKey crypto.PublicKey) error {
	return VerifySignedHTTPRedirectQuery("SAMLResponse", res.HTTPRequest.URL.RawQuery, *res.SigAlg, res.Signature, pubKey)
}
