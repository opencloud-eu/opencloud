package grpc

import (
	"os"
	"strings"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"go-micro.dev/v4/client"
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

func assertInsecureModeRejected(t *testing.T, c client.Client, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error for TLS mode 'insecure', got none")
	}
	if c != nil {
		t.Errorf("expected no client on error, got %v", c)
	}
	if !strings.Contains(err.Error(), "OC_INSECURE") {
		t.Errorf("expected the error to name OC_INSECURE, got %q", err)
	}
}

func assertClientCreated(t *testing.T, c client.Client, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if c == nil {
		t.Fatal("expected a client, got nil")
	}
}

func TestNewClientInsecureTLSMode(t *testing.T) {
	tests := []struct {
		name       string
		opts       []ClientOption
		ocInsecure string
		unsetEnv   bool
		wantErr    bool
	}{
		{
			name:     "insecure allowed",
			opts:     []ClientOption{WithTLSInsecure(true)},
			unsetEnv: true,
			wantErr:  false,
		},
		{
			name:     "insecure not allowed",
			opts:     []ClientOption{WithTLSInsecure(false)},
			unsetEnv: true,
			wantErr:  true,
		},
		{
			name:     "insecure option not given",
			unsetEnv: true,
			wantErr:  true,
		},
		{
			// the environment is applied through the configuration,
			// NewClient itself must only look at the option
			name:       "insecure not allowed, OC_INSECURE set in the environment",
			opts:       []ClientOption{WithTLSInsecure(false)},
			ocInsecure: "true",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setInsecureEnv(t, tt.ocInsecure, tt.unsetEnv)

			c, err := NewClient(append([]ClientOption{WithTLSMode("insecure")}, tt.opts...)...)

			if tt.wantErr {
				assertInsecureModeRejected(t, c, err)
				return
			}
			assertClientCreated(t, c, err)
		})
	}
}

func TestNewClientOtherTLSModesIgnoreInsecure(t *testing.T) {
	for _, mode := range []string{"", "off", "on"} {
		t.Run("mode "+mode, func(t *testing.T) {
			setInsecureEnv(t, "", true)

			c, err := NewClient(WithTLSMode(mode), WithTLSInsecure(false))
			assertClientCreated(t, c, err)
		})
	}
}

func TestGetClientOptionsInsecure(t *testing.T) {
	t.Run("insecure allowed in the configuration", func(t *testing.T) {
		setInsecureEnv(t, "", true)

		c, err := NewClient(GetClientOptions(&shared.GRPCClientTLS{Mode: "insecure", Insecure: true})...)
		assertClientCreated(t, c, err)
	})

	t.Run("insecure not allowed in the configuration", func(t *testing.T) {
		setInsecureEnv(t, "", true)

		c, err := NewClient(GetClientOptions(&shared.GRPCClientTLS{Mode: "insecure"})...)
		assertInsecureModeRejected(t, c, err)
	})
}
