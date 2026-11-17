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

package exporters

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/opencloud-eu/reva/v3/pkg/mentix/utils"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v3/pkg/mentix/config"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/exchangers/exporters/prometheus"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/meshdata"
)

type prometheusSDScrapeCreatorCallback = func(site *meshdata.Site, service *meshdata.Service, endpoint *meshdata.ServiceEndpoint) *prometheus.ScrapeConfig
type prometheusSDScrapeCreator struct {
	outputFilename  string
	creatorCallback prometheusSDScrapeCreatorCallback
	serviceFilter   []string
}

// PrometheusSDExporter implements various Prometheus Service Discovery scrape config exporters.
type PrometheusSDExporter struct {
	BaseExporter

	scrapeCreators map[string]prometheusSDScrapeCreator
}

const (
	labelSiteName    = "__meta_mentix_site"
	labelSiteID      = "__meta_mentix_site_id"
	labelSiteCountry = "__meta_mentix_site_country"
	labelType        = "__meta_mentix_type"
	labelURL         = "__meta_mentix_url"
	labelScheme      = "__meta_mentix_scheme"
	labelHost        = "__meta_mentix_host"
	labelPort        = "__meta_mentix_port"
	labelPath        = "__meta_mentix_path"
	labelServiceHost = "__meta_mentix_service_host"
	labelServiceURL  = "__meta_mentix_service_url"
)

func createGenericScrapeConfig(site *meshdata.Site, service *meshdata.Service, endpoint *meshdata.ServiceEndpoint) *prometheus.ScrapeConfig {
	endpointURL, _ := url.Parse(endpoint.URL)
	labels := getScrapeTargetLabels(site, service, endpoint)
	return &prometheus.ScrapeConfig{
		Targets: []string{endpointURL.Host},
		Labels:  labels,
	}
}
func getScrapeTargetLabels(site *meshdata.Site, service *meshdata.Service, endpoint *meshdata.ServiceEndpoint) map[string]string {
	endpointURL, _ := url.Parse(endpoint.URL)
	labels := map[string]string{
		labelSiteName:    site.Name,
		labelSiteID:      site.ID,
		labelSiteCountry: site.CountryCode,
		labelType:        endpoint.Type.Name,
		labelURL:         endpoint.URL,
		labelScheme:      endpointURL.Scheme,
		labelHost:        endpointURL.Hostname(),
		labelPort:        endpointURL.Port(),
		labelPath:        endpointURL.Path,
		labelServiceHost: service.Host,
		labelServiceURL:  service.URL,
	}

	return labels
}

func (exporter *PrometheusSDExporter) registerScrapeCreators(conf *config.Configuration) error {
	exporter.scrapeCreators = make(map[string]prometheusSDScrapeCreator)

	registerCreator := func(name string, outputFilename string, creator prometheusSDScrapeCreatorCallback, serviceFilter []string) error {
		if len(outputFilename) > 0 { // Only register the creator if an output filename was configured
			exporter.scrapeCreators[name] = prometheusSDScrapeCreator{
				outputFilename:  outputFilename,
				creatorCallback: creator,
				serviceFilter:   serviceFilter,
			}

			// Create the output directory for the target file so it exists when exporting
			if err := os.MkdirAll(filepath.Dir(outputFilename), 0755); err != nil {
				return fmt.Errorf("unable to create output directory tree: %v", err)
			}
		}

		return nil
	}

	// Register all scrape creators
	for _, endpoint := range meshdata.GetServiceEndpoints() {
		epName := strings.ToLower(endpoint)
		filename := path.Join(conf.Exporters.PrometheusSD.OutputPath, "svc_"+epName+".json")

		if err := registerCreator(epName, filename, createGenericScrapeConfig, []string{endpoint}); err != nil {
			return fmt.Errorf("unable to register the '%v' scrape config creator: %v", epName, err)
		}
	}

	return nil
}

