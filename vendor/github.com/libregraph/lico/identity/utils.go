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

package identity

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/libregraph/oidc-go"

	konnectoidc "github.com/libregraph/lico/oidc"
	"github.com/libregraph/lico/oidc/payload"
)

// AuthorizeScopes uses the provided manager and user to filter the provided
// scopes and returns a mapping of only the authorized scopes.
func AuthorizeScopes(manager Manager, user User, scopes map[string]bool) (map[string]bool, map[string]bool) {
	if user == nil {
		return nil, nil
	}

	authorizedScopes := make(map[string]bool)
	unauthorizedScopes := make(map[string]bool)
	supportedScopes := make(map[string]bool)
	for _, scope := range manager.ScopesSupported(scopes) {
		supportedScopes[scope] = true
	}

	for scope, authorizedScope := range scopes {
		for {
			if !authorizedScope {
				// Incoming not authorized.
				break
			}

			authorizedScope = isKnownScope(scope)

			if !authorizedScope {
				if _, ok := supportedScopes[scope]; ok {
					authorizedScope = true
				}
			}

			break
		}

		if authorizedScope {
			authorizedScopes[scope] = true
		} else {
			unauthorizedScopes[scope] = false
		}
	}

	return authorizedScopes, unauthorizedScopes
}

// GetUserClaimsForScopes returns a mapping of user claims of the provided user
// filtered by the provided scopes.
func GetUserClaimsForScopes(user User, scopes map[string]bool, requestedClaimsMaps []*payload.ClaimsRequestMap) map[string]jwt.Claims {
	if user == nil {
		return nil
	}

	claims := make(map[string]jwt.Claims)

	if authorizedScope, _ := scopes[oidc.ScopeEmail]; authorizedScope {
		if userWithEmail, ok := user.(UserWithEmail); ok {
			claims[oidc.ScopeEmail] = &konnectoidc.EmailClaims{
				Email:         userWithEmail.Email(),
				EmailVerified: userWithEmail.EmailVerified(),
			}
		}
	}
	if authorizedScope, _ := scopes[oidc.ScopeProfile]; authorizedScope {
		var profileClaims *konnectoidc.ProfileClaims
		if userWithProfile, ok := user.(UserWithProfile); ok {
			profileClaims = &konnectoidc.ProfileClaims{
				Name:       userWithProfile.Name(),
				FamilyName: userWithProfile.FamilyName(),
				GivenName:  userWithProfile.GivenName(),
			}
		}
		if userWithUsername, ok := user.(UserWithUsername); ok {
			if profileClaims == nil {
				profileClaims = &konnectoidc.ProfileClaims{
					PreferredUsername: userWithUsername.Username(),
				}
			} else {
				profileClaims.PreferredUsername = userWithUsername.Username()
			}
		}
		if profileClaims != nil {
			claims[oidc.ScopeProfile] = profileClaims
		}
	}

	// Add additional supported values for email and profile claims.
	unknownRequestedClaimsWithValue := make(map[string]interface{})
	for _, requestedClaimMap := range requestedClaimsMaps {
		for requestedClaim, requestedClaimEntry := range *requestedClaimMap {
			// NOTE(longsleep): We ignore the actuall value of the claim request
			// and always return requested scopes with standard behavior.
			if scope, ok := payload.GetScopeForClaim(requestedClaim); ok {
				if authorizedScope, _ := scopes[scope]; !authorizedScope {
					// Add claim values if known.
					switch scope {
					case oidc.ScopeEmail:
						if userWithEmail, ok := user.(UserWithEmail); ok {
							scopeClaims := konnectoidc.NewEmailClaims(claims[scope])
							if scopeClaims == nil {
								scopeClaims = &konnectoidc.EmailClaims{}
								claims[scope] = scopeClaims
							}
							switch requestedClaim {
							case oidc.EmailClaim:
								scopeClaims.Email = userWithEmail.Email()
								fallthrough // Always include EmailVerified claim.
							case oidc.EmailVerifiedClaim:
								scopeClaims.EmailVerified = userWithEmail.EmailVerified()
							}
						}
					case oidc.ScopeProfile:
						if userWithProfile, ok := user.(UserWithProfile); ok {
							scopeClaims := konnectoidc.NewProfileClaims(claims[scope])
							if scopeClaims == nil {
								scopeClaims = &konnectoidc.ProfileClaims{}
								claims[scope] = scopeClaims
							}
							switch requestedClaim {
							case oidc.NameClaim:
								scopeClaims.Name = userWithProfile.Name()
							case oidc.FamilyNameClaim:
								scopeClaims.Name = userWithProfile.FamilyName()
							case oidc.GivenNameClaim:
								scopeClaims.Name = userWithProfile.GivenName()
							}
						}
					}
				}
			} else {
				// Add claims which are unknown here to a list of unknown claims
				// with value if the requested claim is with value. This returns
				// the requested claim as is with the provided value.
				if requestedClaimEntry != nil && requestedClaimEntry.Value != nil {
					unknownRequestedClaimsWithValue[requestedClaim] = requestedClaimEntry.Value
				}
			}
		}
	}

	// Add extra claims. Those can  either come from the backend user if it
	// has own scoped claims or might be defined as value by the request.
	var claimsWithoutScope jwt.MapClaims
	if userWithScopedClaims, ok := user.(UserWithScopedClaims); ok {
		// Inject additional scope claims.
		claimsWithoutScope = userWithScopedClaims.ScopedClaims(scopes)
	}
	if len(unknownRequestedClaimsWithValue) > 0 {
		if claimsWithoutScope == nil {
			claimsWithoutScope = make(jwt.MapClaims)
		}
		for claim, value := range unknownRequestedClaimsWithValue {
			claimsWithoutScope[claim] = value
		}
	}
	if claimsWithoutScope != nil {
		claims[""] = claimsWithoutScope
	}

	return claims
}

// GetSessionRef builds a per user and audience unique identifier.
func GetSessionRef(label string, audience string, userID string) *string {
	if userID == "" {
		return nil
	}

	// NOTE(longsleep): For now we ignore the audience. Seems not to have any
	// use to keep multiple sessions from Konnect per audience.
	sessionRef := fmt.Sprintf("%s:-:%s", label, userID)
	return &sessionRef
}

func isKnownScope(scope string) bool {
	// Only authorize the scopes we know.
	switch scope {
	case oidc.ScopeOpenID:
	default:
		// Unknown scopes end up here and are not getting authorized.
		return false
	}

	return true
}
