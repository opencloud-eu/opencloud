// Copyright 2018-2022 CERN
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

package events

import (
	"encoding/json"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
)

// TagsAdded is emitted when a Tag has been added
type TagsAdded struct {
	SpaceOwner *user.UserId
	Tags       string
	Ref        *provider.Reference
	Executant  *user.UserId
	Timestamp  *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (TagsAdded) Unmarshal(v []byte) (interface{}, error) {
	e := TagsAdded{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// TagsRemoved is emitted when a Tag has been added
type TagsRemoved struct {
	SpaceOwner *user.UserId
	Tags       string
	Ref        *provider.Reference
	Executant  *user.UserId
	Timestamp  *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (TagsRemoved) Unmarshal(v []byte) (interface{}, error) {
	e := TagsRemoved{}
	err := json.Unmarshal(v, &e)
	return e, err
}
