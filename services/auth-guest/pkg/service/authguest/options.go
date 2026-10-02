// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package authguest

import (
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/jwt"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

type Option func(*Options)

// Options for the auth-guest service
type Options struct {
	GatewaySelector pool.Selectable[gateway.GatewayAPIClient]
	ServiceAccount  config.ServiceAccount
	JWT             *jwt.JwtService
}

// GatewaySelector adds a grpc client selector for the gateway service
func GatewaySelector(gatewaySelector pool.Selectable[gateway.GatewayAPIClient]) Option {
	return func(o *Options) {
		o.GatewaySelector = gatewaySelector
	}
}

// ServiceAccount configures a service account for the auth-guest service
func ServiceAccount(sa config.ServiceAccount) Option {
	return func(o *Options) {
		o.ServiceAccount = sa
	}
}

// JWT configures the jwt service for the auth-guest service
func JWT(m *jwt.JwtService) Option {
	return func(o *Options) {
		o.JWT = m
	}
}
