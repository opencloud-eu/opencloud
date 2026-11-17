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

//go:build !windows
// +build !windows

package node

import (
	"syscall"
)

// GetAvailableSize stats the filesystem and return the available bytes
func GetAvailableSize(path string) (uint64, error) {
	stat := syscall.Statfs_t{}
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}

	// convert stat.Bavail to uint64 because it returns an int64 on freebsd
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil //nolint:unconvert
}
