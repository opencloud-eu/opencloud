// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package scope

import (
	"context"
	"strings"

	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/rs/zerolog"
)

// Verifier is the function signature which every scope verifier should implement.
type Verifier func(context.Context, *authpb.Scope, interface{}, *zerolog.Logger) (bool, error)

var supportedScopes = map[string]Verifier{
	"user":         userScope,
	"publicshare":  publicshareScope,
	"resourceinfo": resourceinfoScope,
	"lightweight":  lightweightAccountScope,
	"ocmshare":     ocmShareScope,
}

// VerifyScope is the function to be called when dismantling tokens to check if
// the token has access to a particular resource.
func VerifyScope(ctx context.Context, scopeMap map[string]*authpb.Scope, resource interface{}) (bool, error) {
	logger := appctx.GetLogger(ctx)
	for k, scope := range scopeMap {
		for s, f := range supportedScopes {
			if strings.HasPrefix(k, s) {
				if valid, err := f(ctx, scope, resource, logger); err == nil && valid {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func hasRoleEditor(scope *authpb.Scope) bool {
	return scope.Role == authpb.Role_ROLE_OWNER || scope.Role == authpb.Role_ROLE_EDITOR || scope.Role == authpb.Role_ROLE_UPLOADER
}
