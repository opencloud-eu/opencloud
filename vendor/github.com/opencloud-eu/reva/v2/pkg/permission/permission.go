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

package permission

import (
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
)

const (
	// ListAllSpaces is the hardcoded name for the list all spaces permission
	ListAllSpaces string = "Drives.List"
	// CreateSpace is the hardcoded name for the create space permission
	CreateSpace string = "Drives.Create"
	// WritePublicLink is the hardcoded name for the PublicLink.Write permission
	WritePublicLink string = "PublicLink.Write"
	// WriteShare is the hardcoded name for the Shares.Write permission
	WriteShare string = "Shares.Write"
	// GuestMailWrite is the hardcoded name for the Shares.GuestMailWrite permission
	GuestMailWrite string = "Shares.GuestMailWrite"
	// ListFavorites is the hardcoded name for the Favorites.List permission
	ListFavorites string = "Favorites.List"
	// WriteFavorites is the hardcoded name for the Favorites.Write permission
	WriteFavorites string = "Favorites.Write"
	// DeleteReadOnlyPassword is the hardcoded name for the ReadOnlyPublicLinkPassword.Delete permission
	DeleteReadOnlyPassword string = "ReadOnlyPublicLinkPassword.Delete"
)

// Manager defines the interface for the permission service driver
type Manager interface {
	CheckPermission(permission string, subject string, ref *provider.Reference) bool
}
