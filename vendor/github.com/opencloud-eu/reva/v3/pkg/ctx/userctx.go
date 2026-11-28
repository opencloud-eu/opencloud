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

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
)

type key int

const (
	userKey key = iota
	tokenKey
	idKey
	lockIDKey
	scopeKey
	initiatorKey
)

// ContextGetUser returns the user if set in the given context.
func ContextGetUser(ctx context.Context) (*userpb.User, bool) {
	u, ok := ctx.Value(userKey).(*userpb.User)
	return u, ok
}

// ContextMustGetUser panics if user is not in context.
func ContextMustGetUser(ctx context.Context) *userpb.User {
	u, ok := ContextGetUser(ctx)
	if !ok {
		panic("user not found in context")
	}
	return u
}

// ContextSetUser stores the user in the context.
func ContextSetUser(ctx context.Context, u *userpb.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// ContextGetUserID returns the user if set in the given context.
func ContextGetUserID(ctx context.Context) (*userpb.UserId, bool) {
	u, ok := ctx.Value(idKey).(*userpb.UserId)
	return u, ok
}

// ContextSetUserID stores the userid in the context.
func ContextSetUserID(ctx context.Context, id *userpb.UserId) context.Context {
	return context.WithValue(ctx, idKey, id)
}
