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

package manager

// ErrMaxFileCount is the error returned when the max files count specified in the config has reached
type ErrMaxFileCount struct{}

// ErrMaxSize is the error returned when the max total files size specified in the config has reached
type ErrMaxSize struct{}

// ErrEmptyList is the error returned when an empty list is passed when an archiver is created
type ErrEmptyList struct{}

// Error returns the string error msg for ErrMaxFileCount
func (ErrMaxFileCount) Error() string {
	return "reached max files count"
}

// Error returns the string error msg for ErrMaxSize
func (ErrMaxSize) Error() string {
	return "reached max total files size"
}

// Error returns the string error msg for ErrEmptyList
func (ErrEmptyList) Error() string {
	return "list of files to archive empty"
}