// Activate activates the exporter.
func (exporter *PrometheusSDExporter) Activate(conf *config.Configuration, log *zerolog.Logger) error {
	if err := exporter.BaseExporter.Activate(conf, log); err != nil {
		return err
	}

	if err := exporter.registerScrapeCreators(conf); err != nil {
		return fmt.Errorf("unable to register the scrape creators: %v", err)
	}

	// Create all output directories
	for _, creator := range exporter.scrapeCreators {
		if err := os.MkdirAll(filepath.Dir(creator.outputFilename), 0755); err != nil {
			return fmt.Errorf("unable to create directory tree: %v", err)
		}
	}

	// Store PrometheusSD specifics
	exporter.SetEnabledConnectors(conf.Exporters.PrometheusSD.EnabledConnectors)

	return nil
}

// Update is called whenever the mesh data set has changed to reflect these changes.
func (exporter *PrometheusSDExporter) Update(meshDataSet meshdata.Map) error {
	if err := exporter.BaseExporter.Update(meshDataSet); err != nil {
		return err
	}

	// Perform exporting the data asynchronously
	go exporter.exportMeshData()
	return nil
}

func (exporter *PrometheusSDExporter) exportMeshData() {
	// Data is read, so acquire a read lock
	exporter.Locker().RLock()
	defer exporter.Locker().RUnlock()

	for name, creator := range exporter.scrapeCreators {
		scrapes := exporter.createScrapeConfigs(creator.creatorCallback, creator.serviceFilter)
		if err := exporter.exportScrapeConfig(creator.outputFilename, scrapes); err != nil {
			exporter.Log().Err(err).Str("kind", name).Str("file", creator.outputFilename).Msg("error exporting Prometheus SD scrape config")
		} else {
			exporter.Log().Debug().Str("kind", name).Str("file", creator.outputFilename).Msg("exported Prometheus SD scrape config")
		}
	}
}

func (exporter *PrometheusSDExporter) createScrapeConfigs(creatorCallback prometheusSDScrapeCreatorCallback, serviceFilter []string) []*prometheus.ScrapeConfig {
	var scrapes []*prometheus.ScrapeConfig
	var addScrape = func(site *meshdata.Site, service *meshdata.Service, endpoint *meshdata.ServiceEndpoint) {
		if len(serviceFilter) == 0 || utils.FindInStringArray(endpoint.Type.Name, serviceFilter, false) != -1 {
			if scrape := creatorCallback(site, service, endpoint); scrape != nil {
				scrapes = append(scrapes, scrape)
			}
		}
	}

	// Create a scrape config for each service alongside any additional endpoints
	for _, site := range exporter.MeshData().Sites {
		for _, service := range site.Services {
			if !service.IsMonitored {
				continue
			}

			// Add the "main" service to the scrapes
			addScrape(site, service, service.ServiceEndpoint)

			// Add all additional endpoints as well
			for _, endpoint := range service.AdditionalEndpoints {
				if endpoint.IsMonitored {
					addScrape(site, service, endpoint)
				}
			}
		}
	}

	if scrapes == nil {
		scrapes = []*prometheus.ScrapeConfig{}
	}

	return scrapes
}

func (exporter *PrometheusSDExporter) exportScrapeConfig(outputFilename string, v interface{}) error {
	// Encode scrape config as JSON
	data, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return fmt.Errorf("unable to marshal scrape config: %v", err)
	}

	// Write the data to disk
	if err := os.WriteFile(outputFilename, data, 0755); err != nil {
		return fmt.Errorf("unable to write scrape config '%v': %v", outputFilename, err)
	}

	return nil
}

// GetID returns the ID of the exporter.
func (exporter *PrometheusSDExporter) GetID() string {
	return config.ExporterIDPrometheusSD
}

// GetName returns the display name of the exporter.
func (exporter *PrometheusSDExporter) GetName() string {
	return "Prometheus SD"
}

func init() {
	registerExporter(&PrometheusSDExporter{})
}
