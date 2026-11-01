package svc

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
)

var _ = ginkgo.It("verifies TLS unless picture downloads explicitly allow insecure connections", func() {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer server.Close()
	for _, insecure := range []bool{false, true} {
		client := newProfilePictureHTTPClient(insecure)
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
	}
})

var _ = ginkgo.It("does not follow picture download redirects", func() {
	reached := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { reached = true }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer source.Close()
	client := newProfilePictureHTTPClient(false)
	defer client.CloseIdleConnections()
	response, err := client.Get(source.URL)
	Expect(err).To(Succeed())
	defer response.Body.Close()
	Expect(response.StatusCode).To(Equal(http.StatusFound))
	Expect(reached).To(BeFalse())
})

var _ = ginkgo.It("keeps picture TLS configuration independent from event TLS", func() {
	ginkgo.GinkgoT().Setenv("OC_INSECURE", "false")
	ginkgo.GinkgoT().Setenv("GRAPH_EVENTS_TLS_INSECURE", "true")
	var cfg config.Config
	Expect(envdecode.Decode(&cfg)).To(Succeed())
	Expect(cfg.Events.TLSInsecure).To(BeTrue())
	Expect(cfg.ProfilePictureInsecure).To(BeFalse())
	ginkgo.GinkgoT().Setenv("GRAPH_PROFILE_PICTURE_INSECURE", "true")
	Expect(envdecode.Decode(&cfg)).To(Succeed())
	Expect(cfg.ProfilePictureInsecure).To(BeTrue())
	ginkgo.GinkgoT().Setenv("OC_INSECURE", "true")
	ginkgo.GinkgoT().Setenv("GRAPH_PROFILE_PICTURE_INSECURE", "false")
	Expect(envdecode.Decode(&cfg)).To(Succeed())
	Expect(cfg.ProfilePictureInsecure).To(BeFalse())
})
