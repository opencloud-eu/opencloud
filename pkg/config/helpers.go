package config

import (
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/config/binder"
)

// BindSourcesToStructs assigns any config value from a config file / env variable to struct `dst`.
// Backward-compatible re-export; the implementation lives in pkg/config/binder.
var BindSourcesToStructs = binder.BindSourcesToStructs

// LocalEndpoint returns the local endpoint for a given protocol and address.
// Use it when configuring the reva runtime to get a service endpoint in the same
// runtime, e.g. a gateway talking to an authregistry service.
func LocalEndpoint(protocol, addr string) string {
	localEndpoint := addr
	switch protocol {
	case "tcp":
		parts := strings.SplitN(addr, ":", 2)
		if len(parts) == 2 {
			localEndpoint = "dns:127.0.0.1:" + parts[1]
		}
	case "unix":
		localEndpoint = "unix:" + addr
	}
	return localEndpoint
}
