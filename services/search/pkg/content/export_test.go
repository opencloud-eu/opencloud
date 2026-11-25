package content

import "time"

// SetProbeTimeout shortens the version check, a test of a hanging tika would
// otherwise wait seconds per probe
func SetProbeTimeout(d time.Duration) (restore func()) {
	previous := probeTimeout
	probeTimeout = d
	return func() { probeTimeout = previous }
}
