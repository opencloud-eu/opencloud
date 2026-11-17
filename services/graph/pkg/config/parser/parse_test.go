package parser_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/parser"
	thumbnailsdefaults "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/config/defaults"
	thumbnailsparser "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/config/parser"
)

var _ = Describe("the maximum size of a space image", func() {
	BeforeEach(func() {
		GinkgoT().Setenv("OC_CONFIG_DIR", GinkgoT().TempDir())
	})

	// only the environment, the way ParseConfig reads it: ParseConfig itself would
	// also ask for secrets
	decodeEnv := func() *config.Config {
		cfg := defaults.DefaultConfig()
		if err := envdecode.Decode(cfg); !errors.Is(err, envdecode.ErrNoTargetFieldsAreSet) {
			Expect(err).ToNot(HaveOccurred())
		}
		return cfg
	}

	// graph would otherwise accept space images no thumbnail is made of, or
	// refuse ones that would be shown
	DescribeTable("is read the way the thumbnails service reads its maximum input file size",
		func(env map[string]string, want string) {
			for k, v := range env {
				GinkgoT().Setenv(k, v)
			}

			cfg := decodeEnv()

			generator := thumbnailsdefaults.DefaultConfig()
			Expect(thumbnailsparser.ParseConfig(generator)).To(Succeed())

			Expect(cfg.Thumbnails.SpaceImageMaxInputImageFileSize).To(Equal(want))
			Expect(generator.Thumbnail.MaxInputImageFileSize).To(Equal(want))
		},
		Entry("from the shared name", map[string]string{
			"OC_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE": "100MB",
		}, "100MB"),
		Entry("from the old name", map[string]string{
			"THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE": "20MB",
		}, "20MB"),
		Entry("from the old name when both are set", map[string]string{
			"THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE":    "20MB",
			"OC_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE": "100MB",
		}, "20MB"),
	)

	It("can be set lower for graph alone", func() {
		GinkgoT().Setenv("OC_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE", "100MB")
		GinkgoT().Setenv("OC_SPACE_IMAGE_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE", "10MB")

		Expect(decodeEnv().Thumbnails.SpaceImageMaxInputImageFileSize).To(Equal("10MB"))
	})
})

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
