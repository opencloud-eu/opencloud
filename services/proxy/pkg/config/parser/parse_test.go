package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config/parser"
	"github.com/stretchr/testify/require"
)

func TestParseOIDCAudiences(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		env     string
		method  string
		setEnv  bool
		want    []string
		wantErr string
	}{
		{name: "unset defaults to disabled"},
		{name: "YAML", yaml: "oidc:\n  audiences: [opencloud, opencloud-api]\n", want: []string{"opencloud", "opencloud-api"}},
		{name: "empty YAML list", yaml: "oidc:\n  audiences: []\n"},
		{name: "null YAML list", yaml: "oidc:\n  audiences: null\n"},
		{name: "ENV", setEnv: true, env: "opencloud,opencloud-api", want: []string{"opencloud", "opencloud-api"}},
		{name: "ENV trims list entries", setEnv: true, env: " opencloud, opencloud-api ", want: []string{"opencloud", "opencloud-api"}},
		{name: "ENV precedence", yaml: "oidc:\n  audiences: [yaml-audience]\n", setEnv: true, env: "env-audience", want: []string{"env-audience"}},
		{name: "empty ENV disables YAML", yaml: "oidc:\n  audiences: [opencloud]\n", setEnv: true},
		{name: "existing ENV empty segment handling", setEnv: true, env: "opencloud,,opencloud-api", want: []string{"opencloud", "opencloud-api"}},
		{name: "YAML blank entry", yaml: "oidc:\n  audiences: ['']\n", wantErr: "empty or whitespace-only"},
		{name: "YAML whitespace entry", yaml: "oidc:\n  audiences: ['  ']\n", wantErr: "empty or whitespace-only"},
		{name: "ENV whitespace entry", setEnv: true, env: "opencloud, ", wantErr: "empty or whitespace-only"},
		{name: "ENV whitespace only", setEnv: true, env: " ", wantErr: "empty or whitespace-only"},
		{name: "YAML preserves case", yaml: "oidc:\n  audiences: [OpenCloud]\n", want: []string{"OpenCloud"}},
		{name: "YAML incompatible verification", yaml: "oidc:\n  audiences: [opencloud]\n  access_token_verify_method: none\n", wantErr: "require access_token_verify_method to be 'jwt'"},
		{name: "ENV incompatible verification", setEnv: true, env: "opencloud", method: "none", wantErr: "require access_token_verify_method to be 'jwt'"},
		{name: "ENV enables JWT over YAML none", yaml: "oidc:\n  audiences: [opencloud]\n  access_token_verify_method: none\n", method: "jwt", want: []string{"opencloud"}},
		{name: "empty ENV restores none compatibility", yaml: "oidc:\n  audiences: [opencloud]\n  access_token_verify_method: none\n", setEnv: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("OC_CONFIG_DIR", dir)
			t.Setenv("PROXY_OIDC_ACCESS_TOKEN_VERIFY_METHOD", "")
			require.NoError(t, os.Unsetenv("PROXY_OIDC_ACCESS_TOKEN_VERIFY_METHOD"))
			if tt.method != "" {
				t.Setenv("PROXY_OIDC_ACCESS_TOKEN_VERIFY_METHOD", tt.method)
			}
			t.Setenv("PROXY_OIDC_SKIP_USER_INFO", "false")
			t.Setenv("PROXY_OIDC_AUDIENCES", "")
			require.NoError(t, os.Unsetenv("PROXY_OIDC_AUDIENCES"))
			if tt.setEnv {
				t.Setenv("PROXY_OIDC_AUDIENCES", tt.env)
			}
			require.NoError(t, os.WriteFile(filepath.Join(dir, "proxy.yaml"), []byte(tt.yaml), 0600))
			cfg := validProxyConfig()
			err := parser.ParseConfig(cfg)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.ErrorContains(t, err, "PROXY_OIDC_AUDIENCES")
				return
			}
			require.NoError(t, err)
			if len(tt.want) == 0 {
				require.Empty(t, cfg.OIDC.Audiences)
			} else {
				require.Equal(t, tt.want, cfg.OIDC.Audiences)
			}
		})
	}
}

func TestValidateOIDCAudiences(t *testing.T) {
	for _, tt := range []struct {
		name      string
		audiences []string
		method    string
		wantErr   string
	}{
		{name: "disabled JWT", method: "jwt"},
		{name: "disabled none", method: "none"},
		{name: "enabled JWT", audiences: []string{"opencloud"}, method: "jwt"},
		{name: "enabled none", audiences: []string{"opencloud"}, method: "none", wantErr: "require access_token_verify_method to be 'jwt'"},
		{name: "blank", audiences: []string{""}, method: "jwt", wantErr: "empty or whitespace-only"},
		{name: "whitespace", audiences: []string{" \t"}, method: "jwt", wantErr: "empty or whitespace-only"},
		{name: "mixed valid and blank", audiences: []string{"opencloud", ""}, method: "jwt", wantErr: "empty or whitespace-only"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validProxyConfig()
			cfg.OIDC.Audiences = tt.audiences
			cfg.OIDC.AccessTokenVerifyMethod = tt.method
			err := parser.Validate(cfg)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.ErrorContains(t, err, "PROXY_OIDC_AUDIENCES")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func validProxyConfig() *config.Config {
	cfg := defaults.FullDefaultConfig()
	cfg.MachineAuthAPIKey = "test-machine-key"
	cfg.TransferSecret = "test-transfer-secret"
	cfg.ServiceAccount.ServiceAccountID = "test-service-account"
	cfg.ServiceAccount.ServiceAccountSecret = "test-service-secret"
	cfg.Commons = &shared.Commons{URLSigningSecret: "test-url-secret"}
	return cfg
}
