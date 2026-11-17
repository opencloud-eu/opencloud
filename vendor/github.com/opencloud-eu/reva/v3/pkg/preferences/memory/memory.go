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

package memory

import (
	"context"
	"sync"

	ctxpkg "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/preferences"
	"github.com/opencloud-eu/reva/v3/pkg/preferences/registry"
)

func init() {
	registry.Register("memory", New)
}

type mgr struct {
	sync.RWMutex
	keys map[string]map[string]string
}

// New returns an instance of the in-memory preferences manager.
func New(m map[string]interface{}) (preferences.Manager, error) {
	return &mgr{keys: make(map[string]map[string]string)}, nil
}

func (m *mgr) SetKey(ctx context.Context, key, namespace, value string) error {
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		return errtypes.UserRequired("preferences: error getting user from ctx")
	}
	m.Lock()
	defer m.Unlock()

	userKey := u.Id.OpaqueId

	if len(m.keys[userKey]) == 0 {
		m.keys[userKey] = map[string]string{key: value}
	} else {
		m.keys[userKey][key] = value
	}
	return nil
}

func (m *mgr) GetKey(ctx context.Context, key, namespace string) (string, error) {
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		return "", errtypes.UserRequired("preferences: error getting user from ctx")
	}
	m.RLock()
	defer m.RUnlock()

	userKey := u.Id.OpaqueId

	if len(m.keys[userKey]) != 0 {
		if value, ok := m.keys[userKey][key]; ok {
			return value, nil
		}
	}
	return "", errtypes.NotFound("preferences: key not found")
}
