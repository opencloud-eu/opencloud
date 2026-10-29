/*
 * Copyright 2026 Kopano and its licensors
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

package oidc

import (
	oidcgo "github.com/libregraph/oidc-go"
)

// ProviderMetadata is the OpenID Connect Discovery 1.0 provider meta data
// document as specified at
// https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderMetadata,
// extended with the OAuth 2.0 Authorization Server Metadata (RFC 8414) fields
// this provider supports but oidcgo.WellKnown does not define. The embedded
// document is a pointer so this type stays a view onto an existing meta data
// document rather than a copy of it, and the fields it defines are promoted
// into the same flat JSON object when serialized.
//
// The fields below are the ones a client needs to use this provider with the
// Model Context Protocol authorization flow, which requires OAuth 2.1 with
// PKCE and rejects a provider that does not advertise its code challenge
// methods.
type ProviderMetadata struct {
	*oidcgo.WellKnown

	// GrantTypesSupported is https://datatracker.ietf.org/doc/html/rfc8414#section-2.
	// Omitting it means authorization_code and implicit per OpenID Connect
	// Discovery 1.0, which is not what the token endpoint implements.
	GrantTypesSupported []string `json:"grant_types_supported,omitempty"`

	// ResponseModesSupported is https://datatracker.ietf.org/doc/html/rfc8414#section-2.
	ResponseModesSupported []string `json:"response_modes_supported,omitempty"`

	// CodeChallengeMethodsSupported is https://datatracker.ietf.org/doc/html/rfc8414#section-2.
	// A client which requires PKCE has no other way to discover support for it
	// and must refuse to continue when this is absent.
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported,omitempty"`

	// AuthorizationResponseIssParameterSupported is
	// https://datatracker.ietf.org/doc/html/rfc9207#section-2.3. It must only
	// be set when authorization responses actually carry the iss parameter.
	AuthorizationResponseIssParameterSupported bool `json:"authorization_response_iss_parameter_supported,omitempty"`

	// ClientIDMetadataDocumentSupported announces that a client may use the
	// HTTPS URL of its own metadata document as client_id, as specified at
	// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-client-id-metadata-document-02.
	ClientIDMetadataDocumentSupported bool `json:"client_id_metadata_document_supported,omitempty"`
}
