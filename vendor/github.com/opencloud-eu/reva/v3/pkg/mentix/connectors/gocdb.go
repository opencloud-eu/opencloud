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

package connectors

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/opencloud-eu/reva/v3/pkg/mentix/utils"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v3/pkg/mentix/config"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/connectors/gocdb"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/meshdata"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/utils/network"
)

// GOCDBConnector is used to read mesh data from a GOCDB instance.
type GOCDBConnector struct {
	BaseConnector

	gocdbAddress string
}

// Activate activates the connector.
func (connector *GOCDBConnector) Activate(conf *config.Configuration, log *zerolog.Logger) error {
	if err := connector.BaseConnector.Activate(conf, log); err != nil {
		return err
	}

	// Check and store GOCDB specific settings
	connector.gocdbAddress = conf.Connectors.GOCDB.Address
	if len(connector.gocdbAddress) == 0 {
		return fmt.Errorf("no GOCDB address configured")
	}

	return nil
}

// RetrieveMeshData fetches new mesh data.
func (connector *GOCDBConnector) RetrieveMeshData() (*meshdata.MeshData, error) {
	meshData := new(meshdata.MeshData)

	// Query all data from GOCDB
	if err := connector.queryServiceTypes(meshData); err != nil {
		return nil, fmt.Errorf("could not query service types: %v", err)
	}

	if err := connector.querySites(meshData); err != nil {
		return nil, fmt.Errorf("could not query sites: %v", err)
	}

	for _, site := range meshData.Sites {
		// Get services associated with the current site
		if err := connector.queryServices(meshData, site); err != nil {
			return nil, fmt.Errorf("could not query services of site '%v': %v", site.Name, err)
		}

		// Get downtimes scheduled for the current site
		if err := connector.queryDowntimes(meshData, site); err != nil {
			return nil, fmt.Errorf("could not query downtimes of site '%v': %v", site.Name, err)
		}
	}

	meshData.InferMissingData()
	return meshData, nil
}

func (connector *GOCDBConnector) query(v interface{}, method string, isPrivate bool, hasScope bool, params network.URLParams) error {
	var scope string
	if hasScope {
		scope = connector.conf.Connectors.GOCDB.Scope
	}

	// Get the data from GOCDB
	data, err := gocdb.QueryGOCDB(connector.gocdbAddress, method, isPrivate, scope, connector.conf.Connectors.GOCDB.APIKey, params)
	if err != nil {
		return err
	}

	// Unmarshal it
	if err := xml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unable to unmarshal data: %v", err)
	}

	return nil
}

func (connector *GOCDBConnector) queryServiceTypes(meshData *meshdata.MeshData) error {
	var serviceTypes gocdb.ServiceTypes
	if err := connector.query(&serviceTypes, "get_service_types", false, false, network.URLParams{}); err != nil {
		return err
	}

	// Copy retrieved data into the mesh data
	meshData.ServiceTypes = nil
	for _, serviceType := range serviceTypes.Types {
		meshData.ServiceTypes = append(meshData.ServiceTypes, &meshdata.ServiceType{
			Name:        serviceType.Name,
			Description: serviceType.Description,
		})
	}

	return nil
}

func (connector *GOCDBConnector) querySites(meshData *meshdata.MeshData) error {
	var sites gocdb.Sites
	if err := connector.query(&sites, "get_site", false, true, network.URLParams{}); err != nil {
		return err
	}

	// Copy retrieved data into the mesh data
	meshData.Sites = nil
	for _, site := range sites.Sites {
		properties := connector.extensionsToMap(&site.Extensions)

		// The site ID can be set through a property; by default, the site short name will be used
		siteID := meshdata.GetPropertyValue(properties, meshdata.PropertySiteID, site.ShortName)

		// See if an organization has been defined using properties; otherwise, use the official name
		organization := meshdata.GetPropertyValue(properties, meshdata.PropertyOrganization, site.OfficialName)

		meshsite := &meshdata.Site{
			ID:           siteID,
			Name:         site.ShortName,
			FullName:     site.OfficialName,
			Organization: organization,
			Domain:       site.Domain,
			Homepage:     site.Homepage,
			Email:        site.Email,
			Description:  site.Description,
			Country:      site.Country,
			CountryCode:  site.CountryCode,
			Longitude:    site.Longitude,
			Latitude:     site.Latitude,
			Services:     nil,
			Properties:   properties,
			Downtimes:    meshdata.Downtimes{},
		}
		meshData.Sites = append(meshData.Sites, meshsite)
	}

	return nil
}

