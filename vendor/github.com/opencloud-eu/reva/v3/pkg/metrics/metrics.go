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

package metrics

/*
Metrics registers OpenCensus data views of the metrics.
Metrics initializes the driver as specified in the configuration.
*/
import (
	"context"
	"os"
	"time"

	"github.com/opencloud-eu/reva/v3/pkg/metrics/config"
	"github.com/opencloud-eu/reva/v3/pkg/metrics/driver/registry"
	"github.com/opencloud-eu/reva/v3/pkg/metrics/reader"

	"github.com/opencloud-eu/reva/v3/pkg/logger"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

// Init intializes metrics according to the specified configuration
func Init(conf *config.Config) error {
	log := logger.New().With().Int("pid", os.Getpid()).Logger()

	driver := registry.GetDriver(conf.MetricsDataDriverType)

	if driver == nil {
		log.Info().Msg("No metrics are being recorded.")
		// No error but just don't proceed with metrics
		return nil
	}

	// configure the driver
	err := driver.Configure(conf)
	if err != nil {
		return err
	}

	m := &Metrics{
		dataDriver:           driver,
		NumUsersMeasure:      stats.Int64("cs3_org_sciencemesh_site_total_num_users", "The total number of users within this site", stats.UnitDimensionless),
		NumGroupsMeasure:     stats.Int64("cs3_org_sciencemesh_site_total_num_groups", "The total number of groups within this site", stats.UnitDimensionless),
		AmountStorageMeasure: stats.Int64("cs3_org_sciencemesh_site_total_amount_storage", "The total amount of storage used within this site", stats.UnitBytes),
	}

	if err := view.Register(
		m.getNumUsersView(),
		m.getNumGroupsView(),
		m.getAmountStorageView(),
	); err != nil {
		return err
	}

	// periodically record metrics data
	go func() {
		for {
			if err := m.recordMetrics(); err != nil {
				log.Error().Err(err).Msg("Metrics recording failed.")
			}
			<-time.After(time.Millisecond * time.Duration(conf.MetricsRecordInterval))
		}
	}()

	return nil
}

// Metrics the metrics struct
type Metrics struct {
	dataDriver           reader.Reader // the metrics data driver is an implemention of Reader
	NumUsersMeasure      *stats.Int64Measure
	NumGroupsMeasure     *stats.Int64Measure
	AmountStorageMeasure *stats.Int64Measure
}

// RecordMetrics records the latest metrics from the metrics data source as OpenCensus stats views.
func (m *Metrics) recordMetrics() error {
	// record all latest metrics
	if m.dataDriver != nil {
		m.recordNumUsers()
		m.recordNumGroups()
		m.recordAmountStorage()
	}
	return nil
}

// recordNumUsers records the latest number of site users figure
func (m *Metrics) recordNumUsers() {
	ctx := context.Background()
	stats.Record(ctx, m.NumUsersMeasure.M(m.dataDriver.GetNumUsers()))
}

func (m *Metrics) getNumUsersView() *view.View {
	return &view.View{
		Name:        m.NumUsersMeasure.Name(),
		Description: m.NumUsersMeasure.Description(),
		Measure:     m.NumUsersMeasure,
		Aggregation: view.LastValue(),
	}
}

// recordNumGroups records the latest number of site groups figure
func (m *Metrics) recordNumGroups() {
	ctx := context.Background()
	stats.Record(ctx, m.NumGroupsMeasure.M(m.dataDriver.GetNumGroups()))
}

func (m *Metrics) getNumGroupsView() *view.View {
	return &view.View{
		Name:        m.NumGroupsMeasure.Name(),
		Description: m.NumGroupsMeasure.Description(),
		Measure:     m.NumGroupsMeasure,
		Aggregation: view.LastValue(),
	}
}

// recordAmountStorage records the latest amount storage figure
func (m *Metrics) recordAmountStorage() {
	ctx := context.Background()
	stats.Record(ctx, m.AmountStorageMeasure.M(m.dataDriver.GetAmountStorage()))
}

func (m *Metrics) getAmountStorageView() *view.View {
	return &view.View{
		Name:        m.AmountStorageMeasure.Name(),
		Description: m.AmountStorageMeasure.Description(),
		Measure:     m.AmountStorageMeasure,
		Aggregation: view.LastValue(),
	}
}
