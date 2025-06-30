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

package decomposed

import (
	"path"

	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/decomposed/blobstore"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/registry"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/options"
	"github.com/rs/zerolog"
)

func init() {
	registry.Register("decomposed", New)
}

// New returns an implementation to of the storage.FS interface that talk to
// a local filesystem.
func New(m map[string]interface{}, stream events.Stream, log *zerolog.Logger) (storage.FS, error) {
	o, err := options.New(m)
	if err != nil {
		return nil, err
	}

	if log == nil {
		log = &zerolog.Logger{}
	}
	decomposedLog := log.With().Str("driver", "decomposed").Logger()
	log = &decomposedLog

	bs, err := blobstore.New(path.Join(o.Root))
	if err != nil {
		return nil, err
	}

	return decomposedfs.NewDefault(m, bs, stream, log)
}
