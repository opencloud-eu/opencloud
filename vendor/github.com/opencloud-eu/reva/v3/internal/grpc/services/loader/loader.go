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
	// Load core gRPC services.
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/applicationauth"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/appprovider"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/appregistry"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/authprovider"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/authregistry"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/datatx"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/gateway"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/groupprovider"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/helloworld"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/ocmcore"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/ocminvitemanager"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/ocmproviderauthorizer"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/ocmshareprovider"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/permissions"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/preferences"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/publicshareprovider"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/publicstorageprovider"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/sharesstorageprovider"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/storageprovider"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/storageregistry"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/userprovider"
	_ "github.com/opencloud-eu/reva/v3/internal/grpc/services/usershareprovider"
	// Add your own service here
)
