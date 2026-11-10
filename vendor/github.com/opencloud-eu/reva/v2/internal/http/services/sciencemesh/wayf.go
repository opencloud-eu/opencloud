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

package sciencemesh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/opencloud-eu/reva/v2/internal/http/services/ocmd"
	"github.com/opencloud-eu/reva/v2/internal/http/services/reqres"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
)

type wayfHandler struct {
	directoryServices []ocmd.DirectoryService
	ocmClient         *ocmd.OCMClient
	// for /discover, where the host comes from the request body, not from config
	untrustedClient *ocmd.OCMClient
}

type DiscoverRequest struct {
	Domain string `json:"domain"`
}

type DiscoverResponse struct {
	InviteAcceptDialog string `json:"inviteAcceptDialog"`
}

var (
	errInvalidProviderURL  = errors.New("invalid provider URL")
	errInvalidInviteDialog = errors.New("invalid invite accept dialog")
)

// resolveInviteAcceptDialog builds the invite page on the provider origin.
// The discovery value is a path. An absolute URL is accepted only when its
// scheme and host, including port, match the provider; the result is rebuilt
// from those fields and the path, never returned as received.
func resolveInviteAcceptDialog(providerBase, dialog string) (string, error) {
	base, err := url.Parse(providerBase)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" || base.User != nil {
		return "", errInvalidProviderURL
	}

	dialog = strings.TrimSpace(dialog)
	if dialog == "" || strings.ContainsAny(dialog, "\\#") || containsControl(dialog) {
		return "", errInvalidInviteDialog
	}
	ref, err := url.Parse(dialog)
	if err != nil || ref.Opaque != "" || ref.User != nil {
		return "", errInvalidInviteDialog
	}
	if (ref.Scheme != "" || ref.Host != "") &&
		(ref.Scheme != base.Scheme || !strings.EqualFold(ref.Host, base.Host)) {
		return "", errInvalidInviteDialog
	}

	p := ref.Path
	if ref.Scheme != "" && p == "" {
		p = "/"
	}
	if !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") ||
		strings.ContainsAny(p, "\\?#") || containsControl(p) {
		return "", errInvalidInviteDialog
	}
	p = path.Clean(p)
	if !strings.HasPrefix(p, "/") {
		return "", errInvalidInviteDialog
	}
	return (&url.URL{Scheme: base.Scheme, Host: base.Host, Path: p}).String(), nil
}

