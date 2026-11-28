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

package capabilities

import (
	"strings"

	"github.com/opencloud-eu/reva/v3/pkg/owncloud/ocs"
)

type chunkProtocol string

var (
	chunkV1  chunkProtocol = "v1"
	chunkNG  chunkProtocol = "ng"
	chunkTUS chunkProtocol = "tus"
)

func (h *Handler) getCapabilitiesForUserAgent(userAgent string) ocs.CapabilitiesData {
	if userAgent != "" {
		for k, v := range h.userAgentChunkingMap {
			// we could also use a regexp for pattern matching
			if strings.Contains(userAgent, k) {
				// Creating a copy of the capabilities struct is less expensive than taking a lock
				c := h.c
				setCapabilitiesForChunkProtocol(chunkProtocol(v), &c)
				return c
			}
		}
	}
	return h.c
}

func setCapabilitiesForChunkProtocol(cp chunkProtocol, c *ocs.CapabilitiesData) {
	switch cp {
	case chunkV1:
		// 2.7+ will use Chunking V1 if "capabilities > files > bigfilechunking" is "true" AND "capabilities > dav > chunking" is not there
		c.Capabilities.Files.BigFileChunking = true
		c.Capabilities.Dav = nil
		c.Capabilities.Files.TusSupport = nil

	case chunkNG:
		// 2.7+ will use Chunking NG if "capabilities > files > bigfilechunking" is "true" AND "capabilities > dav > chunking" = 1.0
		c.Capabilities.Files.BigFileChunking = true
		c.Capabilities.Dav.Chunking = "1.0"
		c.Capabilities.Files.TusSupport = nil

	case chunkTUS:
		// 2.7+ will use TUS if "capabilities > files > bigfilechunking" is "false" AND "capabilities > dav > chunking" = "" AND "capabilities > files > tus_support" has proper entries.
		c.Capabilities.Files.BigFileChunking = false
		c.Capabilities.Dav.Chunking = ""

		// TODO: infer from various TUS handlers from all known storages
		// until now we take the manually configured tus options
		// c.Capabilities.Files.TusSupport = &data.CapabilitiesFilesTusSupport{
		// 	Version:            "1.0.0",
		// 	Resumable:          "1.0.0",
		// 	Extension:          "creation,creation-with-upload",
		// 	MaxChunkSize:       0,
		// 	HTTPMethodOverride: "",
		// }
	}
}
