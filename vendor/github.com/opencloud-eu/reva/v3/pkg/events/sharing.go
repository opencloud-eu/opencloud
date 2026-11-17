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

package events

import (
	"encoding/json"
	"time"

	group "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
)

// ShareCreated is emitted when a share is created
type ShareCreated struct {
	ShareID   *collaboration.ShareId
	Executant *user.UserId
	Sharer    *user.UserId
	// split the protobuf Grantee oneof so we can use stdlib encoding/json
	GranteeUserID  *user.UserId
	GranteeGroupID *group.GroupId
	Sharee         *provider.Grantee
	ItemID         *provider.ResourceId
	ResourceName   string
	Permissions    *collaboration.SharePermissions
	CTime          *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (ShareCreated) Unmarshal(v []byte) (interface{}, error) {
	e := ShareCreated{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// ShareRemoved is emitted when a share is removed
type ShareRemoved struct {
	Executant *user.UserId
	// split protobuf Spec
	ShareID  *collaboration.ShareId
	ShareKey *collaboration.ShareKey
	// split the protobuf Grantee oneof so we can use stdlib encoding/json
	GranteeUserID  *user.UserId
	GranteeGroupID *group.GroupId

	ItemID       *provider.ResourceId
	ResourceName string
	Timestamp    time.Time
}

// Unmarshal to fulfill umarshaller interface
func (ShareRemoved) Unmarshal(v []byte) (interface{}, error) {
	e := ShareRemoved{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// ShareUpdated is emitted when a share is updated
type ShareUpdated struct {
	Executant      *user.UserId
	ShareID        *collaboration.ShareId
	ItemID         *provider.ResourceId
	ResourceName   string
	Permissions    *collaboration.SharePermissions
	GranteeUserID  *user.UserId
	GranteeGroupID *group.GroupId
	Sharer         *user.UserId
	MTime          *types.Timestamp

	Updated string // Deprecated
	// indicates what was updated
	UpdateMask []string
}

// Unmarshal to fulfill umarshaller interface
func (ShareUpdated) Unmarshal(v []byte) (interface{}, error) {
	e := ShareUpdated{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// ShareExpired is emitted when a share expires
type ShareExpired struct {
	ShareID    *collaboration.ShareId
	ShareOwner *user.UserId
	ItemID     *provider.ResourceId
	Path       string
	ExpiredAt  time.Time
	// split the protobuf Grantee oneof so we can use stdlib encoding/json
	GranteeUserID  *user.UserId
	GranteeGroupID *group.GroupId
}

// Unmarshal to fulfill umarshaller interface
func (ShareExpired) Unmarshal(v []byte) (interface{}, error) {
	e := ShareExpired{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// ReceivedShareUpdated is emitted when a received share is accepted or declined
type ReceivedShareUpdated struct {
	Executant      *user.UserId
	ShareID        *collaboration.ShareId
	ItemID         *provider.ResourceId
	Path           string
	Permissions    *collaboration.SharePermissions
	GranteeUserID  *user.UserId
	GranteeGroupID *group.GroupId
	Sharer         *user.UserId
	MTime          *types.Timestamp

	State string
}

// Unmarshal to fulfill umarshaller interface
func (ReceivedShareUpdated) Unmarshal(v []byte) (interface{}, error) {
	e := ReceivedShareUpdated{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// LinkCreated is emitted when a public link is created
type LinkCreated struct {
	Executant         *user.UserId
	ShareID           *link.PublicShareId
	Sharer            *user.UserId
	ItemID            *provider.ResourceId
	ResourceName      string
	Permissions       *link.PublicSharePermissions
	DisplayName       string
	Expiration        *types.Timestamp
	PasswordProtected bool
	CTime             *types.Timestamp
	Token             string
}

// Unmarshal to fulfill umarshaller interface
func (LinkCreated) Unmarshal(v []byte) (interface{}, error) {
	e := LinkCreated{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// LinkUpdated is emitted when a public link is updated
type LinkUpdated struct {
	Executant         *user.UserId
	ShareID           *link.PublicShareId
	Sharer            *user.UserId
	ItemID            *provider.ResourceId
	ResourceName      string
	Permissions       *link.PublicSharePermissions
	DisplayName       string
	Expiration        *types.Timestamp
	PasswordProtected bool
	MTime             *types.Timestamp
	Token             string

	FieldUpdated string
}

// Unmarshal to fulfill umarshaller interface
func (LinkUpdated) Unmarshal(v []byte) (interface{}, error) {
	e := LinkUpdated{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// LinkAccessed is emitted when a public link is accessed successfully (by token)
type LinkAccessed struct {
	Executant         *user.UserId
	ShareID           *link.PublicShareId
	Sharer            *user.UserId
	ItemID            *provider.ResourceId
	Path              string
	Permissions       *link.PublicSharePermissions
	DisplayName       string
	Expiration        *types.Timestamp
	PasswordProtected bool
	CTime             *types.Timestamp
	Token             string
}

// Unmarshal to fulfill umarshaller interface
func (LinkAccessed) Unmarshal(v []byte) (interface{}, error) {
	e := LinkAccessed{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// LinkAccessFailed is emitted when an access to a public link has resulted in an error (by token)
type LinkAccessFailed struct {
	Executant *user.UserId
	ShareID   *link.PublicShareId
	Token     string
	Status    rpc.Code
	Message   string
	Timestamp *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (LinkAccessFailed) Unmarshal(v []byte) (interface{}, error) {
	e := LinkAccessFailed{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// LinkRemoved is emitted when a share is removed
type LinkRemoved struct {
	Executant *user.UserId
	// split protobuf Ref
	ShareID      *link.PublicShareId
	ShareToken   string
	Timestamp    *types.Timestamp
	ItemID       *provider.ResourceId
	ResourceName string
}

// Unmarshal to fulfill umarshaller interface
func (LinkRemoved) Unmarshal(v []byte) (interface{}, error) {
	e := LinkRemoved{}
	err := json.Unmarshal(v, &e)
	return e, err
}
