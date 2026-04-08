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

package loader

import (
	// Load core authentication managers.
	_ "github.com/opencloud-eu/reva/v2/pkg/auth/manager/appauth"
	_ "github.com/opencloud-eu/reva/v2/pkg/auth/manager/demo"
	_ "github.com/opencloud-eu/reva/v2/pkg/auth/manager/impersonator"
	_ "github.com/opencloud-eu/reva/v2/pkg/auth/manager/json"
	_ "github.com/opencloud-eu/reva/v2/pkg/auth/manager/ldap"
	_ "github.com/opencloud-eu/reva/v2/pkg/auth/manager/machine"
	_ "github.com/opencloud-eu/reva/v2/pkg/auth/manager/ocmshares"
	_ "github.com/opencloud-eu/reva/v2/pkg/auth/manager/oidc"
	_ "github.com/opencloud-eu/reva/v2/pkg/auth/manager/publicshares"
	_ "github.com/opencloud-eu/reva/v2/pkg/auth/manager/serviceaccounts"
	// Add your own here
)
