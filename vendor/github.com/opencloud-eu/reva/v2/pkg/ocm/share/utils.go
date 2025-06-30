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

package share

import (
	appprovider "github.com/cs3org/go-cs3apis/cs3/app/provider/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
)

// NewWebDAVProtocol is an abstraction for creating a WebDAV protocol.
func NewWebDAVProtocol(uri, shareSecred string, perms *ocm.SharePermissions) *ocm.Protocol {
	return &ocm.Protocol{
		Term: &ocm.Protocol_WebdavOptions{
			WebdavOptions: &ocm.WebDAVProtocol{
				Uri:          uri,
				SharedSecret: shareSecred,
				Permissions:  perms,
			},
		},
	}
}

// NewWebappProtocol is an abstraction for creating a Webapp protocol.
func NewWebappProtocol(uriTemplate string, viewMode appprovider.ViewMode) *ocm.Protocol {
	return &ocm.Protocol{
		Term: &ocm.Protocol_WebappOptions{
			WebappOptions: &ocm.WebappProtocol{
				Uri:      uriTemplate,
				ViewMode: viewMode,
			},
		},
	}
}

// NewTransferProtocol is an abstraction for creating a Transfer protocol.
func NewTransferProtocol(sourceURI, sharedSecret string, size uint64) *ocm.Protocol {
	return &ocm.Protocol{
		Term: &ocm.Protocol_TransferOptions{
			TransferOptions: &ocm.TransferProtocol{
				SourceUri:    sourceURI,
				SharedSecret: sharedSecret,
				Size:         size,
			},
		},
	}
}

// NewWebDavAccessMethod is an abstraction for creating a WebDAV access method.
func NewWebDavAccessMethod(perms *provider.ResourcePermissions) *ocm.AccessMethod {
	return &ocm.AccessMethod{
		Term: &ocm.AccessMethod_WebdavOptions{
			WebdavOptions: &ocm.WebDAVAccessMethod{
				Permissions: perms,
			},
		},
	}
}

// NewWebappAccessMethod is an abstraction for creating a Webapp access method.
func NewWebappAccessMethod(mode appprovider.ViewMode) *ocm.AccessMethod {
	return &ocm.AccessMethod{
		Term: &ocm.AccessMethod_WebappOptions{
			WebappOptions: &ocm.WebappAccessMethod{
				ViewMode: mode,
			},
		},
	}
}

// NewTransferAccessMethod is an abstraction for creating a Transfer access method.
func NewTransferAccessMethod() *ocm.AccessMethod {
	return &ocm.AccessMethod{
		Term: &ocm.AccessMethod_TransferOptions{
			TransferOptions: &ocm.TransferAccessMethod{},
		},
	}
}
