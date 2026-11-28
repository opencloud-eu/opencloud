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

	preferences "github.com/cs3org/go-cs3apis/cs3/preferences/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/pkg/errors"
)

func (s *svc) SetKey(ctx context.Context, req *preferences.SetKeyRequest) (*preferences.SetKeyResponse, error) {
	c, err := pool.GetPreferencesClient(s.c.PreferencesEndpoint)
	if err != nil {
		return &preferences.SetKeyResponse{
			Status: status.NewInternal(ctx, "error getting preferences client"),
		}, nil
	}

	res, err := c.SetKey(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling SetKey")
	}

	return res, nil
}

func (s *svc) GetKey(ctx context.Context, req *preferences.GetKeyRequest) (*preferences.GetKeyResponse, error) {
	c, err := pool.GetPreferencesClient(s.c.PreferencesEndpoint)
	if err != nil {
		return &preferences.GetKeyResponse{
			Status: status.NewInternal(ctx, "error getting preferences client"),
		}, nil
	}

	res, err := c.GetKey(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GetKey")
	}

	return res, nil
}
