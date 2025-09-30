package icapclient

import (
	"time"
)

// Config is the shared configuration for the icap client library.
type Config struct {
	ICAPConn ICAPConnConfig
}

// DefaultConfig returns the default configuration for the icap client library.
func DefaultConfig() Config {
	return Config{
		ICAPConn: ICAPConnConfig{
			Timeout: 15 * time.Second,
		},
	}
}

// ConfigOption is a function that configures the icap client.
type ConfigOption func(*Config)

// WithICAPConnectionTimeout sets the timeout for the connection to the icap server.
func WithICAPConnectionTimeout(timeout time.Duration) ConfigOption {
	return func(cfg *Config) {
		if timeout <= 0 {
			return
		}

		cfg.ICAPConn.Timeout = timeout
	}
}
