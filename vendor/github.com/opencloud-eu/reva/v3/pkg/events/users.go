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
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
)

// UserCreated is emitted when a user was created
type UserCreated struct {
	Executant *user.UserId
	UserID    string
	Timestamp *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (UserCreated) Unmarshal(v []byte) (interface{}, error) {
	e := UserCreated{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// UserDeleted is emitted when a user was deleted
type UserDeleted struct {
	Executant *user.UserId
	UserID    string
	Timestamp *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (UserDeleted) Unmarshal(v []byte) (interface{}, error) {
	e := UserDeleted{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// UserSoftDeleted is emitted when a user was soft-deleted
type UserSoftDeleted struct {
	Executant     *user.UserId
	UserID        string
	Timestamp     *types.Timestamp
	RetentionTime time.Duration
	Reason        string
}

// Unmarshal to fulfill unmarshaller interface
func (UserSoftDeleted) Unmarshal(v []byte) (interface{}, error) {
	e := UserSoftDeleted{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// UserFeature represents a user feature
type UserFeature struct {
	Name     string
	Value    string
	OldValue *string
}

// UserFeatureChanged is emitted when a user feature was changed
type UserFeatureChanged struct {
	Executant *user.UserId
	UserID    string
	Features  []UserFeature
	Timestamp *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (UserFeatureChanged) Unmarshal(v []byte) (interface{}, error) {
	e := UserFeatureChanged{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// PersonalDataExtracted is emitted when a user data extraction is finished
type PersonalDataExtracted struct {
	Executant *user.UserId
	Timestamp *types.Timestamp
	ErrorMsg  string
}

// Unmarshal to fulfill umarshaller interface
func (PersonalDataExtracted) Unmarshal(v []byte) (interface{}, error) {
	e := PersonalDataExtracted{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// BackchannelLogout is emitted when the callback from the identity provider is received
type BackchannelLogout struct {
	Executant *user.UserId
	SessionId string
	Timestamp *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (BackchannelLogout) Unmarshal(v []byte) (interface{}, error) {
	e := BackchannelLogout{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// UserSignedIn is emitted when a user signs in
type UserSignedIn struct {
	Executant *user.UserId
	Timestamp *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (UserSignedIn) Unmarshal(v []byte) (interface{}, error) {
	e := UserSignedIn{}
	err := json.Unmarshal(v, &e)
	return e, err
}
