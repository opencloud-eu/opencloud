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

package ctx

import (
	"context"

	auth "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
)

// ContextGetScopes returns the scopes if set in the given context.
func ContextGetScopes(ctx context.Context) (map[string]*auth.Scope, bool) {
	s, ok := ctx.Value(scopeKey).(map[string]*auth.Scope)
	return s, ok
}

// ContextSetScopes stores the scopes in the context.
func ContextSetScopes(ctx context.Context, s map[string]*auth.Scope) context.Context {
	return context.WithValue(ctx, scopeKey, s)
}
