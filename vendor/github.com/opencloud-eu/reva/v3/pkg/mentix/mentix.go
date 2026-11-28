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

package mentix

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/accservice"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/config"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/connectors"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/entity"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/exchangers"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/exchangers/exporters"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/exchangers/importers"
	"github.com/opencloud-eu/reva/v3/pkg/mentix/meshdata"
)

// Mentix represents the main Mentix service object.
type Mentix struct {
	conf *config.Configuration
	log  *zerolog.Logger

	connectors *connectors.Collection
	importers  *importers.Collection
	exporters  *exporters.Collection

	meshDataSet meshdata.Map

	updateInterval time.Duration
}

const (
	runLoopSleeptime = time.Millisecond * 1000
)

func (mntx *Mentix) initialize(conf *config.Configuration, log *zerolog.Logger) error {
	if conf == nil {
		return fmt.Errorf("no configuration provided")
	}
	mntx.conf = conf

	if log == nil {
		return fmt.Errorf("no logger provided")
	}
	mntx.log = log

	// Initialize the connectors that will be used to gather the mesh data
	if err := mntx.initConnectors(); err != nil {
		return fmt.Errorf("unable to initialize connector: %v", err)
	}

	// Initialize the exchangers
	if err := mntx.initExchangers(); err != nil {
		return fmt.Errorf("unable to initialize exchangers: %v", err)
	}

	// Get the update interval
	duration, err := time.ParseDuration(mntx.conf.UpdateInterval)
	if err != nil {
		// If the duration can't be parsed, default to one hour
		duration = time.Hour
	}
	mntx.updateInterval = duration

	// Create empty mesh data set
	mntx.meshDataSet = make(meshdata.Map)

	// Log some infos
	connectorNames := entity.GetNames(mntx.connectors)
	importerNames := entity.GetNames(mntx.importers)
	exporterNames := entity.GetNames(mntx.exporters)
	log.Info().Msgf("mentix started with connectors: %v; importers: %v; exporters: %v; update interval: %v",
		strings.Join(connectorNames, ", "),
		strings.Join(importerNames, ", "),
		strings.Join(exporterNames, ", "),
		duration,
	)

	return nil
}

func (mntx *Mentix) initConnectors() error {
	// Use all connectors exposed by the connectors package
	conns, err := connectors.AvailableConnectors(mntx.conf)
	if err != nil {
		return fmt.Errorf("unable to get registered conns: %v", err)
	}
	mntx.connectors = conns

	if err := mntx.connectors.ActivateAll(mntx.conf, mntx.log); err != nil {
		return fmt.Errorf("unable to activate connectors: %v", err)
	}

	return nil
}

func (mntx *Mentix) initExchangers() error {
	// Use all importers exposed by the importers package
	imps, err := importers.AvailableImporters(mntx.conf)
	if err != nil {
		return fmt.Errorf("unable to get registered importers: %v", err)
	}
	mntx.importers = imps

	if err := mntx.importers.ActivateAll(mntx.conf, mntx.log); err != nil {
		return fmt.Errorf("unable to activate importers: %v", err)
	}

	// Use all exporters exposed by the exporters package
	exps, err := exporters.AvailableExporters(mntx.conf)
	if err != nil {
		return fmt.Errorf("unable to get registered exporters: %v", err)
	}
	mntx.exporters = exps

	if err := mntx.exporters.ActivateAll(mntx.conf, mntx.log); err != nil {
		return fmt.Errorf("unable to activate exporters: %v", err)
	}

	return nil
}

func (mntx *Mentix) startExchangers() error {
	// Start all importers
	if err := mntx.importers.StartAll(); err != nil {
		return fmt.Errorf("unable to start importers: %v", err)
	}

	// Start all exporters
	if err := mntx.exporters.StartAll(); err != nil {
		return fmt.Errorf("unable to start exporters: %v", err)
	}

	return nil
}

func (mntx *Mentix) stopExchangers() {
	mntx.exporters.StopAll()
	mntx.importers.StopAll()
}

func (mntx *Mentix) destroy() {
	mntx.stopExchangers()
}

// Run starts the Mentix service that will periodically pull the configured data source and publish this data
// through the enabled exporters.
func (mntx *Mentix) Run(stopSignal <-chan struct{}) error {
	defer mntx.destroy()

	// Start all im- & exporters; they will be stopped in mntx.destroy
	if err := mntx.startExchangers(); err != nil {
		return fmt.Errorf("unable to start exchangers: %v", err)
	}

	updateTimestamp := time.Time{}
loop:
	for {
		if stopSignal != nil {
			// Poll the stopSignal channel; if a signal was received, break the loop, terminating Mentix gracefully
			select {
			case <-stopSignal:
				break loop

			default:
			}
		}

		// Perform all regular actions
		mntx.tick(&updateTimestamp)

		time.Sleep(runLoopSleeptime)
	}

	return nil
}

