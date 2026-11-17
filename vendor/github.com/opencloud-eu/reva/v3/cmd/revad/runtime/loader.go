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

package runtime

import (
	// These are all the extensions points for REVA
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/interceptors/loader"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/loader"
	_ "github.com/opencloud-eu/reva/v3/internal/http/interceptors/auth/credential/loader"
	_ "github.com/opencloud-eu/reva/v3/internal/http/interceptors/auth/token/loader"
	_ "github.com/opencloud-eu/reva/v3/internal/http/interceptors/auth/tokenwriter/loader"
	_ "github.com/opencloud-eu/reva/v3/internal/http/interceptors/loader"
	_ "github.com/opencloud-eu/reva/v3/internal/http/services/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/app/provider/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/app/registry/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/appauth/manager/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/auth/manager/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/auth/registry/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/datatx/manager/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/group/manager/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/metrics/driver/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/ocm/invite/repository/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/ocm/provider/authorizer/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/ocm/share/repository/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/permission/manager/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/preferences/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/publicshare/manager/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/rhttp/datatx/manager/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/share/cache/warmup/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/share/manager/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/storage/fs/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/storage/registry/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/tenant/manager/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/token/manager/loader"
	_ "github.com/opencloud-eu/reva/v3/pkg/user/manager/loader"
)
