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

package helloworld

import (
	"context"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/reva/v2/internal/grpc/services/helloworld/proto"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

func init() {
	rgrpc.Register("helloworld", New)
}

type conf struct {
	Message string `mapstructure:"message"`
}
type service struct {
	conf *conf
}

// New returns a new PreferencesServiceServer
// It can be tested like this:
// prototool grpc --address 0.0.0.0:9999 --method 'revad.helloworld.HelloWorldService/Hello' --data '{"name": "Alice"}'
func New(m map[string]interface{}, ss *grpc.Server, _ *zerolog.Logger) (rgrpc.Service, error) {
	c := &conf{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "helloworld: error decoding conf")
		return nil, err
	}

	if c.Message == "" {
		c.Message = "Hello"
	}
	service := &service{conf: c}
	return service, nil
}

func (s *service) Close() error {
	return nil
}

func (s *service) UnprotectedEndpoints() []string {
	return []string{}
}

func (s *service) Register(ss *grpc.Server) {
	proto.RegisterHelloWorldServiceServer(ss, s)
}

func (s *service) Hello(ctx context.Context, req *proto.HelloRequest) (*proto.HelloResponse, error) {
	if req.Name == "" {
		req.Name = "Mr. Nobody"
	}
	message := fmt.Sprintf("%s %s", s.conf.Message, req.Name)
	res := &proto.HelloResponse{
		Message: message,
	}
	return res, nil
}
