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

package ocdav

import (
	"strings"
	"testing"
)

func TestUnmarshallReportFilterFiles(t *testing.T) {
	ffXML := `<oc:filter-files  xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns">
    <d:prop>
        <d:getlastmodified />
        <d:getetag />
        <d:getcontenttype />
        <d:resourcetype />
        <oc:fileid />
        <oc:permissions />
        <oc:size />
        <d:getcontentlength />
        <oc:tags />
        <oc:favorite />
        <d:lockdiscovery />
        <oc:comments-unread />
        <oc:owner-display-name />
        <oc:share-types />
    </d:prop>
    <oc:filter-rules>
        <oc:favorite>1</oc:favorite>
    </oc:filter-rules>
</oc:filter-files>`

	reader := strings.NewReader(ffXML)

	report, status, err := readReport(reader)
	if status != 0 || err != nil {
		t.Error("Failed to unmarshal filter-files xml")
	}

	if report.FilterFiles == nil {
		t.Error("Failed to unmarshal filter-files xml. FilterFiles is nil")
	}

	if report.FilterFiles.Rules.Favorite == false {
		t.Error("Failed to correctly unmarshal filter-rules. Favorite is expected to be true.")
	}
}
