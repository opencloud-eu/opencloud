package defaults

import (
	"path/filepath"

	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/nats/pkg/config"
)

// NOTE: Most of this configuration is not needed to keep it as simple as possible
// TODO: Clean up unneeded configuration

// FullDefaultConfig returns a fully initialized default configuration
func FullDefaultConfig() *config.Config {
	cfg := DefaultConfig()
	EnsureDefaults(cfg)
	Sanitize(cfg)
	return cfg
}

// DefaultConfig returns a basic default configuration
func DefaultConfig() *config.Config {
	baseDataPath := defaults.BaseDataPath()
	return &config.Config{
		Debug: config.Debug{
			Addr:   "127.0.0.1:9234",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		Service: config.Service{
			Name: "nats",
		},
		Nats: config.Nats{
			Host:      "127.0.0.1",
			Port:      9233,
			ClusterID: "opencloud-cluster",
			StoreDir:  filepath.Join(baseDataPath, "nats"),
			TLSCert:   filepath.Join(baseDataPath, "nats", "tls.crt"),
			TLSKey:    filepath.Join(baseDataPath, "nats", "tls.key"),
			EnableTLS: false,
			Monitoring: config.Monitoring{
				Enabled:   false,       // don't enable by default for now
				Host:      "127.0.0.1", // only bind to localhost by default for security reasons
				Port:      8222,        // default NATS monitoring HTTP endpoint port
				EnableTLS: false,
			},
		},
	}
}

// EnsureDefaults adds default values to the configuration if they are not set yet
func EnsureDefaults(cfg *config.Config) {
	if cfg.LogLevel == "" {
		cfg.LogLevel = "error"
	}
}

// Sanitize sanitizes the configuration
func Sanitize(cfg *config.Config) {
	// nothing to sanitize here atm
}
