// Copyright 2021 CERN
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

package demo

import (
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/permission"
	"github.com/opencloud-eu/reva/v2/pkg/permission/manager/registry"
)

func init() {
	registry.Register("demo", New)
}

// New returns a new demo permission manager
func New(c map[string]interface{}) (permission.Manager, error) {
	return manager{}, nil
}

type manager struct {
}

func (m manager) CheckPermission(perm string, subject string, ref *provider.Reference) bool {
	switch perm {
	case permission.CreateSpace:
		// TODO Users can only create their own personal space
		// TODO guest accounts cannot create spaces
		return true
	case permission.WritePublicLink:
		return true
	case permission.ListAllSpaces:
		// TODO introduce an admin role to allow listing all spaces
		return false
	case permission.WriteShare:
		// TODO guest accounts cannot share
		return true
	case permission.GuestMailWrite:
		return true
	case permission.ListFavorites:
		// TODO guest accounts cannot list favorites
		return true
	case permission.WriteFavorites:
		// TODO guest accounts cannot write favorites
		return true
	default:
		// We can currently return false all the time.
		// Once we beginn testing roles we need to somehow check the roles of the users here
		return false
	}
}
