// Copyright 2018-2023 CERN
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

package json

import (
	"context"
	"encoding/json"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"

	ocmprovider "github.com/cs3org/go-cs3apis/cs3/ocm/provider/v1beta1"
	"github.com/pkg/errors"

	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/ocm/provider"
	"github.com/opencloud-eu/reva/v3/pkg/ocm/provider/authorizer/registry"
	"github.com/opencloud-eu/reva/v3/pkg/utils/cfg"
)

func init() {
	registry.Register("json", New)
}

var (
	ErrNoIP = errtypes.NotSupported("No IP provided")
)

// New returns a new authorizer object.
func New(m map[string]interface{}) (provider.Authorizer, error) {
	var c config
	if err := cfg.Decode(m, &c); err != nil {
		return nil, err
	}

	providers := []*ocmprovider.ProviderInfo{}
	f, err := os.ReadFile(c.Providers)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		err = json.Unmarshal(f, &providers)
		if err != nil {
			return nil, err
		}
	}

	a := &authorizer{
		providerIPs: sync.Map{},
		conf:        &c,
	}
	a.providers = a.getOCMProviders(providers)

	return a, nil
}

type config struct {
	Providers             string `mapstructure:"providers"`
	VerifyRequestHostname bool   `mapstructure:"verify_request_hostname"`
}

func (c *config) ApplyTemplates() {
	if c.Providers == "" {
		c.Providers = "/etc/revad/ocm-providers.json"
	}
}

type authorizer struct {
	providers   []*ocmprovider.ProviderInfo
	providerIPs sync.Map
	conf        *config
}

func normalizeDomain(d string) (string, error) {
	var urlString string
	if strings.Contains(d, "://") {
		urlString = d
	} else {
		urlString = "https://" + d
	}

	u, err := url.Parse(urlString)
	if err != nil {
		return "", err
	}

	return u.Host, nil
}

func (a *authorizer) GetInfoByDomain(_ context.Context, domain string) (*ocmprovider.ProviderInfo, error) {
	normalizedDomain, err := normalizeDomain(domain)
	if err != nil {
		return nil, err
	}
	for _, p := range a.providers {
		// we can exit early if this an exact match
		if strings.Contains(p.Domain, normalizedDomain) {
			return p, nil
		}

		// check if the domain matches a regex
		if ok, err := regexp.MatchString(p.Domain, normalizedDomain); ok && err == nil {
			// overwrite wildcards with the actual domain
			var services []*ocmprovider.Service
			for _, s := range p.Services {
				services = append(services, &ocmprovider.Service{
					Host: strings.ReplaceAll(s.Host, p.Domain, normalizedDomain),
					Endpoint: &ocmprovider.ServiceEndpoint{
						Type:        s.Endpoint.Type,
						Name:        s.Endpoint.Name,
						Path:        strings.ReplaceAll(s.Endpoint.Path, p.Domain, normalizedDomain),
						IsMonitored: s.Endpoint.IsMonitored,
						Properties:  s.Endpoint.Properties,
					},
					ApiVersion:          s.ApiVersion,
					AdditionalEndpoints: s.AdditionalEndpoints,
				})
			}
			return &ocmprovider.ProviderInfo{
				Name:         p.Name,
				FullName:     p.FullName,
				Description:  p.Description,
				Organization: p.Organization,
				Domain:       normalizedDomain,
				Homepage:     p.Homepage,
				Email:        p.Email,
				Services:     services,
				Properties:   p.Properties,
			}, nil
		}
	}
	return nil, errtypes.NotFound(domain)
}

func (a *authorizer) IsProviderAllowed(ctx context.Context, pi *ocmprovider.ProviderInfo) error {
	log := appctx.GetLogger(ctx)
	var err error
	normalizedDomain, err := normalizeDomain(pi.Domain)
	if err != nil {
		return err
	}
	var providerAuthorized bool
	if normalizedDomain != "" {
		for _, p := range a.providers {
			if ok, err := regexp.MatchString(p.Domain, normalizedDomain); ok && err == nil {
				providerAuthorized = true
				break
			}
		}
	} else {
		providerAuthorized = true
	}

	switch {
	case !providerAuthorized:
		return errtypes.NotFound(pi.GetDomain())
	case !a.conf.VerifyRequestHostname:
		return nil
	case len(pi.Services) == 0:
		return ErrNoIP
	}

	var ocmHost string
	for _, p := range a.providers {
		log.Debug().Msgf("Comparing '%s' to '%s'", p.Domain, normalizedDomain)
		if p.Domain == normalizedDomain {
			ocmHost, err = a.getOCMHost(p)
			if err != nil {
				return err
			}
			break
		}
	}
	if ocmHost == "" {
		return errtypes.InternalError("json: ocm host not specified for mesh provider")
	}

	providerAuthorized = false
	var ipList []string
	if hostIPs, ok := a.providerIPs.Load(ocmHost); ok {
		ipList = hostIPs.([]string)
	} else {
		host, _, err := net.SplitHostPort(ocmHost)
		if err != nil {
			return errors.Wrap(err, "json: error looking up client IP")
		}
		addr, err := net.LookupIP(host)
		if err != nil {
			return errors.Wrap(err, "json: error looking up client IP")
		}
		for _, a := range addr {
			ipList = append(ipList, a.String())
		}
		a.providerIPs.Store(ocmHost, ipList)
	}

	for _, ip := range ipList {
		if ip == pi.Services[0].Host {
			providerAuthorized = true
			break
		}
	}
	if !providerAuthorized {
		return errtypes.NotFound("OCM Host")
	}

	return nil
}

func (a *authorizer) ListAllProviders(ctx context.Context) ([]*ocmprovider.ProviderInfo, error) {
	return a.providers, nil
}

func (a *authorizer) getOCMProviders(providers []*ocmprovider.ProviderInfo) (po []*ocmprovider.ProviderInfo) {
	for _, p := range providers {
		_, err := a.getOCMHost(p)
		if err == nil {
			po = append(po, p)
		}
	}
	return
}

func (a *authorizer) getOCMHost(pi *ocmprovider.ProviderInfo) (string, error) {
	for _, s := range pi.Services {
		if s.Endpoint.Type.Name == "OCM" {
			return s.Host, nil
		}
	}
	return "", errtypes.NotFound("OCM Host")
}
