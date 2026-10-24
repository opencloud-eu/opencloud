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

	group "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
)

// SpaceCreated is emitted when a space is created
type SpaceCreated struct {
	Executant *user.UserId
	ID        *provider.StorageSpaceId
	Owner     *user.UserId
	Root      *provider.ResourceId
	Name      string
	Type      string
	Quota     *provider.Quota
	MTime     *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (SpaceCreated) Unmarshal(v []byte) (interface{}, error) {
	e := SpaceCreated{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// SpaceRenamed is emitted when a space is renamed
type SpaceRenamed struct {
	Executant *user.UserId
	ID        *provider.StorageSpaceId
	Owner     *user.UserId
	Name      string
	Timestamp *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (SpaceRenamed) Unmarshal(v []byte) (interface{}, error) {
	e := SpaceRenamed{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// SpaceDisabled is emitted when a space is disabled
type SpaceDisabled struct {
	Executant *user.UserId
	ID        *provider.StorageSpaceId
	Members   map[string]provider.ResourcePermissions
	Timestamp time.Time
}

// Unmarshal to fulfill umarshaller interface
func (SpaceDisabled) Unmarshal(v []byte) (interface{}, error) {
	e := SpaceDisabled{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// SpaceEnabled is emitted when a space is (re-)enabled
type SpaceEnabled struct {
	Executant *user.UserId
	ID        *provider.StorageSpaceId
	Owner     *user.UserId
	Members   map[string]provider.ResourcePermissions
	Timestamp *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (SpaceEnabled) Unmarshal(v []byte) (interface{}, error) {
	e := SpaceEnabled{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// SpaceDeleted is emitted when a space is deleted
type SpaceDeleted struct {
	Executant    *user.UserId
	ID           *provider.StorageSpaceId
	SpaceName    string
	FinalMembers map[string]provider.ResourcePermissions
	Timestamp    time.Time
}

// Unmarshal to fulfill umarshaller interface
func (SpaceDeleted) Unmarshal(v []byte) (interface{}, error) {
	e := SpaceDeleted{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// SpaceShared is emitted when a space is shared
type SpaceShared struct {
	ShareID        *collaboration.ShareId
	Executant      *user.UserId
	GranteeUserID  *user.UserId
	GranteeGroupID *group.GroupId
	Creator        *user.UserId
	ID             *provider.StorageSpaceId
	Timestamp      time.Time
}

// Unmarshal to fulfill umarshaller interface
func (SpaceShared) Unmarshal(v []byte) (interface{}, error) {
	e := SpaceShared{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// SpaceShareUpdated is emitted when a space share is updated
type SpaceShareUpdated struct {
	ShareID        *collaboration.ShareId
	Executant      *user.UserId
	GranteeUserID  *user.UserId
	GranteeGroupID *group.GroupId
	ID             *provider.StorageSpaceId
	Timestamp      time.Time
}

// Unmarshal to fulfill umarshaller interface
func (SpaceShareUpdated) Unmarshal(v []byte) (interface{}, error) {
	ev := SpaceShareUpdated{}
	err := json.Unmarshal(v, &ev)
	return ev, err
}

// SpaceUnshared is emitted when a space is unshared
type SpaceUnshared struct {
	ShareID        *collaboration.ShareId
	Executant      *user.UserId
	GranteeUserID  *user.UserId
	GranteeGroupID *group.GroupId
	ID             *provider.StorageSpaceId
	Timestamp      time.Time
}

// Unmarshal to fulfill umarshaller interface
func (SpaceUnshared) Unmarshal(v []byte) (interface{}, error) {
	e := SpaceUnshared{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// SpaceUpdated is emitted when a space is updated
type SpaceUpdated struct {
	Executant *user.UserId
	ID        *provider.StorageSpaceId
	Space     *provider.StorageSpace
	Timestamp *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (SpaceUpdated) Unmarshal(v []byte) (interface{}, error) {
	e := SpaceUpdated{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// SpaceMembershipExpired is emitted when a space membership expires
type SpaceMembershipExpired struct {
	ShareID    *collaboration.ShareId
	SpaceOwner *user.UserId
	SpaceID    *provider.StorageSpaceId
	SpaceName  string
	ExpiredAt  time.Time
	// split the protobuf Grantee oneof so we can use stdlib encoding/json
	GranteeUserID  *user.UserId
	GranteeGroupID *group.GroupId
	Timestamp      *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (SpaceMembershipExpired) Unmarshal(v []byte) (interface{}, error) {
	e := SpaceMembershipExpired{}
	err := json.Unmarshal(v, &e)
	return e, err
}
