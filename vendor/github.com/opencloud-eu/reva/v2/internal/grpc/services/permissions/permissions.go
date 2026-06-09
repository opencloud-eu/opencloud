// Copyright 2021 CERN
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

package permissions

import (
	"context"
	"fmt"

	permissions "github.com/cs3org/go-cs3apis/cs3/permissions/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/reva/v2/pkg/permission"
	"github.com/opencloud-eu/reva/v2/pkg/permission/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

func init() {
	rgrpc.Register("permissions", New)
}

type config struct {
	Driver  string                            `mapstructure:"driver" docs:"localhome;The permission driver to be used."`
	Drivers map[string]map[string]interface{} `mapstructure:"drivers" docs:"url:pkg/permission/permission.go"`
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	return c, nil
}

type service struct {
	manager permission.Manager
}

// New returns a new PermissionsServiceServer
func New(m map[string]interface{}, ss *grpc.Server, _ *zerolog.Logger) (rgrpc.Service, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}

	f, ok := registry.NewFuncs[c.Driver]
	if !ok {
		return nil, fmt.Errorf("could not get permission manager '%s'", c.Driver)
	}
	manager, err := f(c.Drivers[c.Driver])
	if err != nil {
		return nil, err
	}

	service := &service{manager: manager}
	return service, nil
}

func (s *service) Close() error {
	return nil
}

func (s *service) UnprotectedEndpoints() []string {
	return []string{}
}

func (s *service) Register(ss *grpc.Server) {
	permissions.RegisterPermissionsAPIServer(ss, s)
}

func (s *service) CheckPermission(ctx context.Context, req *permissions.CheckPermissionRequest) (*permissions.CheckPermissionResponse, error) {
	var subject string
	switch ref := req.SubjectRef.Spec.(type) {
	case *permissions.SubjectReference_UserId:
		subject = ref.UserId.OpaqueId
	case *permissions.SubjectReference_GroupId:
		subject = ref.GroupId.OpaqueId
	}
	var status *rpc.Status
	if ok := s.manager.CheckPermission(req.Permission, subject, req.Ref); ok {
		status = &rpc.Status{Code: rpc.Code_CODE_OK}
	} else {
		status = &rpc.Status{Code: rpc.Code_CODE_PERMISSION_DENIED}
	}
	return &permissions.CheckPermissionResponse{Status: status}, nil
}
