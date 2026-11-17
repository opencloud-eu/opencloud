// Copyright 2018-2023 CERN
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

package nextcloud

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
)

// Response contains data for the Nextcloud mock server to respond
// and to switch to a new server state.
type Response struct {
	code           int
	body           string
	newServerState string
}

const serverStateError = "ERROR"
const serverStateEmpty = "EMPTY"
const serverStateHome = "HOME"

var serverState = serverStateEmpty

var responses = map[string]Response{
	`POST /apps/sciencemesh/~tester/api/ocm/addSentShare {"resourceId":{"opaqueId":"fileid-/some/path"},"name":"Some Name","permissions":{"permissions":{"getPath":true}},"grantee":{"userId":{"idp":"0.0.0.0:19000","opaqueId":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":"USER_TYPE_PRIMARY"},"opaque":{"map":{"sharedSecret":{"decoder":"plain","value":"bGdFU1hjR0VtTg=="}}}},"owner":{"idp":"0.0.0.0:19000","opaqueId":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":"USER_TYPE_PRIMARY"},"creator":{"idp":"0.0.0.0:19000","opaqueId":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":"USER_TYPE_PRIMARY"},"shareType":"SHARE_TYPE_REGULAR"}`: {200, `{"id":{},"resource_id":{},"permissions":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890}}`, serverStateHome},
	`POST /apps/sciencemesh/~tester/api/ocm/addReceivedShare {"md":{"opaque_id":"fileid-/some/path"},"g":{"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"permissions":{"permissions":{"get_path":true}}},"provider_domain":"cern.ch","resource_type":"file","provider_id":2,"owner_opaque_id":"einstein","owner_display_name":"Albert Einstein","protocol":{"name":"webdav","options":{"sharedSecret":"secret","permissions":"webdav-property"}}}`:                                                                                                                                {200, `{"id":{},"resource_id":{},"permissions":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890}}`, serverStateHome},
	`POST /apps/sciencemesh/~tester/api/ocm/GetShare {"Spec":{"Id":{"opaque_id":"some-share-id"}}}`: {200, `{"id":{},"resource_id":{},"permissions":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890}}`, serverStateHome},
	`POST /apps/sciencemesh/~tester/api/ocm/Unshare {"Spec":{"Id":{"opaque_id":"some-share-id"}}}`:  {200, ``, serverStateHome},
	`POST /apps/sciencemesh/~tester/api/ocm/UpdateShare {"ref":{"Spec":{"Id":{"opaque_id":"some-share-id"}}},"p":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}}}`: {200, `{"id":{},"resource_id":{},"permissions":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890}}`, serverStateHome},
	`POST /apps/sciencemesh/~tester/api/ocm/ListShares [{"type":4,"Term":{"Creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}}]`: {200, `[{"id":{},"resource_id":{},"permissions":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890}}]`, serverStateHome},
	`POST /apps/sciencemesh/~tester/api/ocm/ListReceivedShares `:                                            {200, `[{"share":{"id":{},"resource_id":{},"permissions":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890}},"state":2}]`, serverStateHome},
	`POST /apps/sciencemesh/~tester/api/ocm/GetReceivedShare {"Spec":{"Id":{"opaque_id":"some-share-id"}}}`: {200, `{"share":{"id":{},"resource_id":{},"permissions":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890}},"state":2}`, serverStateHome},
	`POST /apps/sciencemesh/~tester/api/ocm/UpdateReceivedShare {"received_share":{"id":{},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890},"state":2},"field_mask":{"paths":["state"]}}`: {200, `{"share":{"id":{},"permissions":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890}},"state":2}`, serverStateHome},

	`POST /index.php/apps/sciencemesh/~marie/api/ocm/addReceivedShare {"md":{"opaque_id":"fileid-/some/path"},"g":{"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"permissions":{"permissions":{"get_path":true}}},"provider_domain":"cern.ch","resource_type":"file","provider_id":2,"owner_opaque_id":"einstein","owner_display_name":"Albert Einstein","protocol":{"name":"webdav","options":{"sharedSecret":"secret","permissions":"webdav-property"}}}`: {200, `{"id":{},"resource_id":{},"permissions":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890}}`, serverStateHome},
	`POST /index.php/apps/sciencemesh/~marie/api/ocm/GetShare {"Spec":{"Id":{"opaque_id":"some-share-id"}}}`: {200, `{"id":{},"resource_id":{},"permissions":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890}}`, serverStateHome},
	`POST /index.php/apps/sciencemesh/~marie/api/ocm/Unshare {"Spec":{"Id":{"opaque_id":"some-share-id"}}}`:  {200, ``, serverStateHome},
	`POST /index.php/apps/sciencemesh/~marie/api/ocm/UpdateShare {"ref":{"Spec":{"Id":{"opaque_id":"some-share-id"}}},"p":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}}}`: {200, `{"id":{},"resource_id":{},"permissions":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890}}`, serverStateHome},
	`POST /index.php/apps/sciencemesh/~marie/api/ocm/ListShares [{"type":4,"Term":{"Creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}}]`: {200, `[{"id":{},"resource_id":{},"permissions":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890}}]`, serverStateHome},
	`POST /index.php/apps/sciencemesh/~marie/api/ocm/ListReceivedShares `:                                            {200, `[{"share":{"id":{},"resource_id":{},"permissions":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890}},"state":2}]`, serverStateHome},
	`POST /index.php/apps/sciencemesh/~marie/api/ocm/GetReceivedShare {"Spec":{"Id":{"opaque_id":"some-share-id"}}}`: {200, `{"share":{"id":{},"resource_id":{},"permissions":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890}},"state":2}`, serverStateHome},
	`POST /index.php/apps/sciencemesh/~marie/api/ocm/UpdateReceivedShare {"received_share":{"share":{"id":{},"permissions":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890}},"state":2},"field_mask":{"paths":["state"]}}`: {200, `{"share":{"id":{},"permissions":{"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}},"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"creator":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"ctime":{"seconds":1234567890},"mtime":{"seconds":1234567890}},"state":2}`, serverStateHome},
}

