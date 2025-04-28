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

package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/longsleep/rndm"

	konnect "github.com/libregraph/lico"
	"github.com/libregraph/lico/identity"
	"github.com/libregraph/lico/oidc"
	"github.com/libregraph/lico/oidc/payload"
)

const sessionVersion = 2

func (p *Provider) getSession(req *http.Request) (*payload.Session, error) {
	serialized, err := p.getSessionCookie(req)
	switch err {
	case nil:
		// breaks
	case http.ErrNoCookie:
		return nil, nil
	default:
		return nil, err
	}
	// Decode.
	return p.unserializeSession(serialized)
}

func (p *Provider) updateOrCreateSession(rw http.ResponseWriter, req *http.Request, ar *payload.AuthenticationRequest, auth identity.AuthRecord) (*payload.Session, error) {
	session := ar.Session
	if session != nil && session.Version == sessionVersion && session.Sub == auth.Subject() {
		// Existing session with same sub.
		return session, nil
	}

	// Create new session.
	session = &payload.Session{
		Version:  sessionVersion,
		ID:       rndm.GenerateRandomString(32),
		Sub:      auth.Subject(),
		Provider: auth.Manager().Name(),
	}

	serialized, err := p.serializeSession(session)
	if err != nil {
		return session, err
	}
	err = p.setSessionCookie(rw, serialized)

	return session, err
}

func (p *Provider) serializeSession(session *payload.Session) (string, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	err := enc.Encode(session)
	if err != nil {
		return "", err
	}

	ciphertext, err := p.encryptionManager.Encrypt(b.Bytes())
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (p *Provider) unserializeSession(value string) (*payload.Session, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}

	raw, err := p.encryptionManager.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}

	var session payload.Session

	r := bytes.NewReader(raw)
	dec := gob.NewDecoder(r)

	err = dec.Decode(&session)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (p *Provider) getUserIDAndSessionRefFromClaims(claims *jwt.RegisteredClaims, sessionClaims *oidc.SessionClaims, identityClaims jwt.MapClaims) (string, *string) {
	if claims == nil || identityClaims == nil {
		return "", nil
	}

	if len(claims.Audience) != 1 {
		return "", nil
	}
	audience := claims.Audience[0]

	userIDClaim, _ := identityClaims[konnect.IdentifiedUserIDClaim].(string)
	if userIDClaim == "" {
		return userIDClaim, nil
	}
	userClaim, _ := identityClaims[konnect.IdentifiedUserClaim].(string)
	if userClaim == "" {
		userClaim = userIDClaim
	}

	if sessionClaims != nil {
		sessionIDClaim := sessionClaims.SessionID
		if sessionIDClaim != "" {
			return userIDClaim, &sessionIDClaim
		}
	}

	// NOTE(longsleep): Return the userID from claims and generate a session ref
	// for it. Session refs use the userClaim if available and set by the
	// underlaying backend.
	return userIDClaim, identity.GetSessionRef(p.identityManager.Name(), audience, userClaim)
}
