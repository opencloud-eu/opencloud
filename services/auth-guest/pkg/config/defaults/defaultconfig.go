// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	"path"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
)

// FullDefaultConfig returns the full default config
func FullDefaultConfig() *config.Config {
	cfg := DefaultConfig()
	EnsureDefaults(cfg)
	Sanitize(cfg)
	return cfg
}

// DefaultConfig return the default configuration
func DefaultConfig() *config.Config {
	return &config.Config{
		Debug: config.Debug{
			Addr:   "127.0.0.1:9267",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		Service: config.Service{
			Name: "auth-guest",
		},
		NumConsumers: 1,
		Events: config.Events{
			Endpoint:  "127.0.0.1:9233",
			Cluster:   "opencloud-cluster",
			EnableTLS: false,
		},
		RevaGateway: shared.DefaultRevaConfig().Address,
		GRPC: config.GRPCConfig{
			Addr:      "127.0.0.1:9265",
			Namespace: "eu.opencloud.api",
			Protocol:  "tcp",
		},
		HTTP: config.HTTP{
			Addr:      "127.0.0.1:9266",
			Root:      "/graph",
			Namespace: "eu.opencloud.web",
			CORS: config.CORS{
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
				AllowedHeaders:   []string{"Authorization", "Origin", "Content-Type", "Accept", "X-Requested-With", "X-Request-Id", "Ocs-Apirequest"},
				AllowCredentials: true,
			},
		},
		Storage: config.Storage{
			RootDirectory: path.Join(defaults.BaseDataPath(), "auth-guest"),
		},
		JWT: config.JWT{
			CookieName: "__Host-oc_guest_session",
			TTL:        24 * time.Hour,
		},
	}
}

// EnsureDefaults ensures the config contains default values
func EnsureDefaults(cfg *config.Config) {
	if cfg.LogLevel == "" {
		cfg.LogLevel = "error"
	}
	if cfg.GRPCClientTLS == nil && cfg.Commons != nil {
		cfg.GRPCClientTLS = structs.CopyOrZeroValue(cfg.Commons.GRPCClientTLS)
	}

	if cfg.GRPC.TLS == nil && cfg.Commons != nil {
		cfg.GRPC.TLS = structs.CopyOrZeroValue(cfg.Commons.GRPCServiceTLS)
	}

	if cfg.TokenManager == nil {
		cfg.TokenManager = &config.TokenManager{}
	}

	if cfg.Commons != nil {
		cfg.HTTP.TLS = cfg.Commons.HTTPServiceTLS
	}
}

// Sanitize sanitizes the config
func Sanitize(cfg *config.Config) {
	// sanitize config
}
