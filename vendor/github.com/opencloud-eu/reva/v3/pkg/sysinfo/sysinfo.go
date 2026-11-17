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
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

// SystemInformation stores general information about Reva and the system it's running on.
type SystemInformation struct {
	// Reva holds the main Reva information
	Reva *RevaVersion `json:"reva"`
}

var (
	// SysInfo provides global system information.
	SysInfo = &SystemInformation{}
)

// ToJSON converts the system information to JSON.
func (sysInfo *SystemInformation) ToJSON() (string, error) {
	data, err := json.MarshalIndent(sysInfo, "", "\t")
	if err != nil {
		return "", fmt.Errorf("unable to marshal the system information: %v", err)
	}
	return string(data), nil
}

func replaceEmptyInfoValues(i interface{}) {
	// Iterate over each field of the given interface and search for "empty" values
	v := reflect.ValueOf(i).Elem()
	for i := 0; i < v.NumField(); i++ {
		// Check if the field is either a struct or a pointer to a struct; in that case, process the field recursively
		f := v.Field(i)
		if f.Kind() == reflect.Struct || (f.Kind() == reflect.Ptr && f.Elem().Kind() == reflect.Struct) {
			replaceEmptyInfoValues(f.Interface())
		} else if f.CanSet() { // Replace empty values with something more meaningful
			if f.Kind() == reflect.String {
				if len(f.String()) == 0 {
					f.SetString("(Unknown)")
				}
			}
		}
	}
}

// InitSystemInfo initializes the global system information object and also registers the corresponding metrics.
func InitSystemInfo(revaVersion *RevaVersion) error {
	SysInfo = &SystemInformation{
		Reva: revaVersion,
	}

	// Replace any empty values in the system information by more meaningful ones
	replaceEmptyInfoValues(SysInfo)

	// Register the system information metrics, as the necessary system info object has been filled out
	if err := registerSystemInfoMetrics(); err != nil {
		return errors.Wrap(err, "unable to register the system info metrics")
	}

	return nil
}
