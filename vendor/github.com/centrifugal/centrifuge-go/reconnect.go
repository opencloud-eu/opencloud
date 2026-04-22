package centrifuge

import (
	"time"

	"github.com/jpillora/backoff"
)

type reconnectStrategy interface {
	timeBeforeNextAttempt(attempt int) time.Duration
}

type backoffReconnect struct {
	// Factor is the multiplying factor for each increment step.
	Factor float64
	// Jitter eases contention by randomizing backoff steps.
	Jitter bool
	// MinMilliseconds is a minimum value of reconnect interval.
	MinDelay time.Duration
	// MaxMilliseconds is a maximum value of reconnect interval.
	MaxDelay time.Duration
}

func (r *backoffReconnect) timeBeforeNextAttempt(attempt int) time.Duration {
	b := &backoff.Backoff{
		Min:    r.MinDelay,
		Max:    r.MaxDelay,
		Factor: r.Factor,
		Jitter: r.Jitter,
	}
	return b.ForAttempt(float64(attempt))
}

// newBackoffReconnect creates a new backoff reconnect strategy with custom min and max delays.
// If minDelay or maxDelay is zero, it uses the default values.
func newBackoffReconnect(minDelay, maxDelay time.Duration) reconnectStrategy {
	if minDelay == 0 {
		minDelay = 200 * time.Millisecond
	}
	if maxDelay == 0 {
		maxDelay = 20 * time.Second
	}
	return &backoffReconnect{
		MinDelay: minDelay,
		MaxDelay: maxDelay,
		Factor:   2,
		Jitter:   true,
	}
}
