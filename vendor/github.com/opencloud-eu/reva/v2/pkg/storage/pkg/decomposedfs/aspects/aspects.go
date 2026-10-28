// Copyright 2018-2024 CERN
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

package aspects

import (
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/permissions"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/revisioncleanup"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/trashbin"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/usermapper"
)

// Aspects holds dependencies for handling aspects of the decomposedfs
type Aspects struct {
	Lookup            node.PathLookup
	Tree              node.Tree
	Trashbin          trashbin.Trashbin
	Permissions       permissions.Permissions
	EventStream       events.Stream
	DisableVersioning bool
	UserMapper        usermapper.Mapper
	// RevisionsCleanup deletes old revisions after new ones have been created, right away or in
	// background workers; disabled without a strategy, and nil disables it as well. It is
	// passed in here rather than created by Decomposedfs because posix needs the same instance
	// (and the same workers) in its own tree, for revisions created from changes on disk.
	RevisionsCleanup *revisioncleanup.Service
}