func containsControl(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

func (h *wayfHandler) init(c *config) error {
	log := appctx.GetLogger(context.Background())

	// Create OCM client for discovery from config
	h.ocmClient = ocmd.NewClient(time.Duration(c.OCMClientTimeout)*time.Second, c.OCMClientInsecure)
	h.untrustedClient = ocmd.NewPublicOnlyClient(time.Duration(c.OCMClientTimeout)*time.Second, c.OCMClientInsecure)
	log.Debug().
		Int("timeout_seconds", c.OCMClientTimeout).
		Bool("insecure", c.OCMClientInsecure).
		Msg("Created OCM client for discovery")

	urls := strings.Fields(c.DirectoryServiceURLs)
	if len(urls) == 0 {
		log.Info().Msg("No directory service URLs configured, starting with empty list")
		h.directoryServices = []ocmd.DirectoryService{}
		return nil
	}

	log.Debug().Int("url_count", len(urls)).Strs("urls", urls).Msg("Initializing WAYF handler with directory service URLs")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	h.directoryServices = []ocmd.DirectoryService{}
	discoveryErrors := 0
	validServersCount := 0
	fetchErrors := 0

	for _, directoryURL := range urls {
		log.Debug().Str("url", directoryURL).Msg("Fetching directory service")

		directoryService, err := h.ocmClient.GetDirectoryService(ctx, directoryURL)
		if err != nil {
			log.Info().Err(err).Str("url", directoryURL).Msg("Failed to fetch directory service, skipping")
			fetchErrors++
			continue
		}

		log.Debug().Str("federation", directoryService.Federation).Int("servers_count", len(directoryService.Servers)).Msg("Processing directory service")

		var validServers []ocmd.DirectoryServiceServer
		for _, srv := range directoryService.Servers {
			if srv.DisplayName == "" || srv.URL == "" {
				log.Debug().Str("federation", directoryService.Federation).
					Str("displayName", srv.DisplayName).
					Str("url", srv.URL).
					Msg("Skipping server with missing displayName or url")
				continue
			}

			log.Debug().Str("federation", directoryService.Federation).Str("server", srv.DisplayName).Str("url", srv.URL).Msg("Discovering server")

			// Discover inviteAcceptDialog from OCM endpoint
			disco, err := h.ocmClient.Discover(ctx, srv.URL)
			if err != nil {
				log.Debug().Err(err).
					Str("federation", directoryService.Federation).
					Str("server", srv.DisplayName).
					Str("url", srv.URL).
					Msg("Failed to discover server, skipping")
				discoveryErrors++
				continue
			}

			inviteDialog := disco.InviteAcceptDialog

			if inviteDialog != "" {
				resolved, err := resolveInviteAcceptDialog(srv.URL, inviteDialog)
				if err != nil {
					log.Debug().Err(err).
						Str("federation", directoryService.Federation).
						Str("server", srv.DisplayName).
						Str("url", srv.URL).
						Str("inviteDialog", disco.InviteAcceptDialog).
						Msg("Invalid invite accept dialog, skipping server")
					continue
				}
				if resolved != inviteDialog {
					log.Debug().Str("original", inviteDialog).Str("resolved", resolved).Msg("Resolved invite accept dialog")
				}
				inviteDialog = resolved
			}

			validServers = append(validServers, ocmd.DirectoryServiceServer{
				DisplayName:        srv.DisplayName,
				URL:                srv.URL,
				InviteAcceptDialog: inviteDialog,
			})
			validServersCount++

			log.Debug().
				Str("federation", directoryService.Federation).
				Str("server", srv.DisplayName).
				Str("inviteAcceptDialog", inviteDialog).
				Msg("Successfully discovered server")
		}

		if len(validServers) > 0 {
			h.directoryServices = append(h.directoryServices, ocmd.DirectoryService{
				Federation: directoryService.Federation,
				Servers:    validServers,
			})
			log.Debug().Str("federation", directoryService.Federation).Int("valid_servers", len(validServers)).Msg("Added directory service with valid servers")
		} else {
			log.Info().Str("federation", directoryService.Federation).
				Msg("Directory service has no valid servers, skipping entirely")
		}
	}

	log.Info().
		Int("directory_services", len(h.directoryServices)).
		Int("valid_servers", validServersCount).
		Int("fetch_errors", fetchErrors).
		Int("discovery_errors", discoveryErrors).
		Msg("WAYF handler initialization completed")

	return nil
}

func (h *wayfHandler) GetFederations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(h.directoryServices); err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error encoding response", err)
		return
	}
}

func (h *wayfHandler) DiscoverProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := appctx.GetLogger(ctx)

	var req DiscoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, "Invalid request body", err)
		return
	}

	if req.Domain == "" {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, "Domain is required", nil)
		return
	}

	domain := req.Domain
	if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
		domain = "https://" + domain
	}

	parsedURL, err := url.Parse(domain)
	if err != nil || parsedURL.Host == "" {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, "Invalid domain format", err)
		return
	}

	log.Debug().Str("domain", domain).Msg("Attempting OCM discovery")
	disco, err := h.untrustedClient.Discover(ctx, domain)
	if err != nil {
		log.Info().Err(err).Str("domain", domain).Msg("Discovery failed")
		reqres.WriteError(w, r, reqres.APIErrorNotFound,
			fmt.Sprintf("Provider at '%s' does not support OCM discovery", req.Domain), err)
		return
	}

	inviteDialog, err := resolveInviteAcceptDialog(domain, disco.InviteAcceptDialog)
	if err != nil {
		log.Info().Err(err).Str("domain", domain).Str("inviteDialog", disco.InviteAcceptDialog).Msg("Provider does not provide a usable invite accept dialog")
		if errors.Is(err, errInvalidProviderURL) {
			reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, "Invalid domain format", err)
			return
		}
		reqres.WriteError(w, r, reqres.APIErrorNotFound,
			fmt.Sprintf("Provider at '%s' does not provide an invite accept dialog", req.Domain), err)
		return
	}

	response := DiscoverResponse{
		InviteAcceptDialog: inviteDialog,
	}

	log.Info().
		Str("domain", req.Domain).
		Str("inviteAcceptDialog", inviteDialog).
		Msg("Discovery successful")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "Error encoding response", err)
		return
	}
}
