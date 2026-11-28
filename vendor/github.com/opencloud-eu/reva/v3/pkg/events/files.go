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

// ContainerCreated is emitted when a directory has been created
type ContainerCreated struct {
	SpaceOwner        *user.UserId
	Executant         *user.UserId
	Ref               *provider.Reference
	ParentID          *provider.ResourceId
	Owner             *user.UserId
	Timestamp         *types.Timestamp
	ImpersonatingUser *user.User
}

// Unmarshal to fulfill umarshaller interface
func (ContainerCreated) Unmarshal(v []byte) (interface{}, error) {
	e := ContainerCreated{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// FileUploaded is emitted when a file is uploaded
type FileUploaded struct {
	SpaceOwner        *user.UserId
	Executant         *user.UserId
	Ref               *provider.Reference
	Owner             *user.UserId
	Timestamp         *types.Timestamp
	ImpersonatingUser *user.User
}

// Unmarshal to fulfill umarshaller interface
func (FileUploaded) Unmarshal(v []byte) (interface{}, error) {
	e := FileUploaded{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// FileTouched is emitted when a file is uploaded
type FileTouched struct {
	SpaceOwner        *user.UserId
	Executant         *user.UserId
	Ref               *provider.Reference
	ParentID          *provider.ResourceId
	Timestamp         *types.Timestamp
	ImpersonatingUser *user.User
}

// Unmarshal to fulfill umarshaller interface
func (FileTouched) Unmarshal(v []byte) (interface{}, error) {
	e := FileTouched{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// FileDownloaded is emitted when a file is downloaded
type FileDownloaded struct {
	Executant         *user.UserId
	Ref               *provider.Reference
	Owner             *user.UserId
	Timestamp         *types.Timestamp
	ImpersonatingUser *user.User
}

// Unmarshal to fulfill umarshaller interface
func (FileDownloaded) Unmarshal(v []byte) (interface{}, error) {
	e := FileDownloaded{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// FileLocked is emitted when a file is locked
type FileLocked struct {
	Executant         *user.UserId
	Ref               *provider.Reference
	Owner             *user.UserId
	Timestamp         *types.Timestamp
	ImpersonatingUser *user.User
}

// Unmarshal to fulfill umarshaller interface
func (FileLocked) Unmarshal(v []byte) (interface{}, error) {
	e := FileLocked{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// FileUnlocked is emitted when a file is unlocked
type FileUnlocked struct {
	Executant         *user.UserId
	Ref               *provider.Reference
	Owner             *user.UserId
	Timestamp         *types.Timestamp
	ImpersonatingUser *user.User
}

// Unmarshal to fulfill umarshaller interface
func (FileUnlocked) Unmarshal(v []byte) (interface{}, error) {
	e := FileUnlocked{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// ItemTrashed is emitted when a file or folder is trashed
type ItemTrashed struct {
	SpaceOwner        *user.UserId
	Executant         *user.UserId
	ID                *provider.ResourceId
	Ref               *provider.Reference
	Owner             *user.UserId
	Timestamp         *types.Timestamp
	ImpersonatingUser *user.User
}

// Unmarshal to fulfill umarshaller interface
func (ItemTrashed) Unmarshal(v []byte) (interface{}, error) {
	e := ItemTrashed{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// ItemMoved is emitted when a file or folder is moved
type ItemMoved struct {
	SpaceOwner        *user.UserId
	Executant         *user.UserId
	Ref               *provider.Reference
	Owner             *user.UserId
	OldReference      *provider.Reference
	Timestamp         *types.Timestamp
	ImpersonatingUser *user.User
}

// Unmarshal to fulfill umarshaller interface
func (ItemMoved) Unmarshal(v []byte) (interface{}, error) {
	e := ItemMoved{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// TrashbinPurged is emitted when the whole trashbin is purged
type TrashbinPurged struct {
	Executant         *user.UserId
	Ref               *provider.Reference
	Owner             *user.UserId
	Timestamp         *types.Timestamp
	ImpersonatingUser *user.User
}

// Unmarshal to fulfill umarshaller interface
func (TrashbinPurged) Unmarshal(v []byte) (interface{}, error) {
	e := TrashbinPurged{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// ItemPurged is emitted when a file or folder is removed from trashbin
type ItemPurged struct {
	Executant         *user.UserId
	ID                *provider.ResourceId
	Ref               *provider.Reference
	Owner             *user.UserId
	Timestamp         *types.Timestamp
	ImpersonatingUser *user.User
}

// Unmarshal to fulfill umarshaller interface
func (ItemPurged) Unmarshal(v []byte) (interface{}, error) {
	e := ItemPurged{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// ItemRestored is emitted when a file or folder is restored from trashbin
type ItemRestored struct {
	SpaceOwner        *user.UserId
	Executant         *user.UserId
	ID                *provider.ResourceId
	Ref               *provider.Reference
	Owner             *user.UserId
	OldReference      *provider.Reference
	Key               string
	Timestamp         *types.Timestamp
	ImpersonatingUser *user.User
}

// Unmarshal to fulfill umarshaller interface
func (ItemRestored) Unmarshal(v []byte) (interface{}, error) {
	e := ItemRestored{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// FileVersionRestored is emitted when a file version is restored
type FileVersionRestored struct {
	SpaceOwner        *user.UserId
	Executant         *user.UserId
	Ref               *provider.Reference
	Owner             *user.UserId
	Key               string
	Timestamp         *types.Timestamp
	ImpersonatingUser *user.User
}

// Unmarshal to fulfill umarshaller interface
func (FileVersionRestored) Unmarshal(v []byte) (interface{}, error) {
	e := FileVersionRestored{}
	err := json.Unmarshal(v, &e)
	return e, err
}
