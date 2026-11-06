package svc_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	svc "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
)

var _ = Describe("profile pictures", func() {
	DescribeTable("defaults photo downloads to the configured issuer origin", func(issuer, override string, allowed, denied string) {
		GinkgoT().Setenv("OC_URL", "https://cloud.example.com")
		GinkgoT().Setenv("OC_OIDC_ISSUER", issuer)
		GinkgoT().Setenv("GRAPH_OIDC_ISSUER", override)
		if override == "" {
			Expect(os.Unsetenv("GRAPH_OIDC_ISSUER")).To(Succeed())
		}
		cfg := config.Config{Commons: &shared.Commons{OpenCloudURL: "https://cloud.example.com"}}
		Expect(envdecode.Decode(&cfg)).To(Succeed())
		provider, err := svc.NewConfiguredUserProfilePhotoService(context.Background(), &cfg)
		Expect(err).To(Succeed())
		photoService := provider.(svc.UsersUserProfilePhotoService)
		Expect(svc.IsProfilePictureURLAllowed(photoService, allowed)).To(BeTrue())
		Expect(svc.IsProfilePictureURLAllowed(photoService, denied)).To(BeFalse())
	},
		Entry("external issuer without its realm path", "https://idp.example.com/realms/cloud", "", "https://idp.example.com/avatar.png", "https://cloud.example.com/avatar.png"),
		Entry("graph-specific issuer overrides the shared issuer", "https://idp.example.com", "https://other-idp.example.com/realm", "https://other-idp.example.com/avatar.png", "https://idp.example.com/avatar.png"),
		Entry("builtin issuer uses the OpenCloud origin", "", "", "https://cloud.example.com/avatar.png", "https://other.example.com/avatar.png"),
	)

	DescribeTable("verifies TLS unless picture downloads explicitly allow insecure connections", func(insecure bool) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
		defer server.Close()
		client := svc.NewProfilePictureHTTPClient(insecure)
		defer client.CloseIdleConnections()
		response, err := client.Get(server.URL)
		if insecure {
			Expect(err).To(Succeed())
			response.Body.Close()
		} else {
			Expect(err).To(HaveOccurred())
		}
		Expect(client.Timeout).To(Equal(10 * time.Second))
		transport := client.Transport.(*http.Transport)
		Expect(transport.TLSClientConfig.MinVersion).To(Equal(uint16(tls.VersionTLS12)))
		Expect(transport.Proxy).NotTo(BeNil())
		Expect(transport.DialContext).NotTo(BeNil())
	},
		Entry("secure by default", false),
		Entry("explicitly insecure", true),
	)

	It("does not follow picture download redirects", func() {
		reached := false
		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { reached = true }))
		defer target.Close()
		source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
		defer source.Close()
		client := svc.NewProfilePictureHTTPClient(false)
		defer client.CloseIdleConnections()
		response, err := client.Get(source.URL)
		Expect(err).To(Succeed())
		defer response.Body.Close()
		Expect(response.StatusCode).To(Equal(http.StatusFound))
		Expect(reached).To(BeFalse())
	})

	It("keeps picture TLS configuration independent from event TLS", func() {
		GinkgoT().Setenv("OC_INSECURE", "false")
		GinkgoT().Setenv("GRAPH_EVENTS_TLS_INSECURE", "true")
		var cfg config.Config
		Expect(envdecode.Decode(&cfg)).To(Succeed())
		Expect(cfg.Events.TLSInsecure).To(BeTrue())
		Expect(cfg.ProfilePictureInsecure).To(BeFalse())
		GinkgoT().Setenv("GRAPH_PROFILE_PICTURE_INSECURE", "true")
		Expect(envdecode.Decode(&cfg)).To(Succeed())
		Expect(cfg.ProfilePictureInsecure).To(BeTrue())
		GinkgoT().Setenv("OC_INSECURE", "true")
		GinkgoT().Setenv("GRAPH_PROFILE_PICTURE_INSECURE", "false")
		Expect(envdecode.Decode(&cfg)).To(Succeed())
		Expect(cfg.ProfilePictureInsecure).To(BeFalse())
	})

})
