// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package authguest

import (
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/jwt"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

type Option func(*Options)

// Options for the auth-guest service
type Options struct {
	GatewaySelector     pool.Selectable[gateway.GatewayAPIClient]
	ServiceAccount      config.ServiceAccount
	JWT                 *jwt.JwtService
	Publisher           events.Publisher
	PinAndNonceTTL      time.Duration
	BackoffLimiter      config.BackoffLimiter
	QuotaLimiter        config.QuotaLimiter
	SessionQuotaLimiter config.QuotaLimiter
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

// EventsPublisher configures the event publisher for the auth-guest service
func EventsPublisher(p events.Publisher) Option {
	return func(o *Options) {
		o.Publisher = p
	}
}

// PinAndNonceTTL configures the shared PIN and nonce lifetime for the auth-guest service
func PinAndNonceTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.PinAndNonceTTL = ttl
	}
}

// BackoffLimiter configures the failed-attempt backoff limiter for the auth-guest service
func BackoffLimiter(l config.BackoffLimiter) Option {
	return func(o *Options) {
		o.BackoffLimiter = l
	}
}

// QuotaLimiter configures the quota limiter for renewals without a valid session
func QuotaLimiter(l config.QuotaLimiter) Option {
	return func(o *Options) {
		o.QuotaLimiter = l
	}
}

// SessionQuotaLimiter configures the quota limiter for renewals authorized by a valid session
func SessionQuotaLimiter(l config.QuotaLimiter) Option {
	return func(o *Options) {
		o.SessionQuotaLimiter = l
	}
}
