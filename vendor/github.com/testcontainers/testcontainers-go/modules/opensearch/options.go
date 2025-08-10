package opensearch

import "github.com/testcontainers/testcontainers-go"

// Options is a struct for specifying options for the OpenSearch container.
type Options struct {
	Password string
	Username string
}

func defaultOptions() *Options {
	return &Options{
		Username: defaultUsername,
		Password: defaultPassword,
	}
}

// Compiler check to ensure that Option implements the testcontainers.ContainerCustomizer interface.
var _ testcontainers.ContainerCustomizer = (*Option)(nil)

// Option is an option for the OpenSearch container.
type Option func(*Options)

// Customize is a NOOP. It's defined to satisfy the testcontainers.ContainerCustomizer interface.
func (o Option) Customize(*testcontainers.GenericContainerRequest) error {
	// NOOP to satisfy interface.
	return nil
}

// WithPassword sets the password for the OpenSearch container.
func WithPassword(password string) Option {
	return func(o *Options) {
		o.Password = password
	}
}

// WithUsername sets the username for the OpenSearch container.
func WithUsername(username string) Option {
	return func(o *Options) {
		o.Username = username
	}
}
