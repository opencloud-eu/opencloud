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

package sharedconf

import (
	"fmt"
	"os"
	"sync"

	"github.com/mitchellh/mapstructure"
)

var (
	sharedConf     = &conf{}
	sharedConfOnce sync.Once
)

// ClientOptions represent additional options (e.g. tls settings) for the grpc clients
type ClientOptions struct {
	TLSMode    string `mapstructure:"tls_mode"`
	CACertFile string `mapstructure:"cacert"`
}

type conf struct {
	JWTSecret             string        `mapstructure:"jwt_secret"`
	GatewaySVC            string        `mapstructure:"gatewaysvc"`
	DataGateway           string        `mapstructure:"datagateway"`
	SkipUserGroupsInToken bool          `mapstructure:"skip_user_groups_in_token"`
	MultiTenantEnabled    bool          `mapstructure:"multi_tenant_enabled"`
	GRPCClientOptions     ClientOptions `mapstructure:"grpc_client_options"`
}

// Decode decodes the configuration.
func Decode(v interface{}) error {
	var err error

	sharedConfOnce.Do(func() {
		if err = mapstructure.Decode(v, sharedConf); err != nil {
			return
		}
		// add some defaults
		if sharedConf.GatewaySVC == "" {
			sharedConf.GatewaySVC = "0.0.0.0:19000"
		}

		// this is the default address we use for the data gateway HTTP service
		if sharedConf.DataGateway == "" {
			host, err := os.Hostname()
			if err != nil || host == "" {
				sharedConf.DataGateway = "http://0.0.0.0:19001/datagateway"
			} else {
				sharedConf.DataGateway = fmt.Sprintf("http://%s:19001/datagateway", host)
			}
		}

		// TODO(labkode): would be cool to autogenerate one secret and print
		// it on init time.
		if sharedConf.JWTSecret == "" {
			sharedConf.JWTSecret = "changemeplease"
		}
	})

	return err
}

// GetJWTSecret returns the package level configured jwt secret if not overwritten.
func GetJWTSecret(val string) string {
	if val == "" {
		return sharedConf.JWTSecret
	}
	return val
}

// GetGatewaySVC returns the package level configured gateway service if not overwritten.
func GetGatewaySVC(val string) string {
	if val == "" {
		return sharedConf.GatewaySVC
	}
	return val
}

// GetDataGateway returns the package level data gateway endpoint if not overwritten.
func GetDataGateway(val string) string {
	if val == "" {
		return sharedConf.DataGateway
	}
	return val
}

// SkipUserGroupsInToken returns whether to skip encoding user groups in the access tokens.
func SkipUserGroupsInToken() bool {
	return sharedConf.SkipUserGroupsInToken
}

// MultiTenantEnabled returns whether this is a mulit-tenant enabled configuratio
func MultiTenantEnabled() bool {
	return sharedConf.MultiTenantEnabled
}

// GRPCClientOptions returns the global grpc client options
func GRPCClientOptions() ClientOptions {
	return sharedConf.GRPCClientOptions
}

// this is used by the tests
func resetOnce() {
	sharedConf = &conf{}
	sharedConfOnce = sync.Once{}
}