func (mntx *Mentix) tick(updateTimestamp *time.Time) {
	// Let all importers do their work first
	meshDataUpdated, err := mntx.processImporters()
	if err != nil {
		mntx.log.Err(err).Msgf("an error occurred while processing the importers: %v", err)
	}

	// If mesh data has been imported or enough time has passed, update the stored mesh data and all exporters
	if meshDataUpdated || time.Since(*updateTimestamp) >= mntx.updateInterval {
		// Retrieve and update the mesh data; if the importers modified any data, these changes will
		// be reflected automatically here
		if meshDataSet, err := mntx.retrieveMeshDataSet(); err == nil {
			if err := mntx.applyMeshDataSet(meshDataSet); err != nil {
				mntx.log.Err(err).Msg("failed to apply mesh data")
			}
		} else {
			mntx.log.Err(err).Msg("failed to retrieve mesh data")
		}

		*updateTimestamp = time.Now()
	}
}

func (mntx *Mentix) processImporters() (bool, error) {
	meshDataUpdated := false

	for _, importer := range mntx.importers.Importers {
		updated, err := importer.Process(mntx.connectors)
		if err != nil {
			return false, fmt.Errorf("unable to process importer '%v': %v", importer.GetName(), err)
		}
		meshDataUpdated = meshDataUpdated || updated

		if updated {
			mntx.log.Debug().Msgf("mesh data imported from '%v'", importer.GetName())
		}
	}

	return meshDataUpdated, nil
}

func (mntx *Mentix) retrieveMeshDataSet() (meshdata.Map, error) {
	meshDataSet := make(meshdata.Map)

	for _, connector := range mntx.connectors.Connectors {
		meshData, err := connector.RetrieveMeshData()
		if err == nil {
			meshDataSet[connector.GetID()] = meshData
		} else {
			mntx.log.Err(err).Msgf("retrieving mesh data from connector '%v' failed", connector.GetName())
		}
	}

	return meshDataSet, nil
}

func (mntx *Mentix) applyMeshDataSet(meshDataSet meshdata.Map) error {
	// Check if mesh data from any connector has changed
	meshDataChanged := false
	for connectorID, meshData := range meshDataSet {
		if !meshData.Compare(mntx.meshDataSet[connectorID]) {
			meshDataChanged = true
			break
		}
	}

	if meshDataChanged {
		mntx.log.Debug().Msg("mesh data changed, applying")

		mntx.meshDataSet = meshDataSet

		exchangers := make([]exchangers.Exchanger, 0, len(mntx.exporters.Exporters)+len(mntx.importers.Importers))
		exchangers = append(exchangers, mntx.exporters.Exchangers()...)
		exchangers = append(exchangers, mntx.importers.Exchangers()...)

		for _, exchanger := range exchangers {
			if err := exchanger.Update(mntx.meshDataSet); err != nil {
				return fmt.Errorf("unable to update mesh data on exchanger '%v': %v", exchanger.GetName(), err)
			}
		}
	}

	return nil
}

// GetRequestImporters returns all exporters that can handle HTTP requests.
func (mntx *Mentix) GetRequestImporters() []exchangers.RequestExchanger {
	return mntx.importers.GetRequestImporters()
}

// GetRequestExporters returns all exporters that can handle HTTP requests.
func (mntx *Mentix) GetRequestExporters() []exchangers.RequestExchanger {
	return mntx.exporters.GetRequestExporters()
}

// RequestHandler handles any incoming HTTP requests by asking each RequestExchanger whether it wants to
// handle the request (usually based on the relative URL path).
func (mntx *Mentix) RequestHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	log := appctx.GetLogger(r.Context())

	switch r.Method {
	case http.MethodGet:
		mntx.handleRequest(mntx.GetRequestExporters(), w, r, log)

	case http.MethodPost:
		mntx.handleRequest(mntx.GetRequestImporters(), w, r, log)

	default:
		log.Err(fmt.Errorf("unsupported method")).Msg("error handling incoming request")
	}
}

func (mntx *Mentix) handleRequest(exchangers []exchangers.RequestExchanger, w http.ResponseWriter, r *http.Request, log *zerolog.Logger) {
	// Ask each RequestExchanger if it wants to handle the request
	for _, exchanger := range exchangers {
		if exchanger.WantsRequest(r) {
			exchanger.HandleRequest(w, r, mntx.conf, log)
		}
	}
}

// New creates a new Mentix service instance.
func New(conf *config.Configuration, log *zerolog.Logger) (*Mentix, error) {
	// Configure the accounts service upfront
	if err := accservice.InitAccountsService(conf); err != nil {
		return nil, fmt.Errorf("unable to initialize the accounts service: %v", err)
	}

	mntx := new(Mentix)
	if err := mntx.initialize(conf, log); err != nil {
		return nil, fmt.Errorf("unable to initialize Mentix: %v", err)
	}
	return mntx, nil
}
