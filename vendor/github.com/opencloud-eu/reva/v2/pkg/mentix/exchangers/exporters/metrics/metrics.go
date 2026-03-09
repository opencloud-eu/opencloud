// Copyright 2018-2020 CERN
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

import (
	"context"

	"github.com/opencloud-eu/reva/v2/pkg/mentix/config"
	"github.com/opencloud-eu/reva/v2/pkg/mentix/meshdata"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Metrics exposes various Mentix related metrics via Prometheus.
type Metrics struct {
	conf *config.Configuration
	log  *zerolog.Logger

	isScheduledStats *stats.Int64Measure
}

const (
	keySiteID      = "site_id"
	keySiteName    = "site"
	keyServiceType = "service_type"
)

func (m *Metrics) initialize(conf *config.Configuration, log *zerolog.Logger) error {
	if conf == nil {
		return errors.Errorf("no configuration provided")
	}
	m.conf = conf

	if log == nil {
		return errors.Errorf("no logger provided")
	}
	m.log = log

	if err := m.registerMetrics(); err != nil {
		return errors.Wrap(err, "error while registering metrics")
	}

	return nil
}

func (m *Metrics) registerMetrics() error {
	// Create the OpenCensus statistics and a corresponding view
	m.isScheduledStats = stats.Int64("site_is_scheduled", "A boolean metric which shows whether the given site is currently scheduled or not", stats.UnitDimensionless)
	isScheduledView := &view.View{
		Name:        m.isScheduledStats.Name(),
		Description: m.isScheduledStats.Description(),
		Measure:     m.isScheduledStats,
		TagKeys:     []tag.Key{tag.MustNewKey(keySiteID), tag.MustNewKey(keySiteName), tag.MustNewKey(keyServiceType)},
		Aggregation: view.LastValue(),
	}

	if err := view.Register(isScheduledView); err != nil {
		return errors.Wrap(err, "unable to register the site schedule status metrics view")
	}

	return nil
}

// Update is used to update/expose all metrics.
func (m *Metrics) Update(meshData *meshdata.MeshData) error {
	for _, site := range meshData.Sites {
		if err := m.exportSiteMetrics(site); err != nil {
			return errors.Wrapf(err, "error while exporting metrics for site '%v'", site.Name)
		}
	}

	return nil
}

func (m *Metrics) exportSiteMetrics(site *meshdata.Site) error {
	mutators := []tag.Mutator{
		tag.Insert(tag.MustNewKey(keySiteID), site.ID),
		tag.Insert(tag.MustNewKey(keySiteName), site.Name),
		tag.Insert(tag.MustNewKey(keyServiceType), "SCIENCEMESH_HCHECK"),
	}

	// Create a new context to serve the metrics
	if ctx, err := tag.New(context.Background(), mutators...); err == nil {
		isScheduled := int64(1)
		if site.Downtimes.IsAnyActive() {
			isScheduled = 0
		}
		stats.Record(ctx, m.isScheduledStats.M(isScheduled))
	} else {
		return errors.Wrap(err, "unable to create a context for the site schedule status metrics")
	}

	return nil
}

// New creates a new Metrics instance.
func New(conf *config.Configuration, log *zerolog.Logger) (*Metrics, error) {
	m := &Metrics{}
	if err := m.initialize(conf, log); err != nil {
		return nil, errors.Wrap(err, "unable to create new metrics object")
	}
	return m, nil
}
