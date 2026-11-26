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

package scope

import (
	"context"
	"strings"

	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
	"github.com/rs/zerolog"
)

func lightweightAccountScope(_ context.Context, scope *authpb.Scope, resource interface{}, _ *zerolog.Logger) (bool, error) {
	// Lightweight accounts have access to resources shared with them.
	// These cannot be resolved from here, but need to be added to the scope from
	// where the call to mint tokens is made.
	// From here, we only allow ListReceivedShares calls
	switch v := resource.(type) {
	case *collaboration.ListReceivedSharesRequest:
		return true, nil
	case string:
		return checkLightweightPath(v), nil
	}
	return false, nil
}

func checkLightweightPath(path string) bool {
	paths := []string{
		"/ocs/v2.php/apps/files_sharing/api/v1/shares",
		"/ocs/v1.php/apps/files_sharing/api/v1/shares",
		"/ocs/v2.php/apps/files_sharing//api/v1/shares",
		"/ocs/v1.php/apps/files_sharing//api/v1/shares",
		"/ocs/v2.php/cloud/capabilities",
		"/ocs/v1.php/cloud/capabilities",
		"/ocs/v2.php/cloud/user",
		"/ocs/v1.php/cloud/user",
		"/remote.php/webdav",
		"/remote.php/dav/files",
		"/app/open",
		"/app/new",
		"/archiver",
		"/dataprovider",
		"/data",
	}
	for _, p := range paths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// AddLightweightAccountScope adds the scope to allow access to lightweight user.
func AddLightweightAccountScope(role authpb.Role, scopes map[string]*authpb.Scope) (map[string]*authpb.Scope, error) {
	ref := &provider.Reference{Path: "/"}
	val, err := utils.MarshalProtoV1ToJSON(ref)
	if err != nil {
		return nil, err
	}
	if scopes == nil {
		scopes = make(map[string]*authpb.Scope)
	}
	scopes["lightweight"] = &authpb.Scope{
		Resource: &types.OpaqueEntry{
			Decoder: "json",
			Value:   val,
		},
		Role: role,
	}
	return scopes, nil
}
