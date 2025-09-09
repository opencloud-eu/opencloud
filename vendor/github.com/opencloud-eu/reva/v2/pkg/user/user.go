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

package user

import (
	"context"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/plugin"
)

// Manager is the interface to implement to manipulate users.
type Manager interface {
	plugin.Plugin
	// GetUser returns the user metadata identified by a uid.
	// The groups of the user are omitted if specified, as these might not be required for certain operations
	// and might involve computational overhead.
	GetUser(ctx context.Context, uid *userpb.UserId, skipFetchingGroups bool) (*userpb.User, error)
	// GetUserByClaim returns the user identified by a specific value for a given claim.
	GetUserByClaim(ctx context.Context, claim, value, tenantID string, skipFetchingGroups bool) (*userpb.User, error)
	// GetUserGroups returns the groups a user identified by a uid belongs to.
	GetUserGroups(ctx context.Context, uid *userpb.UserId) ([]string, error)
	// FindUsers returns all the user objects which match a query parameter.
	FindUsers(ctx context.Context, query, tenantID string, skipFetchingGroups bool) ([]*userpb.User, error)
}
