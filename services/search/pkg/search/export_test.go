package search

import "time"

// SetUnreachableRetryDelay shortens the wait between extraction retries, a walk
// test would otherwise sleep a second per retry
func SetUnreachableRetryDelay(d time.Duration) (restore func()) {
	previous := unreachableRetryDelay
	unreachableRetryDelay = d
	return func() { unreachableRetryDelay = previous }
}
