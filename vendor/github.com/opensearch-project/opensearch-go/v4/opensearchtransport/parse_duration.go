// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import (
	"strconv"
	"time"
)

// parseDuration parses a duration string with flexible format support.
// It tries, in order:
//  1. time.ParseDuration format (e.g., "30s", "1m", "500ms")
//  2. Integer seconds via strconv.ParseInt (e.g., "30" -> 30s)
//  3. Fractional seconds via strconv.ParseFloat (e.g., "1.5" -> 1.5s)
//
// Returns the parsed duration and true on success, or zero and false
// if all parse attempts fail.
func parseDuration(s string) (time.Duration, bool) {
	if d, err := time.ParseDuration(s); err == nil {
		return d, true
	}
	if secs, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Duration(secs) * time.Second, true
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return time.Duration(f * float64(time.Second)), true
	}
	return 0, false
}
