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

package micro

import (
	"strings"

	"github.com/opencloud-eu/reva/v3/pkg/app/registry/registry"
)

const defaultPriority = "0"

func init() {
	registry.Register("micro", New)
}

type mimeTypeConfig struct {
	MimeType      string `mapstructure:"mime_type"`
	Extension     string `mapstructure:"extension"`
	Name          string `mapstructure:"name"`
	Description   string `mapstructure:"description"`
	Icon          string `mapstructure:"icon"`
	DefaultApp    string `mapstructure:"default_app"`
	AllowCreation bool   `mapstructure:"allow_creation"`
}

// use the UTF-8 record separator
func splitMimeTypes(s string) []string {
	return strings.Split(s, "␞")
}

func joinMimeTypes(mimetypes []string) string {
	return strings.Join(mimetypes, "␞")
}
