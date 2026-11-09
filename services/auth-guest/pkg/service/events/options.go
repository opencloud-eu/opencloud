// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest"
	"github.com/opencloud-eu/reva/v2/pkg/events"
)

// Option for the auth-guest service
type Option func(*Options)

// Options for the auth-guest service
type Options struct {
	Context          context.Context
	Logger           log.Logger
	Stream           events.Stream
	RegisteredEvents []events.Unmarshaller
	NumConsumers     int
	AuthGuestService authguest.AuthGuest
}

// Context configures a context for the auth-guest service
func Context(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

// Logger configures a logger for the auth-guest service
func Logger(log log.Logger) Option {
	return func(o *Options) {
		o.Logger = log
	}
}

// Stream configures an event stream for the auth-guest service
func Stream(s events.Stream) Option {
	return func(o *Options) {
		o.Stream = s
	}
}

// RegisteredEvents registers the events the service should listen to
func RegisteredEvents(e []events.Unmarshaller) Option {
	return func(o *Options) {
		o.RegisteredEvents = e
	}
}

// NumConsumers configures the amount of concurrent event consumers
func NumConsumers(num int) Option {
	return func(o *Options) {
		o.NumConsumers = num
	}
}

// AuthGuestService configures the guest auth domain service.
func AuthGuestService(s authguest.AuthGuest) Option {
	return func(o *Options) {
		o.AuthGuestService = s
	}
}
