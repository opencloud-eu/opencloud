package service

import (
	"github.com/opencloud-eu/opencloud/pkg/log"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	"go-micro.dev/v4/store"
	"go.opentelemetry.io/otel/trace"
)

// Option for the userlog service
type Option func(*Options)

// Options for the userlog service
type Options struct {
	Logger        log.Logger
	Store         store.Store
	HistoryClient ehsvc.EventHistoryService
	TraceProvider trace.TracerProvider
}

// Logger configures a logger for the userlog service
func Logger(log log.Logger) Option {
	return func(o *Options) {
		o.Logger = log
	}
}

// Store defines the store for the userlog service
func Store(s store.Store) Option {
	return func(o *Options) {
		o.Store = s
	}
}

// HistoryClient adds a grpc client for the eventhistory service
func HistoryClient(hc ehsvc.EventHistoryService) Option {
	return func(o *Options) {
		o.HistoryClient = hc
	}
}

// TraceProvider adds a tracer provider for the userlog service
func TraceProvider(tp trace.TracerProvider) Option {
	return func(o *Options) {
		o.TraceProvider = tp
	}
}