// GetNextcloudServerMock returns a handler that pretends to be a remote Nextcloud server.
func GetNextcloudServerMock(called *[]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(strings.Builder)
		_, err := io.Copy(buf, r.Body)
		if err != nil {
			panic("Error reading response into buffer")
		}
		var key = fmt.Sprintf("%s %s %s", r.Method, r.URL, buf.String())
		*called = append(*called, key)
		response := responses[key]
		if (response == Response{}) {
			key = fmt.Sprintf("%s %s %s %s", r.Method, r.URL, buf.String(), serverState)
			response = responses[key]
		}
		if (response == Response{}) {
			fmt.Printf("%s %s %s %s\n", r.Method, r.URL, buf.String(), serverState)
			response = Response{500, fmt.Sprintf("{\"response not defined\": \"%s\"}", key), serverStateEmpty}
		}
		serverState = responses[key].newServerState
		if serverState == `` {
			serverState = serverStateError
		}
		w.WriteHeader(response.code)
		// w.Header().Set("Etag", "mocker-etag")
		_, err = w.Write([]byte(responses[key].body))
		if err != nil {
			panic(err)
		}
	})
}

// TestingHTTPClient thanks to https://itnext.io/how-to-stub-requests-to-remote-hosts-with-go-6c2c1db32bf2
// Ideally, this function would live in tests/helpers, but
// if we put it there, it gets excluded by .dockerignore, and the
// Docker build fails (see https://github.com/cs3org/reva/issues/1999)
// So putting it here for now - open to suggestions if someone knows
// a better way to inject this.
func TestingHTTPClient(handler http.Handler) (*http.Client, func()) {
	s := httptest.NewServer(handler)

	cli := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, s.Listener.Addr().String())
			},
		},
	}

	return cli, s.Close
}
