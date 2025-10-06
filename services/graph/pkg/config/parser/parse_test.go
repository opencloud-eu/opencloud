package parser_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/parser"
)

var _ = Describe("Validate", func() {
	var cfg *config.Config

	BeforeEach(func() {
		cfg = defaults.DefaultConfig()
		cfg.Application.ID = "graph-app-id"
		cfg.ServiceAccount.ServiceAccountID = "graph-service-account"
		cfg.ServiceAccount.ServiceAccountSecret = "graph-service-password"
		cfg.Commons = &shared.Commons{
			TokenManager: &shared.TokenManager{
				JWTSecret: "jwt-secret",
			},
		}
		defaults.EnsureDefaults(cfg)
	})

	When("multi-tenant support is disabled", func() {
		It("should accept a setup with the 'cs3' identity backend", func() {
			cfg.Identity.Backend = "cs3"
			err := parser.Validate(cfg)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should accept a setup with the 'ldap' identity backend", func() {
			cfg.Identity.Backend = "ldap"
			// we need to set a password to pass validation
			cfg.Identity.LDAP.BindPassword = "bind-password"
			err := parser.Validate(cfg)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("multi-tenant support is disabled", func() {
		BeforeEach(func() {
			cfg.Commons.MultiTenantEnabled = true
		})
		It("should accept a setup with the 'cs3' identity backend", func() {
			cfg.Identity.Backend = "cs3"
			err := parser.Validate(cfg)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject a setup with the 'ldap' identity backend", func() {
			cfg.Identity.Backend = "ldap"
			cfg.Identity.LDAP.BindPassword = "bind-password"
			err := parser.Validate(cfg)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("The identity backend must be set to 'cs3' for the 'graph' service.")))
		})
	})

	It("rejcts a setup with an invalid identity backend", func() {
		cfg.Identity.Backend = "invalid-backend"
		err := parser.Validate(cfg)
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("is not a valid identity backend")))
	})
})
