// Copyright 2018-2022 CERN
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

// Package bytesize provides easy conversions from human readable strings (eg. 10MB) to bytes
package bytesize

import (
	"fmt"
	"strconv"
	"strings"
)

// ByteSize is the size in bytes
type ByteSize uint64

// List of available byte sizes
// NOTE: max is exabyte as we convert to uint64
const (
	KB ByteSize = 1000
	MB ByteSize = 1000 * KB
	GB ByteSize = 1000 * MB
	TB ByteSize = 1000 * GB
	PB ByteSize = 1000 * TB
	EB ByteSize = 1000 * PB

	KiB ByteSize = 1024
	MiB ByteSize = 1024 * KiB
	GiB ByteSize = 1024 * MiB
	TiB ByteSize = 1024 * GiB
	PiB ByteSize = 1024 * TiB
	EiB ByteSize = 1024 * PiB
)

// Parse parses a Bytesize from a string
func Parse(s string) (ByteSize, error) {
	sanitized := strings.TrimSpace(s)
	if !strings.HasSuffix(sanitized, "B") {
		u, err := strconv.Atoi(sanitized)
		return ByteSize(u), err
	}

	var (
		value int
		unit  string
	)

	template := "%d%s"
	_, err := fmt.Sscanf(sanitized, template, &value, &unit)
	if err != nil {
		return 0, err
	}

	bytes := ByteSize(value)
	switch unit {
	case "KB":
		bytes *= KB
	case "MB":
		bytes *= MB
	case "GB":
		bytes *= GB
	case "TB":
		bytes *= TB
	case "PB":
		bytes *= PB
	case "EB":
		bytes *= EB
	case "KiB":
		bytes *= KiB
	case "MiB":
		bytes *= MiB
	case "GiB":
		bytes *= GiB
	case "TiB":
		bytes *= TiB
	case "PiB":
		bytes *= PiB
	case "EiB":
		bytes *= EiB
	default:
		return 0, fmt.Errorf("unknown unit '%s'. Use common abbreviations such as KB, MiB, GB", unit)
	}

	return bytes, nil
}

// Bytes converts the ByteSize to an uint64
func (b ByteSize) Bytes() uint64 {
	return uint64(b)
}

// String converts the ByteSize to a string
func (b ByteSize) String() string {
	return strconv.FormatUint(uint64(b), 10)
}
