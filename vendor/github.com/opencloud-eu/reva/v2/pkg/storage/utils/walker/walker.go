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

package walker

import (
	"context"
	"path/filepath"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

// WalkFunc is the type of function called by Walk to visit each file or directory
//
// Each time the Walk function meet a file/folder path is set to the full path of this.
// The err argument reports an error related to the path, and the function can decide the action to
// do with this.
//
// The error result returned by the function controls how Walk continues. If the function returns the special value SkipDir, Walk skips the current directory.
// Otherwise, if the function returns a non-nil error, Walk stops entirely and returns that error.
type WalkFunc func(wd string, info *provider.ResourceInfo, err error) error

// Walker is an interface implemented by objects that are able to walk from a dir rooted into the passed path
type Walker interface {
	// Walk walks the file tree rooted at root, calling fn for each file or folder in the tree, including the root.
	Walk(ctx context.Context, root *provider.ResourceId, fn WalkFunc) error
}

type revaWalker struct {
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
}

// NewWalker creates a Walker object that uses the reva gateway
func NewWalker(gatewaySelector pool.Selectable[gateway.GatewayAPIClient]) Walker {
	return &revaWalker{gatewaySelector: gatewaySelector}
}

// Walk walks the file tree rooted at root, calling fn for each file or folder in the tree, including the root.
func (r *revaWalker) Walk(ctx context.Context, root *provider.ResourceId, fn WalkFunc) error {
	info, err := r.stat(ctx, root)

	if err != nil {
		return fn("", nil, err)
	}

	err = r.walkRecursively(ctx, "", info, fn)

	if err == filepath.SkipDir {
		return nil
	}

	return err
}

func (r *revaWalker) walkRecursively(ctx context.Context, wd string, info *provider.ResourceInfo, fn WalkFunc) error {

	if info.Type != provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		return fn(wd, info, nil)
	}

	err := fn(wd, info, nil)
	if err != nil {
		return err
	}

	list, err := r.readDir(ctx, info.Id)
	if err != nil {
		return err
	}
	for _, file := range list {
		err = r.walkRecursively(ctx, filepath.Join(wd, info.Path), file, fn)
		if err != nil && (file.Type != provider.ResourceType_RESOURCE_TYPE_CONTAINER || err != filepath.SkipDir) {
			return err
		}
	}

	return nil
}

func (r *revaWalker) readDir(ctx context.Context, id *provider.ResourceId) ([]*provider.ResourceInfo, error) {
	gatewayClient, err := r.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	resp, err := gatewayClient.ListContainer(ctx, &provider.ListContainerRequest{Ref: &provider.Reference{ResourceId: id, Path: "."}})

	switch {
	case err != nil:
		return nil, err
	case resp.Status.Code != rpc.Code_CODE_OK:
		return nil, errtypes.NewErrtypeFromStatus(resp.Status)
	}

	return resp.Infos, nil
}

func (r *revaWalker) stat(ctx context.Context, id *provider.ResourceId) (*provider.ResourceInfo, error) {
	gatewayClient, err := r.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	resp, err := gatewayClient.Stat(ctx, &provider.StatRequest{Ref: &provider.Reference{ResourceId: id, Path: "."}})

	switch {
	case err != nil:
		return nil, err
	case resp.Status.Code != rpc.Code_CODE_OK:
		return nil, errtypes.NewErrtypeFromStatus(resp.Status)
	}

	return resp.Info, nil
}
