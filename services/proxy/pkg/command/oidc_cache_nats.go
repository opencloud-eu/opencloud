package command

import (
	"crypto/tls"

	"github.com/nats-io/nats.go"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
)

// connectOIDCNATSCache mirrors the NATS authentication and TLS options used by
// reva's store factory. Cleanup needs the same access as ordinary cache writes.
func connectOIDCNATSCache(cfg *config.Cache) (*nats.Conn, error) {
	opts := nats.GetDefaultOptions()
	opts.Name = "opencloud-proxy-oidc-cleanup"
	opts.Servers = cfg.Nodes
	opts.User, opts.Password = cfg.AuthUsername, cfg.AuthPassword
	if cfg.EnableTLS {
		if cfg.TLSRootCACertificate != "" {
			if err := nats.RootCAs(cfg.TLSRootCACertificate)(&opts); err != nil {
				return nil, err
			}
		} else {
			if err := nats.Secure(&tls.Config{
				MinVersion: tls.VersionTLS12, InsecureSkipVerify: cfg.TLSInsecure, //nolint:gosec
			})(&opts); err != nil {
				return nil, err
			}
		}
	}
	return opts.Connect()
}
