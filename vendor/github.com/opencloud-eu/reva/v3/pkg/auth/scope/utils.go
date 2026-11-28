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
	"fmt"
	"strings"

	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
)

// FormatScope create a pretty print of the scope
func FormatScope(scopeType string, scope *authpb.Scope) (string, error) {
	// TODO(gmgigi96): check decoder type
	switch {
	case strings.HasPrefix(scopeType, "user"):
		// user scope
		var ref provider.Reference
		err := utils.UnmarshalJSONToProtoV1(scope.Resource.Value, &ref)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s %s", ref.String(), scope.Role.String()), nil
	case strings.HasPrefix(scopeType, "publicshare"):
		// public share
		var pShare link.PublicShare
		err := utils.UnmarshalJSONToProtoV1(scope.Resource.Value, &pShare)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("share:\"%s\" %s", pShare.Id.OpaqueId, scope.Role.String()), nil
	case strings.HasPrefix(scopeType, "resourceinfo"):
		var resInfo provider.ResourceInfo
		err := utils.UnmarshalJSONToProtoV1(scope.Resource.Value, &resInfo)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("path:\"%s\" %s", resInfo.Path, scope.Role.String()), nil
	default:
		return "", errtypes.NotSupported("scope not yet supported")
	}
}
