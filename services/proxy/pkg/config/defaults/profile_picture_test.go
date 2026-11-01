package defaults

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
)

var _ = It("configures the picture claim independently from account provisioning", func() {
	cfg := DefaultConfig()
	Expect(cfg.AutoprovisionAccounts).To(BeFalse())
	Expect(cfg.OIDC.PictureClaim).To(Equal("picture"))
	GinkgoT().Setenv("PROXY_OIDC_PICTURE_CLAIM", "avatar")
	Expect(envdecode.Decode(cfg)).To(Succeed())
	Expect(cfg.OIDC.PictureClaim).To(Equal("avatar"))
	GinkgoT().Setenv("PROXY_OIDC_PICTURE_CLAIM", "")
	Expect(envdecode.Decode(cfg)).To(Succeed())
	Expect(cfg.OIDC.PictureClaim).To(BeEmpty())
})
