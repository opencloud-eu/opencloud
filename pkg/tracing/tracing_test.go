package tracing

import (
	"os"
	"testing"
)

func TestResolveConfig(t *testing.T) {
	tests := map[string]struct {
		cfg              Config
		exporter         exporterKind
		legacyEnv        map[string]string
		wantWarningCount int
	}{
		"defaults to none": {
			cfg:       Config{},
			exporter:  exporterNone,
			legacyEnv: map[string]string{},
		},
		"legacy enabled picks otlp": {
			cfg:              Config{Enabled: true},
			exporter:         exporterOTLP,
			legacyEnv:        map[string]string{},
			wantWarningCount: 1,
		},
		"explicit console exporter": {
			cfg:       Config{Exporter: "console"},
			exporter:  exporterConsole,
			legacyEnv: map[string]string{},
		},
		"legacy endpoint populates env": {
			cfg:      Config{Enabled: true, Endpoint: "collector:4317"},
			exporter: exporterOTLP,
			legacyEnv: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "collector:4317",
				"OTEL_EXPORTER_OTLP_INSECURE": "true",
			},
			wantWarningCount: 2,
		},
		"legacy jaeger maps to otlp": {
			cfg:              Config{Type: "jaeger"},
			exporter:         exporterOTLP,
			legacyEnv:        map[string]string{},
			wantWarningCount: 3,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			resolved, err := resolveConfig(tt.cfg)
			if err != nil {
				t.Fatalf("resolveConfig returned error: %v", err)
			}
			if resolved.exporter != tt.exporter {
				t.Fatalf("unexpected exporter: got %q want %q", resolved.exporter, tt.exporter)
			}
			if len(resolved.warnings) != tt.wantWarningCount {
				t.Fatalf("unexpected warning count: got %d want %d", len(resolved.warnings), tt.wantWarningCount)
			}
			if len(resolved.legacyEnv) != len(tt.legacyEnv) {
				t.Fatalf("unexpected legacy env len: got %d want %d", len(resolved.legacyEnv), len(tt.legacyEnv))
			}
			for key, value := range tt.legacyEnv {
				if resolved.legacyEnv[key] != value {
					t.Fatalf("legacy env %s mismatch: got %q want %q", key, resolved.legacyEnv[key], value)
				}
			}
		})
	}
}

func TestApplyLegacyEnv(t *testing.T) {
	const envKey = "OTEL_EXPORTER_OTLP_ENDPOINT"
	t.Setenv(envKey, "existing")

	applyLegacyEnv(map[string]string{
		envKey: "legacy",
	})

	if got := os.Getenv(envKey); got != "existing" {
		t.Fatalf("expected existing env to be preserved, got %q", got)
	}
}

func TestNewOTLPClientFromEnv(t *testing.T) {
	t.Run("defaults to grpc", func(t *testing.T) {
		t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "")
		client, err := newOTLPClientFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatalf("expected client instance")
		}
	})

	t.Run("http protocol", func(t *testing.T) {
		t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
		client, err := newOTLPClientFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatalf("expected client instance")
		}
	})
}
