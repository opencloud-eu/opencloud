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

package gateway

import (
	"context"
	"net/url"
	"path"
	"strings"

	ocmprovider "github.com/cs3org/go-cs3apis/cs3/ocm/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/pkg/errors"
)

type webdavEndpoint struct {
	filePath string
	endpoint string
	token    string
}

func (s *svc) extractEndpointInfo(ctx context.Context, targetURL string) (*webdavEndpoint, error) {
	if targetURL == "" {
		return nil, errtypes.BadRequest("gateway: ref target is an empty uri")
	}

	uri, err := url.Parse(targetURL)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error parsing target uri: "+targetURL)
	}
	if uri.Scheme != "webdav" {
		return nil, errtypes.NotSupported("ref target does not have the webdav scheme")
	}

	m, err := url.ParseQuery(uri.RawQuery)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error parsing target resource name")
	}

	return &webdavEndpoint{
		filePath: m["name"][0],
		endpoint: uri.Host,
		token:    uri.User.String(),
	}, nil
}

func (s *svc) getWebdavEndpoint(ctx context.Context, domain string) (string, error) {
	meshProvider, err := s.GetInfoByDomain(ctx, &ocmprovider.GetInfoByDomainRequest{
		Domain: domain,
	})
	if err != nil {
		return "", errors.Wrap(err, "gateway: error calling GetInfoByDomain")
	}
	for _, s := range meshProvider.ProviderInfo.Services {
		if strings.ToLower(s.Endpoint.Type.Name) == "webdav" {
			return s.Endpoint.Path, nil
		}
	}
	return "", errtypes.NotFound(domain)
}

func (s *svc) getWebdavHost(ctx context.Context, domain string) (string, error) {
	meshProvider, err := s.GetInfoByDomain(ctx, &ocmprovider.GetInfoByDomainRequest{
		Domain: domain,
	})
	if err != nil {
		return "", errors.Wrap(err, "gateway: error calling GetInfoByDomain")
	}
	for _, s := range meshProvider.ProviderInfo.Services {
		if strings.ToLower(s.Endpoint.Type.Name) == "webdav" {
			return s.Host, nil
		}
	}
	return "", errtypes.NotFound(domain)
}

func appendNameQuery(targetURL string, nameQueries ...string) (string, error) {
	uri, err := url.Parse(targetURL)
	if err != nil {
		return "", err
	}
	q, err := url.ParseQuery(uri.RawQuery)
	if err != nil {
		return "", err
	}
	name := append([]string{q["name"][0]}, nameQueries...)
	q.Set("name", path.Join(name...))
	uri.RawQuery = q.Encode()
	return uri.String(), nil
}
