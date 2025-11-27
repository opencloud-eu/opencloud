package defaults

import (
	"time"

	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/config"
)

// FullDefaultConfig returns a fully initialized default configuration which is needed for doc generation.
func FullDefaultConfig() *config.Config {
	cfg := DefaultConfig()
	EnsureDefaults(cfg)
	Sanitize(cfg)
	return cfg
}

// DefaultConfig returns the services default config
func DefaultConfig() *config.Config {
	return &config.Config{
		Debug: config.Debug{
			Addr:  "127.0.0.1:9277",
			Token: "",
		},
		Service: config.Service{
			Name: "antivirus",
		},
		Events: config.Events{
			Endpoint: "127.0.0.1:9233",
			Cluster:  "opencloud-cluster",
		},
		Workers:              10,
		InfectedFileHandling: "delete",
		// defaults from clamav sample conf: MaxScanSize=400M, MaxFileSize=100M, StreamMaxLength=100M
		// https://github.com/Cisco-Talos/clamav/blob/main/etc/clamd.conf.sample
		MaxScanSize:     "100MB",
		MaxScanSizeMode: config.MaxScanSizeModePartial,
		Scanner: config.Scanner{
			Type: config.ScannerTypeClamAV,
			ClamAV: config.ClamAV{
				Socket:  "/run/clamav/clamd.ctl",
				Timeout: 5 * time.Minute,
			},
			ICAP: config.ICAP{
				URL:     "icap://127.0.0.1:1344",
				Service: "avscan",
				Timeout: 5 * time.Minute,
			},
		},
	}
}

// EnsureDefaults adds default values to the configuration if they are not set yet
func EnsureDefaults(cfg *config.Config) {
	if cfg.Log == nil {
		cfg.Log = &config.Log{}
	}
}

// Sanitize sanitizes the configuration
func Sanitize(cfg *config.Config) {
	defaultConfig := DefaultConfig()

	if cfg.MaxScanSize == "" {
		cfg.MaxScanSize = defaultConfig.MaxScanSize
	}
}
