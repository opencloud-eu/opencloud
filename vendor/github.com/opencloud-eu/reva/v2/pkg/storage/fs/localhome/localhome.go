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

package localhome

import (
	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/registry"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/localfs"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

func init() {
	registry.Register("localhome", New)
}

type config struct {
	Root        string `mapstructure:"root" docs:"/var/tmp/reva/;Path of root directory for user storage."`
	ShareFolder string `mapstructure:"share_folder" docs:"/MyShares;Path for storing share references."`
	UserLayout  string `mapstructure:"user_layout" docs:"{{.Username}};Template for user home directories"`
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	return c, nil
}

// New returns an implementation to of the storage.FS interface that talks to
// a local filesystem with user homes.
func New(m map[string]interface{}, _ events.Stream, _ *zerolog.Logger) (storage.FS, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}

	conf := localfs.Config{
		Root:        c.Root,
		ShareFolder: c.ShareFolder,
		UserLayout:  c.UserLayout,
	}
	return localfs.NewLocalFS(&conf)
}
