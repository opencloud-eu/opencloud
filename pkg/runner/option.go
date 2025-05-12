package runner

import (
	"time"
)

var (
	// DefaultInterruptDuration is the default value for the `WithInterruptDuration`
	// for the "regular" runners. This global value can be adjusted if needed.
	// TODO: To discuss the default timeout
	DefaultInterruptDuration = 20 * time.Second
	// DefaultGroupInterruptDuration is the default value for the `WithInterruptDuration`
	// for the group runners. This global value can be adjusted if needed.
	// TODO: To discuss the default timeout
	DefaultGroupInterruptDuration = 25 * time.Second
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	InterruptDuration time.Duration
}

// WithInterruptDuration provides a function to set the interrupt
// duration option.
func WithInterruptDuration(val time.Duration) Option {
	return func(o *Options) {
		o.InterruptDuration = val
	}
}
