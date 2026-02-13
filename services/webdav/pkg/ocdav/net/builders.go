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

package net

import (
	"net/url"
	"time"

	cs3types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// ContentDispositionAttachment builds a ContentDisposition Attachment header with various filename encodings
func ContentDispositionAttachment(filename string) string {
	return "attachment; filename*=UTF-8''" + url.PathEscape(filename) + "; filename=\"" + filename + "\""
}

// RFC1123Z formats a CS3 Timestamp to be used in HTTP headers like Last-Modified
func RFC1123Z(ts *cs3types.Timestamp) string {
	t := utils.TSToTime(ts).UTC()
	return t.Format(time.RFC1123Z)
}
