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

package appauth

import (
	"context"

	apppb "github.com/cs3org/go-cs3apis/cs3/auth/applications/v1beta1"
	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
)

// Manager is the interface that manages application authentication mechanisms.
type Manager interface {
	// GenerateAppPassword creates a password with specified scope to be used by
	// third-party applications.
	GenerateAppPassword(ctx context.Context, scope map[string]*authpb.Scope, label string, expiration *typespb.Timestamp) (*apppb.AppPassword, error)
	// ListAppPasswords lists the application passwords created by a user.
	ListAppPasswords(ctx context.Context) ([]*apppb.AppPassword, error)
	// InvalidateAppPassword invalidates a generated password.
	InvalidateAppPassword(ctx context.Context, secret string) error
	// GetAppPassword retrieves the password information by the combination of username and password.
	GetAppPassword(ctx context.Context, user *userpb.UserId, secret string) (*apppb.AppPassword, error)
}
