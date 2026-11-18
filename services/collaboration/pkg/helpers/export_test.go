package helpers

import "time"

// SetFirstRetryDelayForTest sets the delay of the first retry of
// RefreshAppProvider after a failed fetch, and returns a function that
// restores it.
func SetFirstRetryDelayForTest(d time.Duration) (restore func()) {
	old := firstRetryDelay
	firstRetryDelay = d
	return func() { firstRetryDelay = old }
}
