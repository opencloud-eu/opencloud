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

package common

import (
	"os"
	"time"
)

// DataDescriptor implements os.FileInfo to provide file information for non-file data objects.
// This is used, for example, when uploading data that doesn't come from a local file.
type DataDescriptor struct {
	name string
	size int64
}

// Name returns the quasi-filename of this object.
func (ddesc *DataDescriptor) Name() string {
	return ddesc.name
}

// Size returns the specified data size.
func (ddesc *DataDescriptor) Size() int64 {
	return ddesc.size
}

// Mode always returns a 0700 file mode.
func (ddesc *DataDescriptor) Mode() os.FileMode {
	return 0700
}

// ModTime always returns the current time as the modification time.
func (ddesc *DataDescriptor) ModTime() time.Time {
	return time.Now()
}

// IsDir always returns false.
func (ddesc *DataDescriptor) IsDir() bool {
	return false
}

// Sys returns nil, as this object doesn't represent a system object.
func (ddesc *DataDescriptor) Sys() interface{} {
	return nil
}

// CreateDataDescriptor creates a new descriptor for non-file data objects.
func CreateDataDescriptor(name string, size int64) DataDescriptor {
	return DataDescriptor{
		name: name,
		size: size,
	}
}
