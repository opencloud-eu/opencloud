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

package user

import (
	"fmt"
	"net/http"

	"github.com/opencloud-eu/reva/v3/internal/http/services/owncloud/ocs/response"
	"github.com/opencloud-eu/reva/v3/pkg/conversions"
	ctxpkg "github.com/opencloud-eu/reva/v3/pkg/ctx"
)

// The Handler renders the user endpoint
type Handler struct {
}

// GetSelf handles GET requests on /cloud/user
func (h *Handler) GetSelf(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// TODO move user to handler parameter?
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "missing user in context", fmt.Errorf("missing user in context"))
		return
	}

	response.WriteOCSSuccess(w, r, &User{
		ID:          u.Username,
		DisplayName: u.DisplayName,
		Email:       u.Mail,
		UserType:    conversions.UserTypeString(u.Id.Type),
	})
}

// User holds user data
type User struct {
	ID          string `json:"id" xml:"id"`                     // UserID in ocs is the username
	DisplayName string `json:"display-name" xml:"display-name"` // is used in ocs/v(1|2).php/cloud/user - yes this is different from the users endpoint
	Email       string `json:"email" xml:"email"`
	UserType    string `json:"user-type" xml:"user-type"`
}
