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
	"fmt"

	"github.com/opencloud-eu/reva/v3/pkg/publicshare/manager/json/persistence"
)

type memory struct {
	db map[string]interface{}
}

// New returns a new Cache instance
func New() persistence.Persistence {
	return &memory{
		db: map[string]interface{}{},
	}
}

func (p *memory) Init(_ context.Context) error {
	return nil
}

func (p *memory) Read(_ context.Context) (persistence.PublicShares, error) {
	if p.db == nil {
		return nil, fmt.Errorf("not initialized")
	}
	return p.db, nil
}
func (p *memory) Write(_ context.Context, db persistence.PublicShares) error {
	if p.db == nil {
		return fmt.Errorf("not initialized")
	}
	p.db = db
	return nil
}
