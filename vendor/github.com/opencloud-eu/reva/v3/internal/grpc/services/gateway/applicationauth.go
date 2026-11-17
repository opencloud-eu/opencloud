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

	appauthpb "github.com/cs3org/go-cs3apis/cs3/auth/applications/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/pkg/errors"
)

func (s *svc) GenerateAppPassword(ctx context.Context, req *appauthpb.GenerateAppPasswordRequest) (*appauthpb.GenerateAppPasswordResponse, error) {
	c, err := pool.GetAppAuthProviderServiceClient(s.c.ApplicationAuthEndpoint)
	if err != nil {
		return &appauthpb.GenerateAppPasswordResponse{
			Status: status.NewInternal(ctx, "error getting app auth provider client"),
		}, nil
	}

	res, err := c.GenerateAppPassword(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GenerateAppPassword")
	}

	return res, nil
}

func (s *svc) ListAppPasswords(ctx context.Context, req *appauthpb.ListAppPasswordsRequest) (*appauthpb.ListAppPasswordsResponse, error) {
	c, err := pool.GetAppAuthProviderServiceClient(s.c.ApplicationAuthEndpoint)
	if err != nil {
		return &appauthpb.ListAppPasswordsResponse{
			Status: status.NewInternal(ctx, "error getting app auth provider client"),
		}, nil
	}

	res, err := c.ListAppPasswords(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling ListAppPasswords")
	}

	return res, nil
}

func (s *svc) InvalidateAppPassword(ctx context.Context, req *appauthpb.InvalidateAppPasswordRequest) (*appauthpb.InvalidateAppPasswordResponse, error) {
	c, err := pool.GetAppAuthProviderServiceClient(s.c.ApplicationAuthEndpoint)
	if err != nil {
		return &appauthpb.InvalidateAppPasswordResponse{
			Status: status.NewInternal(ctx, "error getting app auth provider client"),
		}, nil
	}

	res, err := c.InvalidateAppPassword(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling InvalidateAppPassword")
	}

	return res, nil
}

func (s *svc) GetAppPassword(ctx context.Context, req *appauthpb.GetAppPasswordRequest) (*appauthpb.GetAppPasswordResponse, error) {
	c, err := pool.GetAppAuthProviderServiceClient(s.c.ApplicationAuthEndpoint)
	if err != nil {
		return &appauthpb.GetAppPasswordResponse{
			Status: status.NewInternal(ctx, "error getting app auth provider client"),
		}, nil
	}

	res, err := c.GetAppPassword(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GetAppPassword")
	}

	return res, nil
}
