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

package wellknown

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
)

const OCMAPIVersion = "1.2.0"

type OcmProviderConfig struct {
	OCMPrefix          string `docs:"ocm;The prefix URL where the OCM API is served."                                            mapstructure:"ocm_prefix"`
	Endpoint           string `docs:"This host's full URL. If it's not configured, it is assumed OCM is not available."          mapstructure:"endpoint"`
	Provider           string `docs:"reva;A friendly name that defines this service."                                            mapstructure:"provider"`
	WebdavRoot         string `docs:"/remote.php/dav/ocm;The root URL of the WebDAV endpoint to serve OCM shares."               mapstructure:"webdav_root"`
	WebappRoot         string `docs:"/external/sciencemesh;The root URL to serve Web apps via OCM."                              mapstructure:"webapp_root"`
	InviteAcceptDialog string `docs:"/open-cloud-mesh/accept-invite;The frontend URL where to land when receiving an invitation" mapstructure:"invite_accept_dialog"`
	EnableWebapp       bool   `docs:"false;Whether web apps are enabled in OCM shares."                                          mapstructure:"enable_webapp"`
	EnableDatatx       bool   `docs:"false;Whether data transfers are enabled in OCM shares."                                    mapstructure:"enable_datatx"`
}

type OcmDiscoveryData struct {
	Enabled            bool            `json:"enabled"            xml:"enabled"`
	APIVersion         string          `json:"apiVersion"         xml:"apiVersion"`
	Endpoint           string          `json:"endPoint"           xml:"endPoint"`
	Provider           string          `json:"provider"           xml:"provider"`
	ResourceTypes      []resourceTypes `json:"resourceTypes"      xml:"resourceTypes"`
	Capabilities       []string        `json:"capabilities"       xml:"capabilities"`
	InviteAcceptDialog string          `json:"inviteAcceptDialog" xml:"inviteAcceptDialog"`
}

type resourceTypes struct {
	Name       string            `json:"name"`
	ShareTypes []string          `json:"shareTypes"`
	Protocols  map[string]string `json:"protocols"`
}

type wkocmHandler struct {
	data *OcmDiscoveryData
}

func (c *OcmProviderConfig) ApplyDefaults() {
	if c.OCMPrefix == "" {
		c.OCMPrefix = "ocm"
	}
	if c.Provider == "" {
		c.Provider = "reva"
	}
	if c.WebdavRoot == "" {
		c.WebdavRoot = "/remote.php/dav/ocm/"
	}
	if c.WebdavRoot[len(c.WebdavRoot)-1:] != "/" {
		c.WebdavRoot += "/"
	}
	if c.WebappRoot == "" {
		c.WebappRoot = "/external/sciencemesh/"
	}
	if c.WebappRoot[len(c.WebappRoot)-1:] != "/" {
		c.WebappRoot += "/"
	}
	if c.InviteAcceptDialog == "" {
		c.InviteAcceptDialog = "/open-cloud-mesh/accept-invite"
	}
}

func (h *wkocmHandler) init(c *OcmProviderConfig) {
	// generates the (static) data structure to be exposed by /.well-known/ocm:
	// first prepare an empty and disabled payload
	c.ApplyDefaults()
	d := &OcmDiscoveryData{}
	d.Enabled = false
	d.Endpoint = ""
	d.APIVersion = OCMAPIVersion
	d.Provider = c.Provider
	d.ResourceTypes = []resourceTypes{{
		Name:       "file",
		ShareTypes: []string{},
		Protocols:  map[string]string{},
	}}
	d.Capabilities = []string{}

	if c.Endpoint == "" {
		h.data = d
		return
	}

	endpointURL, err := url.Parse(c.Endpoint)
	if err != nil {
		h.data = d
		return
	}

	// now prepare the enabled one
	d.Enabled = true
	d.Endpoint, _ = url.JoinPath(c.Endpoint, c.OCMPrefix)
	rtProtos := map[string]string{}
	// webdav is always enabled
	rtProtos["webdav"] = filepath.Join(endpointURL.Path, c.WebdavRoot)
	if c.EnableWebapp {
		rtProtos["webapp"] = filepath.Join(endpointURL.Path, c.WebappRoot)
	}
	if c.EnableDatatx {
		rtProtos["datatx"] = filepath.Join(endpointURL.Path, c.WebdavRoot)
	}
	d.ResourceTypes = []resourceTypes{{
		Name:       "file",           // so far we only support `file`
		ShareTypes: []string{"user"}, // so far we only support `user`
		Protocols:  rtProtos,         // expose the protocols as per configuration
	}}
	// for now, we hardcoded the capabilities, as this is currently only advisory
	d.Capabilities = []string{"invites", "webdav-uri", "protocol-object", "invite-wayf"}
	d.InviteAcceptDialog, _ = url.JoinPath(c.Endpoint, c.InviteAcceptDialog)
	h.data = d
}

// Ocm handles the OCM discovery endpoint specified in
// https://cs3org.github.io/OCM-API/docs.html?repo=OCM-API&user=cs3org#/paths/~1ocm-provider/get
func (h *wkocmHandler) Ocm(w http.ResponseWriter, r *http.Request) {
	log := appctx.GetLogger(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if r.UserAgent() == "Nextcloud Server Crawler" {
		// Nextcloud decided to only support OCM 1.0 and 1.1, not any 1.x as per SemVer. See
		// https://github.com/nextcloud/server/pull/39574#issuecomment-1679191188
		h.data.APIVersion = "1.1"
	} else {
		h.data.APIVersion = OCMAPIVersion
	}
	indented, _ := json.MarshalIndent(h.data, "", "   ")
	if _, err := w.Write(indented); err != nil {
		log.Err(err).Msg("Error writing to ResponseWriter")
	}
}
