// Copyright 2025 CERN
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

package ocmd

// DirectoryService represents a directory service listing per OCM spec Appendix C.
type DirectoryService struct {
	Federation string                   `json:"federation"`
	Servers    []DirectoryServiceServer `json:"servers"`
}

// DirectoryServiceServer represents a single OCM server in a directory service.
type DirectoryServiceServer struct {
	DisplayName string `json:"displayName"`
	URL         string `json:"url"`
	// Added after discovery, not in raw response
	InviteAcceptDialog string `json:"inviteAcceptDialog,omitempty"`
}
