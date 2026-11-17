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

package sysinfo

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/opencloud-eu/reva/v3/pkg/utils"
)

type sysInfoMetricsLabels = map[tag.Key]string

func registerSystemInfoMetrics() error {
	labels := getSystemInfoMetricsLabels("", SysInfo)

	// Collect all labels and their values; the values are stored as mutators
	tagKeys := make([]tag.Key, 0, len(labels))
	mutators := make([]tag.Mutator, 0, len(labels))
	for key, value := range labels {
		tagKeys = append(tagKeys, key)
		mutators = append(mutators, tag.Insert(key, value))
	}

	// Create the OpenCensus statistics and a corresponding view
	sysInfoStats := stats.Int64("sys_info", "A metric with a constant '1' value labeled by various system information elements", stats.UnitDimensionless)
	sysInfoView := &view.View{
		Name:        sysInfoStats.Name(),
		Description: sysInfoStats.Description(),
		Measure:     sysInfoStats,
		TagKeys:     tagKeys,
		Aggregation: view.LastValue(),
	}

	if err := view.Register(sysInfoView); err != nil {
		return errors.Wrap(err, "unable to register the system info metrics view")
	}

	// Create a new context to serve the metrics
	if ctx, err := tag.New(context.Background(), mutators...); err == nil {
		// Just record a simple hardcoded '1' to expose the system info as a metric
		stats.Record(ctx, sysInfoStats.M(1))
	} else {
		return errors.Wrap(err, "unable to create a context for the system info metrics")
	}

	return nil
}

func getSystemInfoMetricsLabels(root string, i interface{}) sysInfoMetricsLabels {
	labels := sysInfoMetricsLabels{}

	// Iterate over each field of the given interface, recursively collecting the values as labels
	v := reflect.ValueOf(i).Elem()
	for i := 0; i < v.NumField(); i++ {
		// Check if the field was tagged with 'sysinfo:omitlabel'; if so, skip this field
		tags := v.Type().Field(i).Tag.Get("sysinfo")
		if strings.Contains(tags, "omitlabel") {
			continue
		}

		// Get the name of the field from the parent structure
		fieldName := utils.ToSnakeCase(v.Type().Field(i).Name)
		if len(root) > 0 {
			fieldName = "_" + fieldName
		}
		fieldName = root + fieldName

		// Check if the field is either a struct or a pointer to a struct; in that case, process the field recursively
		f := v.Field(i)
		if f.Kind() == reflect.Struct || (f.Kind() == reflect.Ptr && f.Elem().Kind() == reflect.Struct) {
			// Merge labels recursively
			for key, val := range getSystemInfoMetricsLabels(fieldName, f.Interface()) {
				labels[key] = val
			}
		} else { // Store the value of the field in the labels
			key := tag.MustNewKey(fieldName)
			labels[key] = fmt.Sprintf("%v", f)
		}
	}

	return labels
}
