package capabilities

import (
	"sync/atomic"

	"github.com/opencloud-eu/reva/v3/pkg/owncloud/ocs"
)

// allow the consuming part to change defaults, e.g., tests
var defaultCapabilities atomic.Pointer[ocs.Capabilities]

func init() { //nolint:gochecknoinits
	ResetDefault()
}

// ResetDefault resets the default [Capabilities] to the default values.
func ResetDefault() {
	defaultCapabilities.Store(
		&ocs.Capabilities{
			Theme: &ocs.CapabilitiesTheme{
				Logo: &ocs.CapabilitiesThemeLogo{
					PermittedFileTypes: map[string]string{
						".jpg":  "image/jpeg",
						".jpeg": "image/jpeg",
						".png":  "image/png",
						".gif":  "image/gif",
					},
				},
			},
		},
	)
}

// Default returns the default [Capabilities].
func Default() *ocs.Capabilities { return defaultCapabilities.Load() }
