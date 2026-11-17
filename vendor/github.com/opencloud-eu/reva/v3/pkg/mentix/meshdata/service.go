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

package meshdata

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/opencloud-eu/reva/v3/pkg/mentix/utils/network"
)

// Service represents a service managed by Mentix.
type Service struct {
	*ServiceEndpoint

	Host                string
	AdditionalEndpoints []*ServiceEndpoint
}

// FindEndpoint searches for an additional endpoint with the given name.
func (service *Service) FindEndpoint(name string) *ServiceEndpoint {
	for _, endpoint := range service.AdditionalEndpoints {
		if strings.EqualFold(endpoint.Name, name) {
			return endpoint
		}
	}

	return nil
}

// InferMissingData infers missing data from other data where possible.
func (service *Service) InferMissingData() {
	service.ServiceEndpoint.InferMissingData()

	// Infer missing data
	if service.Host == "" {
		if serviceURL, err := url.Parse(service.URL); err == nil {
			service.Host = network.ExtractDomainFromURL(serviceURL, true)
		}
	}
}

// Verify checks if the service data is valid.
func (service *Service) Verify() error {
	if err := service.ServiceEndpoint.Verify(); err != nil {
		return err
	}

	return nil
}

// ServiceType represents a service type managed by Mentix.
type ServiceType struct {
	Name        string
	Description string
}

// InferMissingData infers missing data from other data where possible.
func (serviceType *ServiceType) InferMissingData() {
	// Infer missing data
	if serviceType.Description == "" {
		serviceType.Description = serviceType.Name
	}
}

// Verify checks if the service type data is valid.
func (serviceType *ServiceType) Verify() error {
	// Verify data
	if serviceType.Name == "" {
		return fmt.Errorf("service type name missing")
	}

	return nil
}

// ServiceEndpoint represents a service endpoint managed by Mentix.
type ServiceEndpoint struct {
	Type        *ServiceType
	Name        string
	RawURL      string
	URL         string
	IsMonitored bool
	Properties  map[string]string
}

// InferMissingData infers missing data from other data where possible.
func (serviceEndpoint *ServiceEndpoint) InferMissingData() {
}

// Verify checks if the service endpoint data is valid.
func (serviceEndpoint *ServiceEndpoint) Verify() error {
	if serviceEndpoint.Type == nil {
		return fmt.Errorf("service endpoint type missing")
	}
	if serviceEndpoint.Name == "" {
		return fmt.Errorf("service endpoint name missing")
	}
	if serviceEndpoint.URL == "" {
		return fmt.Errorf("service endpoint URL missing")
	}

	return nil
}
