package grpc

import (
	"os"
	"strings"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/registry"
)

func TestMain(m *testing.M) {
	// NewClient looks up the service registry; use the in-memory one so that
	// the tests don't need a running NATS server.
	registry.GetRegistry(registry.Inmemory())
	os.Exit(m.Run())
}

// setInsecureEnv sets OC_INSECURE for the duration of the test, or unsets it
// when unset is true. The original value is restored afterwards.
func setInsecureEnv(t *testing.T, value string, unset bool) {
	t.Helper()
	t.Setenv("OC_INSECURE", value)
	if unset {
		if err := os.Unsetenv("OC_INSECURE"); err != nil {
			t.Fatalf("unsetting OC_INSECURE: %v", err)
		}
	}
}

func TestNewClientInsecureTLSMode(t *testing.T) {
	tests := []struct {
		name       string
		ocInsecure string
		unset      bool
		wantErr    bool
	}{
		{name: "OC_INSECURE not set", unset: true, wantErr: true},
		{name: "OC_INSECURE empty", ocInsecure: "", wantErr: true},
		{name: "OC_INSECURE false", ocInsecure: "false", wantErr: true},
		{name: "OC_INSECURE invalid", ocInsecure: "yes-please", wantErr: true},
		{name: "OC_INSECURE true", ocInsecure: "true", wantErr: false},
		{name: "OC_INSECURE TRUE", ocInsecure: "TRUE", wantErr: false},
		{name: "OC_INSECURE 1", ocInsecure: "1", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setInsecureEnv(t, tt.ocInsecure, tt.unset)

			c, err := NewClient(WithTLSMode("insecure"))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error for TLS mode 'insecure', got none")
				}
				if c != nil {
					t.Errorf("expected no client on error, got %v", c)
				}
				if !strings.Contains(err.Error(), "OC_INSECURE") {
					t.Errorf("expected the error to name OC_INSECURE, got %q", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if c == nil {
				t.Fatal("expected a client, got nil")
			}
		})
	}
}

func TestNewClientOtherTLSModesIgnoreOCInsecure(t *testing.T) {
	for _, mode := range []string{"", "off", "on"} {
		t.Run("mode "+mode, func(t *testing.T) {
			setInsecureEnv(t, "", true)

			c, err := NewClient(WithTLSMode(mode))
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if c == nil {
				t.Fatal("expected a client, got nil")
			}
		})
	}
}
