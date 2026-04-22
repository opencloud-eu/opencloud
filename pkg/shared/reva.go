package shared

import (
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

var defaultRevaConfig = Reva{
	Address: "eu.opencloud.api.gateway",
}

func DefaultRevaConfig() *Reva {
	// copy
	ret := defaultRevaConfig
	return &ret
}

func (r *Reva) GetRevaOptions() []pool.Option {
	tm, _ := pool.StringToTLSMode(r.TLS.Mode)
	opts := []pool.Option{
		pool.WithTLSMode(tm),
	}
	return opts
}

func (r *Reva) GetGRPCClientConfig() map[string]any {
	return map[string]any{
		"tls_mode":   r.TLS.Mode,
		"tls_cacert": r.TLS.CACert,
	}
}
