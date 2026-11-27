package shared_test

import (
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
	"github.com/opencloud-eu/opencloud/pkg/shared"
)

func TestGRPCClientTLSInsecureFromEnv(t *testing.T) {
	tests := []struct {
		ocInsecure string
		want       bool
	}{
		{ocInsecure: "true", want: true},
		{ocInsecure: "1", want: true},
		{ocInsecure: "false", want: false},
	}

	for _, tt := range tests {
		t.Run("OC_INSECURE="+tt.ocInsecure, func(t *testing.T) {
			t.Setenv("OC_INSECURE", tt.ocInsecure)

			cfg := shared.GRPCClientTLS{}
			if err := envdecode.Decode(&cfg); err != nil {
				t.Fatalf("decoding the environment: %v", err)
			}
			if cfg.Insecure != tt.want {
				t.Errorf("expected Insecure to be %v, got %v", tt.want, cfg.Insecure)
			}
		})
	}
}
