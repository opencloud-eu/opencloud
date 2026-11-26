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

package auth

import (
	"context"
	"net/rpc"

	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	hcplugin "github.com/hashicorp/go-plugin"
	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"github.com/opencloud-eu/reva/v3/pkg/plugin"
)

func init() {
	plugin.Register("authprovider", &ProviderPlugin{})
}

// ProviderPlugin is the implementation of plugin.Plugin so we can serve/consume this.
type ProviderPlugin struct {
	Impl Manager
}

// Server returns the RPC Server which serves the methods that the Client calls over net/rpc
func (p *ProviderPlugin) Server(*hcplugin.MuxBroker) (interface{}, error) {
	return &RPCServer{Impl: p.Impl}, nil
}

// Client returns interface implementation for the plugin that communicates to the server end of the plugin
func (p *ProviderPlugin) Client(b *hcplugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &RPCClient{Client: c}, nil
}

// RPCClient is an implementation of Manager that talks over RPC.
type RPCClient struct{ Client *rpc.Client }

// ConfigureArg for RPC
type ConfigureArg struct {
	Ml map[string]interface{}
}

// ConfigureReply for RPC
type ConfigureReply struct {
	Err error
}

// Configure RPCClient configure method
func (m *RPCClient) Configure(ml map[string]interface{}) error {
	args := ConfigureArg{Ml: ml}
	resp := ConfigureReply{}
	err := m.Client.Call("Plugin.Configure", args, &resp)
	if err != nil {
		return err
	}
	return resp.Err
}

// AuthenticateArgs for RPC
type AuthenticateArgs struct {
	Ctx          map[interface{}]interface{}
	ClientID     string
	ClientSecret string
}

// AuthenticateReply for RPC
type AuthenticateReply struct {
	User  *user.User
	Auth  map[string]*authpb.Scope
	Error error
}

// Authenticate RPCClient Authenticate method
func (m *RPCClient) Authenticate(ctx context.Context, clientID, clientSecret string) (*user.User, map[string]*authpb.Scope, error) {
	ctxVal := appctx.GetKeyValuesFromCtx(ctx)
	args := AuthenticateArgs{Ctx: ctxVal, ClientID: clientID, ClientSecret: clientSecret}
	reply := AuthenticateReply{}
	err := m.Client.Call("Plugin.Authenticate", args, &reply)
	if err != nil {
		return nil, nil, err
	}
	return reply.User, reply.Auth, reply.Error
}

// RPCServer is the server that RPCClient talks to, conforming to the requirements of net/rpc
type RPCServer struct {
	// This is the real implementation
	Impl Manager
}

// Configure RPCServer Configure method
func (m *RPCServer) Configure(args ConfigureArg, resp *ConfigureReply) error {
	resp.Err = m.Impl.Configure(args.Ml)
	return nil
}

// Authenticate RPCServer Authenticate method
func (m *RPCServer) Authenticate(args AuthenticateArgs, resp *AuthenticateReply) error {
	ctx := appctx.PutKeyValuesToCtx(args.Ctx)
	resp.User, resp.Auth, resp.Error = m.Impl.Authenticate(ctx, args.ClientID, args.ClientSecret)
	return nil
}