func (connector *GOCDBConnector) queryServices(meshData *meshdata.MeshData, site *meshdata.Site) error {
	var services gocdb.Services
	if err := connector.query(&services, "get_service", false, true, network.URLParams{"sitename": site.Name}); err != nil {
		return err
	}

	getServiceURLString := func(service *gocdb.Service, endpoint *gocdb.ServiceEndpoint, host string) string {
		urlstr := "https://" + host // Fall back to the provided hostname
		if svcURL, err := connector.getServiceURL(service, endpoint); err == nil {
			urlstr = svcURL.String()
		}
		return urlstr
	}

	// Copy retrieved data into the mesh data
	site.Services = nil
	for _, service := range services.Services {
		host := service.Host

		// If a URL is provided, extract the port from it and append it to the host
		if len(service.URL) > 0 {
			if hostURL, err := url.Parse(service.URL); err == nil {
				if port := hostURL.Port(); len(port) > 0 {
					host += ":" + port
				}
			}
		}

		// Assemble additional endpoints
		var endpoints []*meshdata.ServiceEndpoint
		for _, endpoint := range service.Endpoints.Endpoints {
			endpoints = append(endpoints, &meshdata.ServiceEndpoint{
				Type:        connector.findServiceType(meshData, endpoint.Type),
				Name:        endpoint.Name,
				RawURL:      endpoint.URL,
				URL:         getServiceURLString(service, endpoint, host),
				IsMonitored: strings.EqualFold(endpoint.IsMonitored, "Y"),
				Properties:  connector.extensionsToMap(&endpoint.Extensions),
			})
		}

		// Add the service to the site
		site.Services = append(site.Services, &meshdata.Service{
			ServiceEndpoint: &meshdata.ServiceEndpoint{
				Type:        connector.findServiceType(meshData, service.Type),
				Name:        service.Type,
				RawURL:      service.URL,
				URL:         getServiceURLString(service, nil, host),
				IsMonitored: strings.EqualFold(service.IsMonitored, "Y"),
				Properties:  connector.extensionsToMap(&service.Extensions),
			},
			Host:                host,
			AdditionalEndpoints: endpoints,
		})
	}

	return nil
}

func (connector *GOCDBConnector) queryDowntimes(meshData *meshdata.MeshData, site *meshdata.Site) error {
	var downtimes gocdb.Downtimes
	if err := connector.query(&downtimes, "get_downtime_nested_services", false, true, network.URLParams{"topentity": site.Name, "ongoing_only": "yes"}); err != nil {
		return err
	}

	// Copy retrieved data into the mesh data
	site.Downtimes.Clear()
	for _, dt := range downtimes.Downtimes {
		if !strings.EqualFold(dt.Severity, "outage") { // Only take real outages into account
			continue
		}

		services := make([]string, 0, len(dt.AffectedServices.Services))
		for _, service := range dt.AffectedServices.Services {
			// Only add critical services to the list of affected services
			if utils.FindInStringArray(service.Type, connector.conf.Services.CriticalTypes, false) != -1 {
				services = append(services, service.Type)
			}
		}

		_, _ = site.Downtimes.ScheduleDowntime(time.Unix(dt.StartDate, 0), time.Unix(dt.EndDate, 0), services)
	}

	return nil
}

func (connector *GOCDBConnector) findServiceType(meshData *meshdata.MeshData, name string) *meshdata.ServiceType {
	for _, serviceType := range meshData.ServiceTypes {
		if strings.EqualFold(serviceType.Name, name) {
			return serviceType
		}
	}

	// If the service type doesn't exist, create a default one
	return &meshdata.ServiceType{Name: name, Description: ""}
}

func (connector *GOCDBConnector) extensionsToMap(extensions *gocdb.Extensions) map[string]string {
	properties := make(map[string]string)
	for _, ext := range extensions.Extensions {
		properties[ext.Key] = ext.Value
	}
	return properties
}

func (connector *GOCDBConnector) getServiceURL(service *gocdb.Service, endpoint *gocdb.ServiceEndpoint) (*url.URL, error) {
	urlstr := service.URL
	if len(urlstr) == 0 {
		// The URL defaults to the hostname using the HTTPS protocol
		urlstr = "https://" + service.Host
	}

	svcURL, err := url.ParseRequestURI(urlstr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse URL '%v': %v", urlstr, err)
	}

	// If an endpoint was provided, use its path
	if endpoint != nil {
		// If the endpoint URL is an absolute one, just use that; otherwise, make an absolute one out of it
		if endpointURL, err := url.ParseRequestURI(endpoint.URL); err == nil && len(endpointURL.Scheme) > 0 {
			svcURL = endpointURL
		} else {
			// Replace entire URL path if the relative path starts with a slash; otherwise, just append
			if strings.HasPrefix(endpoint.URL, "/") {
				svcURL.Path = endpoint.URL
			} else {
				svcURL.Path = path.Join(svcURL.Path, endpoint.URL)
				if strings.HasSuffix(endpoint.URL, "/") { // Restore trailing slash if necessary
					svcURL.Path += "/"
				}
			}
		}
	}

	return svcURL, nil
}

// GetID returns the ID of the connector.
func (connector *GOCDBConnector) GetID() string {
	return config.ConnectorIDGOCDB
}

// GetName returns the display name of the connector.
func (connector *GOCDBConnector) GetName() string {
	return "GOCDB"
}

func init() {
	registerConnector(&GOCDBConnector{})
}
