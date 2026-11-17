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

package gateway

import (
	"context"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	registry "github.com/cs3org/go-cs3apis/cs3/storage/registry/v1beta1"
	ctxpkg "github.com/opencloud-eu/reva/v3/pkg/ctx"
	sdk "github.com/opencloud-eu/reva/v3/pkg/sdk/common"
	"github.com/opencloud-eu/reva/v3/pkg/storage/cache"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

/*
   Cached Registry
*/

type cachedRegistryClient struct {
	c     registry.RegistryAPIClient
	cache cache.ProviderCache
}

func (c *cachedRegistryClient) ListStorageProviders(ctx context.Context, in *registry.ListStorageProvidersRequest, opts ...grpc.CallOption) (*registry.ListStorageProvidersResponse, error) {

	spaceID := sdk.DecodeOpaqueMap(in.Opaque)["space_id"]

	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		return nil, errors.New("user not found in context")
	}

	key := c.cache.GetKey(u.GetId(), spaceID)
	if key != "" {
		s := &registry.ListStorageProvidersResponse{}
		if err := c.cache.PullFromCache(key, s); err == nil {
			return s, nil
		}
	}

	resp, err := c.c.ListStorageProviders(ctx, in, opts...)
	switch {
	case err != nil:
		return nil, err
	case resp.Status.Code != rpc.Code_CODE_OK:
		return resp, nil
	case spaceID == "":
		return resp, nil
	case spaceID == utils.ShareStorageSpaceID: // TODO do we need to compare providerid and spaceid separately?
		return resp, nil
	default:
		return resp, c.cache.PushToCache(key, resp)
	}
}

// not cached

func (c *cachedRegistryClient) GetStorageProviders(ctx context.Context, in *registry.GetStorageProvidersRequest, opts ...grpc.CallOption) (*registry.GetStorageProvidersResponse, error) {
	return c.c.GetStorageProviders(ctx, in, opts...)
}

func (c *cachedRegistryClient) GetHome(ctx context.Context, in *registry.GetHomeRequest, opts ...grpc.CallOption) (*registry.GetHomeResponse, error) {
	return c.c.GetHome(ctx, in, opts...)
}
