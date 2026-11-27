package activitylog

import (
	"time"

	"github.com/nats-io/nats.go"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// Option for the activitylog service
type Option func(*Options)

// Options for the activitylog service
type Options struct {
	Logger              log.Logger
	MaxActivities       int
	WriteBufferDuration time.Duration
	JetStream           nats.JetStreamContext
}

// Logger configures a logger for the activitylog service
func Logger(log log.Logger) Option {
	return func(o *Options) {
		o.Logger = log
	}
}

func MaxActivities(max int) Option {
	return func(o *Options) {
		o.MaxActivities = max
	}
}
func WriteBufferDuration(d time.Duration) Option {
	return func(o *Options) {
		o.WriteBufferDuration = d
	}
}

// JetStream configures the JetStream context used to read the data version from the stream metadata
func JetStream(js nats.JetStreamContext) Option {
	return func(o *Options) {
		o.JetStream = js
	}
}
