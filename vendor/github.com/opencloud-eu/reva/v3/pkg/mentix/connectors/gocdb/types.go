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

package gocdb

// Extension represents Key-Value pairs in GOCDB.
type Extension struct {
	Key   string `xml:"KEY"`
	Value string `xml:"VALUE"`
}

// Extensions is a list of Extension objects.
type Extensions struct {
	Extensions []*Extension `xml:"EXTENSION"`
}

// ServiceType represents a service type in GOCDB.
type ServiceType struct {
	Name        string `xml:"SERVICE_TYPE_NAME"`
	Description string `xml:"SERVICE_TYPE_DESC"`
}

// ServiceTypes is a list of ServiceType objects.
type ServiceTypes struct {
	Types []*ServiceType `xml:"SERVICE_TYPE"`
}

// Site represents a site in GOCDB.
type Site struct {
	ShortName    string     `xml:"SHORT_NAME"`
	OfficialName string     `xml:"OFFICIAL_NAME"`
	Description  string     `xml:"SITE_DESCRIPTION"`
	Homepage     string     `xml:"HOME_URL"`
	Email        string     `xml:"CONTACT_EMAIL"`
	Domain       string     `xml:"DOMAIN>DOMAIN_NAME"`
	Country      string     `xml:"COUNTRY"`
	CountryCode  string     `xml:"COUNTRY_CODE"`
	Latitude     float32    `xml:"LATITUDE"`
	Longitude    float32    `xml:"LONGITUDE"`
	Extensions   Extensions `xml:"EXTENSIONS"`
}

// Sites is a list of Site objects.
type Sites struct {
	Sites []*Site `xml:"SITE"`
}

// ServiceEndpoint represents an additional service endpoint of a service in GOCDB.
type ServiceEndpoint struct {
	Name        string     `xml:"NAME"`
	URL         string     `xml:"URL"`
	Type        string     `xml:"INTERFACENAME"`
	IsMonitored string     `xml:"ENDPOINT_MONITORED"`
	Extensions  Extensions `xml:"EXTENSIONS"`
}

// ServiceEndpoints is a list of ServiceEndpoint objects.
type ServiceEndpoints struct {
	Endpoints []*ServiceEndpoint `xml:"ENDPOINT"`
}

// Service represents a service in GOCDB.
type Service struct {
	Host        string           `xml:"HOSTNAME"`
	Type        string           `xml:"SERVICE_TYPE"`
	IsMonitored string           `xml:"NODE_MONITORED"`
	URL         string           `xml:"URL"`
	Endpoints   ServiceEndpoints `xml:"ENDPOINTS"`
	Extensions  Extensions       `xml:"EXTENSIONS"`
}

// Services is a list of Service objects.
type Services struct {
	Services []*Service `xml:"SERVICE_ENDPOINT"`
}

// DowntimeService represents a service scheduled for downtime.
type DowntimeService struct {
	Type string `xml:"SERVICE_TYPE"`
}

// DowntimeServices represents a list of DowntimeService objects.
type DowntimeServices struct {
	Services []*DowntimeService `xml:"SERVICE"`
}

// Downtime is a scheduled downtime for a site.
type Downtime struct {
	Severity  string `xml:"SEVERITY"`
	StartDate int64  `xml:"START_DATE"`
	EndDate   int64  `xml:"END_DATE"`

	AffectedServices DowntimeServices `xml:"SERVICES"`
}

// Downtimes represents a list of Downtime objects.
type Downtimes struct {
	Downtimes []*Downtime `xml:"DOWNTIME"`
}
