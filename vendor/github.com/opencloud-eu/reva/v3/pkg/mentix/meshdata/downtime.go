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
	"time"

	"github.com/pkg/errors"
)

// Downtimes represents all scheduled downtimes of a site.
type Downtimes struct {
	Downtimes []*Downtime
}

// ScheduleDowntime schedules a new downtime.
func (dts *Downtimes) ScheduleDowntime(start time.Time, end time.Time, affectedServices []string) (*Downtime, error) {
	// Create a new downtime and verify it
	dt := &Downtime{
		StartDate:        start,
		EndDate:          end,
		AffectedServices: affectedServices,
	}
	dt.InferMissingData()
	if err := dt.Verify(); err != nil {
		return nil, err
	}

	// Only schedule the downtime if it hasn't expired yet
	if dt.IsExpired() {
		return nil, nil
	}

	dts.Downtimes = append(dts.Downtimes, dt)
	return dt, nil
}

// Clear clears all downtimes.
func (dts *Downtimes) Clear() {
	dts.Downtimes = make([]*Downtime, 0, 10)
}

// IsAnyActive returns true if any downtime is currently active.
func (dts *Downtimes) IsAnyActive() bool {
	for _, dt := range dts.Downtimes {
		if dt.IsActive() {
			return true
		}
	}
	return false
}

// InferMissingData infers missing data from other data where possible.
func (dts *Downtimes) InferMissingData() {
	for _, dt := range dts.Downtimes {
		dt.InferMissingData()
	}
}

// Verify checks if the downtimes data is valid.
func (dts *Downtimes) Verify() error {
	for _, dt := range dts.Downtimes {
		if err := dt.Verify(); err != nil {
			return err
		}
	}

	return nil
}

// Downtime represents a single scheduled downtime.
type Downtime struct {
	StartDate        time.Time
	EndDate          time.Time
	AffectedServices []string
}

// IsActive returns true if the downtime is currently active.
func (dt *Downtime) IsActive() bool {
	now := time.Now()
	return dt.StartDate.Before(now) && dt.EndDate.After(now)
}

// IsPending returns true if the downtime is yet to come.
func (dt *Downtime) IsPending() bool {
	return dt.StartDate.After(time.Now())
}

// IsExpired returns true of the downtime has expired (i.e., lies in the past).
func (dt *Downtime) IsExpired() bool {
	return dt.EndDate.Before(time.Now())
}

// InferMissingData infers missing data from other data where possible.
func (dt *Downtime) InferMissingData() {
}

// Verify checks if the downtime data is valid.
func (dt *Downtime) Verify() error {
	if dt.EndDate.Before(dt.StartDate) {
		return errors.Errorf("downtime end is before its start")
	}
	if len(dt.AffectedServices) == 0 {
		return errors.Errorf("no services affected by downtime")
	}

	return nil
}
