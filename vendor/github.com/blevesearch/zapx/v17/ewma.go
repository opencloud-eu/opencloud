//  Copyright (c) 2026 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zap

// ewma maintains an exponentially weighted moving average of the number of
// hits a cache entry receives per monitoring cycle.
type ewma struct {
	// alpha is the smoothing factor in (0, 1): the weight given to the most
	// recent sample. Higher values react faster to changes in traffic, lower
	// values retain history longer.
	alpha float64
	// avg is the current moving average of hits per cycle.
	avg float64
	// every hit to the cache entry is recorded as part of a sample
	// which will be used to calculate the average in the next cycle of average
	// computation (which is average traffic for the field till now). this is
	// used to track the per second hits to the cache entries.
	sample uint64
}

// add folds the latest cycle's hit count into the moving average:
//
//	X(t) = a.v + (1 - a).X(t-1)
//
// The first non-zero sample seeds the average directly rather than being
// smoothed from zero. A zero-hit cycle multiplies the average by (1 - alpha),
// so an average of 1 hit per cycle decays to exactly (1 - alpha) after one
// idle cycle — which is why the cache cleanup passes avg <= (1 - alpha)
// as the eviction threshold: it means the entry is averaging less than
// one hit per cycle.
func (e *ewma) add(val uint64) {
	if e.avg == 0.0 {
		e.avg = float64(val)
	} else {
		// the exponentially weighted moving average
		// X(t) = a.v + (1 - a).X(t-1)
		e.avg = e.alpha*float64(val) + (1-e.alpha)*e.avg
	}
}
