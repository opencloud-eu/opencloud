package oidc

import (
	"fmt"
	"strings"
)

const (
	Iss                    = "iss"
	Sub                    = "sub"
	Email                  = "email"
	Name                   = "name"
	PreferredUsername      = "preferred_username"
	UIDNumber              = "uidnumber"
	GIDNumber              = "gidnumber"
	Groups                 = "groups"
	OpenCloudUUID          = "openclouduuid"
	OpenCloudRoutingPolicy = "opencloud.routing.policy"
)

// SplitWithEscaping splits s into segments using separator which can be escaped using the escape string
// See https://codereview.stackexchange.com/a/280193
func SplitWithEscaping(s string, separator string, escapeString string) []string {
	a := strings.Split(s, separator)

	for i := len(a) - 2; i >= 0; i-- {
		if strings.HasSuffix(a[i], escapeString) {
			a[i] = a[i][:len(a[i])-len(escapeString)] + separator + a[i+1]
			a = append(a[:i+1], a[i+2:]...)
		}
	}
	return a
}

// WalkSegments uses the given array of segments to walk the claims and return whatever interface was found
func WalkSegments(segments []string, claims map[string]any) (any, error) {
	i := 0
	for ; i < len(segments)-1; i++ {
		switch castedClaims := claims[segments[i]].(type) {
		case map[string]any:
			claims = castedClaims
		case map[any]any:
			claims = make(map[string]any, len(castedClaims))
			for k, v := range castedClaims {
				if s, ok := k.(string); ok {
					claims[s] = v
				} else {
					return nil, fmt.Errorf("could not walk claims path, key '%v' is not a string", k)
				}
			}
		default:
			return nil, fmt.Errorf("unsupported type '%v'", castedClaims)
		}
	}
	return claims[segments[i]], nil
}

// ReadStringClaim returns the string obtained by following the . seperated path in the claims
func ReadStringClaim(path string, claims map[string]any) (string, error) {
	// check the simple case first
	value, _ := claims[path].(string)
	if value != "" {
		return value, nil
	}

	claim, err := WalkSegments(SplitWithEscaping(path, ".", "\\"), claims)
	if err != nil {
		return "", err
	}

	if value, _ = claim.(string); value != "" {
		return value, nil
	}

	return value, fmt.Errorf("claim path '%s' not set or empty", path)
}
