// Copyright 2018-2020 CERN
// Copyright 2025 OpenCloud GmbH
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

package null

import (
	"context"

	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/group"
	"github.com/opencloud-eu/reva/v2/pkg/group/manager/registry"
)

func init() {
	registry.Register("null", New)
}

type manager struct {
}

// New returns a group manager implementation that return NOT FOUND or empty result set for every call
func New(m map[string]interface{}) (group.Manager, error) {
	return &manager{}, nil
}

func (m *manager) GetGroup(ctx context.Context, gid *grouppb.GroupId, skipFetchingMembers bool) (*grouppb.Group, error) {
	return nil, errtypes.NotFound(gid.OpaqueId)
}

func (m *manager) GetGroupByClaim(ctx context.Context, claim, value string, skipFetchingMembers bool) (*grouppb.Group, error) {
	return nil, errtypes.NotFound(value)
}

func (m *manager) FindGroups(ctx context.Context, query string, skipFetchingMembers bool) ([]*grouppb.Group, error) {
	return []*grouppb.Group{}, nil
}

func (m *manager) GetMembers(ctx context.Context, gid *grouppb.GroupId) ([]*userpb.UserId, error) {
	return nil, errtypes.NotFound(gid.OpaqueId)
}

func (m *manager) HasMember(ctx context.Context, gid *grouppb.GroupId, uid *userpb.UserId) (bool, error) {
	return false, errtypes.NotFound(gid.OpaqueId)
}
