package init

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"
)

// createConfig runs CreateConfig into a temporary directory and returns the
// written configuration file, parsed into a generic map.
func createConfig(t *testing.T, insecure bool) map[string]any {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config")

	if err := CreateConfig(insecure, false, false, configPath, "admin", true); err != nil {
		t.Fatalf("creating the config: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(configPath, configFilename))
	if err != nil {
		t.Fatalf("reading the config: %v", err)
	}
	cfg := map[string]any{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parsing the config: %v", err)
	}
	return cfg
}

func TestCreateConfigGRPCClientTLS(t *testing.T) {
	t.Run("insecure allows the insecure grpc client TLS mode", func(t *testing.T) {
		cfg := createConfig(t, true)

		grpcClientTLS, ok := cfg["grpc_client_tls"].(map[any]any)
		if !ok {
			t.Fatalf("expected a grpc_client_tls section, got %#v", cfg["grpc_client_tls"])
		}
		if grpcClientTLS["insecure"] != true {
			t.Errorf("expected grpc_client_tls.insecure to be true, got %#v", grpcClientTLS["insecure"])
		}
		// only the permission is written, the mode stays as configured elsewhere
		if _, ok := grpcClientTLS["mode"]; ok {
			t.Errorf("expected no grpc_client_tls.mode, got %#v", grpcClientTLS["mode"])
		}
	})

	t.Run("secure does not write a grpc_client_tls section", func(t *testing.T) {
		cfg := createConfig(t, false)

		if v, ok := cfg["grpc_client_tls"]; ok {
			t.Errorf("expected no grpc_client_tls section, got %#v", v)
		}
	})
}
