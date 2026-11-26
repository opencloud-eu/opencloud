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

package pool

import (
	appprovider "github.com/cs3org/go-cs3apis/cs3/app/provider/v1beta1"
	appregistry "github.com/cs3org/go-cs3apis/cs3/app/registry/v1beta1"
	applicationauth "github.com/cs3org/go-cs3apis/cs3/auth/applications/v1beta1"
	authprovider "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	authregistry "github.com/cs3org/go-cs3apis/cs3/auth/registry/v1beta1"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	group "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	tenant "github.com/cs3org/go-cs3apis/cs3/identity/tenant/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	ocmcore "github.com/cs3org/go-cs3apis/cs3/ocm/core/v1beta1"
	invitepb "github.com/cs3org/go-cs3apis/cs3/ocm/invite/v1beta1"
	ocmprovider "github.com/cs3org/go-cs3apis/cs3/ocm/provider/v1beta1"
	permissions "github.com/cs3org/go-cs3apis/cs3/permissions/v1beta1"
	preferences "github.com/cs3org/go-cs3apis/cs3/preferences/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	storageregistry "github.com/cs3org/go-cs3apis/cs3/storage/registry/v1beta1"
	datatx "github.com/cs3org/go-cs3apis/cs3/tx/v1beta1"
)

// GetGatewayServiceClient returns a GatewayServiceClient.
func GetGatewayServiceClient(id string, opts ...Option) (gateway.GatewayAPIClient, error) {
	selector, _ := GatewaySelector(id, opts...)
	return selector.Next()
}

// GetUserProviderServiceClient returns a UserProviderServiceClient.
func GetUserProviderServiceClient(id string, opts ...Option) (user.UserAPIClient, error) {
	selector, _ := IdentityUserSelector(id, opts...)
	return selector.Next()
}

// GetTenantProviderServiceClient returns a TenantProviderServiceClient.
func GetTenantProviderServiceClient(id string, opts ...Option) (tenant.TenantAPIClient, error) {
	selector, _ := IdentityTenantSelector(id, opts...)
	return selector.Next()
}

// GetGroupProviderServiceClient returns a GroupProviderServiceClient.
func GetGroupProviderServiceClient(id string, opts ...Option) (group.GroupAPIClient, error) {
	selector, _ := IdentityGroupSelector(id, opts...)
	return selector.Next()
}

// GetStorageProviderServiceClient returns a StorageProviderServiceClient.
func GetStorageProviderServiceClient(id string, opts ...Option) (storageprovider.ProviderAPIClient, error) {
	selector, _ := StorageProviderSelector(id, opts...)
	return selector.Next()
}

// GetSpacesProviderServiceClient returns a SpacesProviderServiceClient.
func GetSpacesProviderServiceClient(id string, opts ...Option) (storageprovider.SpacesAPIClient, error) {
	selector, _ := SpacesProviderSelector(id, opts...)
	return selector.Next()
}

// GetAuthRegistryServiceClient returns a new AuthRegistryServiceClient.
func GetAuthRegistryServiceClient(id string, opts ...Option) (authregistry.RegistryAPIClient, error) {
	selector, _ := AuthRegistrySelector(id, opts...)
	return selector.Next()
}

// GetAuthProviderServiceClient returns a new AuthProviderServiceClient.
func GetAuthProviderServiceClient(id string, opts ...Option) (authprovider.ProviderAPIClient, error) {
	selector, _ := AuthProviderSelector(id, opts...)
	return selector.Next()
}

// GetAppAuthProviderServiceClient returns a new AppAuthProviderServiceClient.
func GetAppAuthProviderServiceClient(id string, opts ...Option) (applicationauth.ApplicationsAPIClient, error) {
	selector, _ := AuthApplicationSelector(id, opts...)
	return selector.Next()
}

// GetUserShareProviderClient returns a new UserShareProviderClient.
func GetUserShareProviderClient(id string, opts ...Option) (collaboration.CollaborationAPIClient, error) {
	selector, _ := SharingCollaborationSelector(id, opts...)
	return selector.Next()
}

// GetOCMShareProviderClient returns a new OCMShareProviderClient.
func GetOCMShareProviderClient(id string, opts ...Option) (ocm.OcmAPIClient, error) {
	selector, _ := SharingOCMSelector(id, opts...)
	return selector.Next()
}

// GetOCMInviteManagerClient returns a new OCMInviteManagerClient.
func GetOCMInviteManagerClient(id string, opts ...Option) (invitepb.InviteAPIClient, error) {
	selector, _ := OCMInviteSelector(id, opts...)
	return selector.Next()
}

// GetPublicShareProviderClient returns a new PublicShareProviderClient.
func GetPublicShareProviderClient(id string, opts ...Option) (link.LinkAPIClient, error) {
	selector, _ := SharingLinkSelector(id, opts...)
	return selector.Next()
}

// GetPreferencesClient returns a new PreferencesClient.
func GetPreferencesClient(id string, opts ...Option) (preferences.PreferencesAPIClient, error) {
	selector, _ := PreferencesSelector(id, opts...)
	return selector.Next()
}

// GetPermissionsClient returns a new PermissionsClient.
func GetPermissionsClient(id string, opts ...Option) (permissions.PermissionsAPIClient, error) {
	selector, _ := PermissionsSelector(id, opts...)
	return selector.Next()
}

// GetAppRegistryClient returns a new AppRegistryClient.
func GetAppRegistryClient(id string, opts ...Option) (appregistry.RegistryAPIClient, error) {
	selector, _ := AppRegistrySelector(id, opts...)
	return selector.Next()
}

// GetAppProviderClient returns a new AppRegistryClient.
func GetAppProviderClient(id string, opts ...Option) (appprovider.ProviderAPIClient, error) {
	selector, _ := AppProviderSelector(id, opts...)
	return selector.Next()
}

// GetStorageRegistryClient returns a new StorageRegistryClient.
func GetStorageRegistryClient(id string, opts ...Option) (storageregistry.RegistryAPIClient, error) {
	selector, _ := StorageRegistrySelector(id, opts...)
	return selector.Next()
}

// GetOCMProviderAuthorizerClient returns a new OCMProviderAuthorizerClient.
func GetOCMProviderAuthorizerClient(id string, opts ...Option) (ocmprovider.ProviderAPIClient, error) {
	selector, _ := OCMProviderSelector(id, opts...)
	return selector.Next()
}

// GetOCMCoreClient returns a new OCMCoreClient.
func GetOCMCoreClient(id string, opts ...Option) (ocmcore.OcmCoreAPIClient, error) {
	selector, _ := OCMCoreSelector(id, opts...)
	return selector.Next()
}

// GetDataTxClient returns a new DataTxClient.
func GetDataTxClient(id string, opts ...Option) (datatx.TxAPIClient, error) {
	selector, _ := TXSelector(id, opts...)
	return selector.Next()
}
