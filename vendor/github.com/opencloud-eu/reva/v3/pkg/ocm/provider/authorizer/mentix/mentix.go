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

package mentix

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	ocmprovider "github.com/cs3org/go-cs3apis/cs3/ocm/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/ocm/provider"
	"github.com/opencloud-eu/reva/v3/pkg/ocm/provider/authorizer/registry"
	"github.com/opencloud-eu/reva/v3/pkg/rhttp"
	"github.com/opencloud-eu/reva/v3/pkg/utils/cfg"
	"github.com/pkg/errors"
)

func init() {
	registry.Register("mentix", New)
}

// Client is a Mentix API client.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// New returns a new authorizer object.
func New(m map[string]interface{}) (provider.Authorizer, error) {
	var c config
	if err := cfg.Decode(m, &c); err != nil {
		return nil, err
	}

	client := &Client{
		BaseURL: c.URL,
		HTTPClient: rhttp.GetHTTPClient(
			rhttp.Context(context.Background()),
			rhttp.Timeout(time.Duration(c.Timeout*int64(time.Second))),
			rhttp.Insecure(c.Insecure),
		),
	}

	return &authorizer{
		client:      client,
		providerIPs: sync.Map{},
		conf:        &c,
	}, nil
}

type config struct {
	URL                   string `mapstructure:"url"`
	Timeout               int64  `mapstructure:"timeout"`
	RefreshInterval       int64  `mapstructure:"refresh"`
	VerifyRequestHostname bool   `mapstructure:"verify_request_hostname"`
	Insecure              bool   `mapstructure:"insecure" docs:"false;Whether to skip certificate checks when sending requests."`
}

func (c *config) ApplyDefaults() {
	if c.URL == "" {
		c.URL = "http://localhost:9600/mentix/cs3"
	}
}

type authorizer struct {
	providers           []*ocmprovider.ProviderInfo
	providersExpiration int64
	client              *Client
	providerIPs         sync.Map
	conf                *config
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

	return u.Hostname(), nil
}

func (a *authorizer) fetchProviders() ([]*ocmprovider.ProviderInfo, error) {
	if (a.providers != nil) && (time.Now().Unix() < a.providersExpiration) {
		return a.providers, nil
	}

	req, err := http.NewRequest(http.MethodGet, a.client.BaseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	res, err := a.client.HTTPClient.Do(req)
	if err != nil {
		err = errors.Wrap(err,
			fmt.Sprintf("mentix: error fetching provider list from: %s", a.client.BaseURL))
		return nil, err
	}

	defer res.Body.Close()

	providers := make([]*ocmprovider.ProviderInfo, 0)
	if err = json.NewDecoder(res.Body).Decode(&providers); err != nil {
		return nil, err
	}

	a.providers = a.getOCMProviders(providers)
	if a.conf.RefreshInterval > 0 {
		a.providersExpiration = time.Now().Unix() + a.conf.RefreshInterval
	}
	return a.providers, nil
}

func (a *authorizer) GetInfoByDomain(ctx context.Context, domain string) (*ocmprovider.ProviderInfo, error) {
	normalizedDomain, err := normalizeDomain(domain)
	if err != nil {
		return nil, err
	}

	providers, err := a.fetchProviders()
	if err != nil {
		return nil, err
	}
	for _, p := range providers {
		if strings.Contains(p.Domain, normalizedDomain) {
			return p, nil
		}
	}
	return nil, errtypes.NotFound(domain)
}

func (a *authorizer) IsProviderAllowed(ctx context.Context, pi *ocmprovider.ProviderInfo) error {
	providers, err := a.fetchProviders()
	if err != nil {
		return err
	}
	normalizedDomain, err := normalizeDomain(pi.Domain)
	if err != nil {
		return err
	}

	var providerAuthorized bool
	if normalizedDomain != "" {
		for _, p := range providers {
			if p.Domain == normalizedDomain {
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
		return errtypes.NotSupported(
			fmt.Sprintf("mentix: provider %s has no supported services", pi.GetDomain()))
	}

	var ocmHost string
	for _, p := range providers {
		if p.Domain == normalizedDomain {
			ocmHost, err = a.getOCMHost(p)
			if err != nil {
				return err
			}
			break
		}
	}
	if ocmHost == "" {
		return errtypes.NotSupported(
			fmt.Sprintf("mentix: provider %s is missing OCM endpoint", pi.GetDomain()))
	}

	providerAuthorized = false
	var ipList []string
	if hostIPs, ok := a.providerIPs.Load(ocmHost); ok {
		ipList = hostIPs.([]string)
	} else {
		addr, err := net.LookupIP(ocmHost)
		if err != nil {
			return errors.Wrap(err,
				fmt.Sprintf("mentix: error looking up IPs for OCM endpoint %s", ocmHost))
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
		return errtypes.BadRequest(
			fmt.Sprintf(
				"Invalid requesting OCM endpoint IP %s of provider %s",
				pi.Services[0].Host, pi.GetDomain()))
	}

	return nil
}

func (a *authorizer) ListAllProviders(ctx context.Context) ([]*ocmprovider.ProviderInfo, error) {
	providers, err := a.fetchProviders()
	if err != nil {
		return nil, err
	}
	return providers, nil
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

func (a *authorizer) getOCMHost(provider *ocmprovider.ProviderInfo) (string, error) {
	for _, s := range provider.Services {
		if s.Endpoint.Type.Name == "OCM" {
			return s.Host, nil
		}
	}
	return "", errtypes.NotFound("OCM Host")
}
