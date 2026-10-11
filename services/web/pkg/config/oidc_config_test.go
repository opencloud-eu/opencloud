package config_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config/parser"
)

func TestOIDCWebConfiguration(t *testing.T) {
	Describe("OIDC Configuration Environment Decoding", func() {
		var (
			testEnvVars map[string]string
			savedEnv    map[string]string
		)

		BeforeEach(func() {
			testEnvVars = map[string]string{
				"WEB_OIDC_AUTHORITY":                "https://idp.example.com",
				"WEB_OIDC_METADATA_URL":             "https://idp.example.com/.well-known/openid-configuration",
				"WEB_OIDC_CLIENT_ID":                "web",
				"WEB_OIDC_CLIENT_SECRET":            "secret",
				"WEB_OIDC_RESPONSE_TYPE":            "code",
				"WEB_OIDC_SCOPE":                    "openid profile email",
				"WEB_OIDC_CLIENT_AUTHENTICATION":    "client_secret_post",
				"WEB_OIDC_POST_LOGOUT_REDIRECT_URI": "https://app.example.com/login",
			}

			// pretend env are callee saved
			// not sure what the standards for env vars between tests are
			savedEnv = make(map[string]string)
			for k := range testEnvVars {
				if val, exists := os.LookupEnv(k); exists {
					savedEnv[k] = val
				}
			}

			for k, v := range testEnvVars {
				err := os.Setenv(k, v)
				Expect(err).To(BeNil())
			}
		})

		AfterEach(func() {
			// restore
			for k := range testEnvVars {
				err := os.Unsetenv(k)
				Expect(err).To(BeNil())
			}
		})

		It("successfully decodes OIDC environment variables into the struct", func() {
			var cfg config.Config

			err := parser.ParseConfig(&cfg)
			Expect(err).To(BeNil())

			oidcCfg := &cfg.Web.Config.OpenIDConnect

			Expect(oidcCfg.Authority).To(Equal("https://idp.example.com"))
			Expect(oidcCfg.MetadataURL).To(Equal("https://idp.example.com/.well-known/openid-configuration"))
			Expect(oidcCfg.ClientID).To(Equal("web"))
			Expect(oidcCfg.ClientSecret).To(Equal("secret"))
			Expect(oidcCfg.ResponseType).To(Equal("code"))
			Expect(oidcCfg.Scope).To(Equal("openid profile email"))
			Expect(oidcCfg.ClientAuthentication).To(Equal("client_secret_post"))
			Expect(oidcCfg.PostLogoutRedirectURI).To(Equal("https://app.example.com/login"))
		})
	})
}
