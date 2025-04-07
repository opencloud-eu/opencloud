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
	"net/http"

	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocs/config"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocs/response"
	"github.com/opencloud-eu/reva/v2/pkg/owncloud/ocs"
)

// Handler renders the capability endpoint
type Handler struct {
	c                     ocs.CapabilitiesData
	defaultUploadProtocol string
	userAgentChunkingMap  map[string]string
}

// Init initializes this and any contained handlers
func (h *Handler) Init(c *config.Config) {
	h.c = c.Capabilities
	h.defaultUploadProtocol = c.DefaultUploadProtocol
	h.userAgentChunkingMap = c.UserAgentChunkingMap

	// capabilities
	if h.c.Capabilities == nil {
		h.c.Capabilities = &ocs.Capabilities{}
	}

	// core

	if h.c.Capabilities.Core == nil {
		h.c.Capabilities.Core = &ocs.CapabilitiesCore{}
	}
	if h.c.Capabilities.Core.PollInterval == 0 {
		h.c.Capabilities.Core.PollInterval = 60
	}
	if h.c.Capabilities.Core.WebdavRoot == "" {
		h.c.Capabilities.Core.WebdavRoot = "remote.php/webdav"
	}
	// h.c.Capabilities.Core.SupportURLSigning is boolean

	if h.c.Capabilities.Core.Status == nil {
		h.c.Capabilities.Core.Status = &ocs.Status{}
	}
	// h.c.Capabilities.Core.Status.Installed is boolean
	// h.c.Capabilities.Core.Status.Maintenance is boolean
	// h.c.Capabilities.Core.Status.NeedsDBUpgrade is boolean
	if h.c.Capabilities.Core.Status.Version == "" {
		h.c.Capabilities.Core.Status.Version = "10.0.11.5" // TODO make build determined
	}
	if h.c.Capabilities.Core.Status.VersionString == "" {
		h.c.Capabilities.Core.Status.VersionString = "10.0.11" // TODO make build determined
	}
	if h.c.Capabilities.Core.Status.Edition == "" {
		h.c.Capabilities.Core.Status.Edition = "" // TODO make build determined
	}
	if h.c.Capabilities.Core.Status.ProductName == "" {
		h.c.Capabilities.Core.Status.ProductName = "reva" // TODO make build determined
	}
	if h.c.Capabilities.Core.Status.Product == "" {
		h.c.Capabilities.Core.Status.Product = "reva" // TODO make build determined
	}
	if h.c.Capabilities.Core.Status.Hostname == "" {
		h.c.Capabilities.Core.Status.Hostname = "" // TODO get from context?
	}

	// checksums

	if h.c.Capabilities.Checksums == nil {
		h.c.Capabilities.Checksums = &ocs.CapabilitiesChecksums{}
	}
	if h.c.Capabilities.Checksums.SupportedTypes == nil {
		h.c.Capabilities.Checksums.SupportedTypes = []string{"SHA256"}
	}
	if h.c.Capabilities.Checksums.PreferredUploadType == "" {
		h.c.Capabilities.Checksums.PreferredUploadType = "SHA1"
	}

	// files

	if h.c.Capabilities.Files == nil {
		h.c.Capabilities.Files = &ocs.CapabilitiesFiles{}
	}

	if h.c.Capabilities.Files.BlacklistedFiles == nil {
		h.c.Capabilities.Files.BlacklistedFiles = []string{}
	}
	// h.c.Capabilities.Files.Undelete is boolean
	// h.c.Capabilities.Files.Versioning is boolean
	// h.c.Capabilities.Files.Favorites is boolean

	if h.c.Capabilities.Files.Archivers == nil {
		h.c.Capabilities.Files.Archivers = []*ocs.CapabilitiesArchiver{}
	}

	if h.c.Capabilities.Files.AppProviders == nil {
		h.c.Capabilities.Files.AppProviders = []*ocs.CapabilitiesAppProvider{}
	}

	// dav

	if h.c.Capabilities.Dav == nil {
		h.c.Capabilities.Dav = &ocs.CapabilitiesDav{}
	}
	if h.c.Capabilities.Dav.Trashbin == "" {
		h.c.Capabilities.Dav.Trashbin = "1.0"
	}
	if h.c.Capabilities.Dav.Reports == nil {
		h.c.Capabilities.Dav.Reports = []string{}
	}

	// sharing

	if h.c.Capabilities.FilesSharing == nil {
		h.c.Capabilities.FilesSharing = &ocs.CapabilitiesFilesSharing{}
	}

	// h.c.Capabilities.FilesSharing.APIEnabled is boolean

	if h.c.Capabilities.FilesSharing.Public == nil {
		h.c.Capabilities.FilesSharing.Public = &ocs.CapabilitiesFilesSharingPublic{}
	}

	// h.c.Capabilities.FilesSharing.IsPublic.Enabled is boolean
	h.c.Capabilities.FilesSharing.Public.Enabled = true

	if h.c.Capabilities.FilesSharing.Public.Password == nil {
		h.c.Capabilities.FilesSharing.Public.Password = &ocs.CapabilitiesFilesSharingPublicPassword{}
	}

	if h.c.Capabilities.FilesSharing.Public.Password.EnforcedFor == nil {
		h.c.Capabilities.FilesSharing.Public.Password.EnforcedFor = &ocs.CapabilitiesFilesSharingPublicPasswordEnforcedFor{}
	}

	// h.c.Capabilities.FilesSharing.IsPublic.Password.EnforcedFor.ReadOnly is boolean
	// h.c.Capabilities.FilesSharing.IsPublic.Password.EnforcedFor.ReadWrite is boolean
	// h.c.Capabilities.FilesSharing.IsPublic.Password.EnforcedFor.ReadWriteDelete is boolean
	// h.c.Capabilities.FilesSharing.IsPublic.Password.EnforcedFor.UploadOnly is boolean

	// h.c.Capabilities.FilesSharing.IsPublic.Password.Enforced is boolean

	if h.c.Capabilities.FilesSharing.Public.ExpireDate == nil {
		h.c.Capabilities.FilesSharing.Public.ExpireDate = &ocs.CapabilitiesFilesSharingPublicExpireDate{}
	}
	// h.c.Capabilities.FilesSharing.IsPublic.ExpireDate.Enabled is boolean

	// h.c.Capabilities.FilesSharing.IsPublic.SendMail is boolean
	// h.c.Capabilities.FilesSharing.IsPublic.SocialShare is boolean
	// h.c.Capabilities.FilesSharing.IsPublic.Upload is boolean
	// h.c.Capabilities.FilesSharing.IsPublic.Multiple is boolean
	// h.c.Capabilities.FilesSharing.IsPublic.SupportsUploadOnly is boolean

	if h.c.Capabilities.FilesSharing.User == nil {
		h.c.Capabilities.FilesSharing.User = &ocs.CapabilitiesFilesSharingUser{}
	}

	// h.c.Capabilities.FilesSharing.User.SendMail is boolean

	// h.c.Capabilities.FilesSharing.Resharing is boolean
	// h.c.Capabilities.FilesSharing.GroupSharing is boolean
	// h.c.Capabilities.FilesSharing.SharingRoles is boolean
	// h.c.Capabilities.FilesSharing.AutoAcceptShare is boolean
	// h.c.Capabilities.FilesSharing.ShareWithGroupMembersOnly is boolean
	// h.c.Capabilities.FilesSharing.ShareWithMembershipGroupsOnly is boolean

	if h.c.Capabilities.FilesSharing.UserEnumeration == nil {
		h.c.Capabilities.FilesSharing.UserEnumeration = &ocs.CapabilitiesFilesSharingUserEnumeration{}
	}

	// h.c.Capabilities.FilesSharing.UserEnumeration.Enabled is boolean
	// h.c.Capabilities.FilesSharing.UserEnumeration.GroupMembersOnly is boolean

	if h.c.Capabilities.FilesSharing.DefaultPermissions == 0 {
		h.c.Capabilities.FilesSharing.DefaultPermissions = 31
	}
	if h.c.Capabilities.FilesSharing.Federation == nil {
		h.c.Capabilities.FilesSharing.Federation = &ocs.CapabilitiesFilesSharingFederation{}
	}

	// h.c.Capabilities.FilesSharing.Federation.Outgoing is boolean
	// h.c.Capabilities.FilesSharing.Federation.Incoming is boolean

	if h.c.Capabilities.FilesSharing.SearchMinLength == 0 {
		h.c.Capabilities.FilesSharing.SearchMinLength = 2
	}

	// notifications

	// if h.c.Capabilities.Notifications == nil {
	// 	 h.c.Capabilities.Notifications = &ocs.CapabilitiesNotifications{}
	// }
	// if h.c.Capabilities.Notifications.Endpoints == nil {
	//    h.c.Capabilities.Notifications.Endpoints = []string{"list", "get", "delete"}
	//  }

	// version

	if h.c.Version == nil {
		h.c.Version = &ocs.Version{
			// TODO get from build env
			Major:          10,
			Minor:          0,
			Micro:          11,
			String:         "10.0.11",
			Edition:        "",
			Product:        "reva",
			ProductVersion: "",
		}
	}

	// upload protocol-specific details
	setCapabilitiesForChunkProtocol(chunkProtocol(h.defaultUploadProtocol), &h.c)

}

// Handler renders the capabilities
func (h *Handler) GetCapabilities(w http.ResponseWriter, r *http.Request) {
	c := h.getCapabilitiesForUserAgent(r.UserAgent())
	response.WriteOCSSuccess(w, r, c)
}
