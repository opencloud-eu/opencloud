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

// Default values for the guard policies (nonce, backoff and quota limiters).
const (
	defaultSessionCookieName   = "__Host-oc_guest_session"
	defaultNonceCookieBaseName = "__Host-oc_guest_nonce"
	defaultPinAndNonceTTL      = 30 * time.Minute
	defaultBackoffMaxAttempts  = 5
	defaultBackoffBase         = time.Second
	defaultBackoffCap          = 30 * time.Second
	defaultQuotaCooldown       = time.Minute
	defaultQuotaWindow         = 24 * time.Hour
	defaultQuotaDailyCap       = 20
	// Renewals with a valid session issue a PIN, so their quota is small: enough
	// to refresh a session before it expires, plus a few retries.
	defaultSessionQuotaCooldown = 5 * time.Minute
	defaultSessionQuotaWindow   = 24 * time.Hour
	defaultSessionQuotaDailyCap = 5
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
			Addr:      "127.0.0.1:9268",
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
			CookieName: defaultSessionCookieName,
			TTL:        24 * time.Hour,
		},
		Nonce: config.Nonce{
			CookieBaseName: defaultNonceCookieBaseName,
		},
		PinAndNonceTTL: defaultPinAndNonceTTL,
		BackoffLimiter: config.BackoffLimiter{
			MaxAttempts: defaultBackoffMaxAttempts,
			BackoffBase: defaultBackoffBase,
			BackoffCap:  defaultBackoffCap,
		},
		QuotaLimiter: config.QuotaLimiter{
			DailyCap: defaultQuotaDailyCap,
			Cooldown: defaultQuotaCooldown,
			Window:   defaultQuotaWindow,
		},
		SessionQuotaLimiter: config.SessionQuotaLimiter{
			DailyCap: defaultSessionQuotaDailyCap,
			Cooldown: defaultSessionQuotaCooldown,
			Window:   defaultSessionQuotaWindow,
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

	if cfg.TokenManager == nil && cfg.Commons != nil && cfg.Commons.TokenManager != nil {
		cfg.TokenManager = &config.TokenManager{
			JWTSecret: cfg.Commons.TokenManager.JWTSecret,
		}
	} else if cfg.TokenManager == nil {
		cfg.TokenManager = &config.TokenManager{}
	}

	if cfg.Commons != nil {
		cfg.HTTP.TLS = cfg.Commons.HTTPServiceTLS
	}

	if cfg.Nonce.CookieBaseName == "" {
		cfg.Nonce.CookieBaseName = defaultNonceCookieBaseName
	}
	if cfg.PinAndNonceTTL <= 0 {
		cfg.PinAndNonceTTL = defaultPinAndNonceTTL
	}
	if cfg.BackoffLimiter.MaxAttempts <= 0 {
		cfg.BackoffLimiter.MaxAttempts = defaultBackoffMaxAttempts
	}
	if cfg.BackoffLimiter.BackoffBase <= 0 {
		cfg.BackoffLimiter.BackoffBase = defaultBackoffBase
	}
	if cfg.BackoffLimiter.BackoffCap <= 0 {
		cfg.BackoffLimiter.BackoffCap = defaultBackoffCap
	}
	if cfg.QuotaLimiter.Cooldown <= 0 {
		cfg.QuotaLimiter.Cooldown = defaultQuotaCooldown
	}
	if cfg.QuotaLimiter.Window <= 0 {
		cfg.QuotaLimiter.Window = defaultQuotaWindow
	}
	if cfg.QuotaLimiter.DailyCap <= 0 {
		cfg.QuotaLimiter.DailyCap = defaultQuotaDailyCap
	}
	if cfg.SessionQuotaLimiter.Cooldown <= 0 {
		cfg.SessionQuotaLimiter.Cooldown = defaultSessionQuotaCooldown
	}
	if cfg.SessionQuotaLimiter.Window <= 0 {
		cfg.SessionQuotaLimiter.Window = defaultSessionQuotaWindow
	}
	if cfg.SessionQuotaLimiter.DailyCap <= 0 {
		cfg.SessionQuotaLimiter.DailyCap = defaultSessionQuotaDailyCap
	}
}

// Sanitize sanitizes the config
func Sanitize(cfg *config.Config) {
	// sanitize config
}
