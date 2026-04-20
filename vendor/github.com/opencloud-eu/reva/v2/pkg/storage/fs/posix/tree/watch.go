package tree

import (
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
)

var ErrUnsupportedWatcher = errtypes.NotSupported("watching the filesystem is not supported on this platform")

// NoopWatcher is a watcher that does nothing
type NoopWatcher struct{}

// Watch does nothing
func (*NoopWatcher) Watch(_ string) {}
