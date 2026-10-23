package http

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestProfilePictureClientTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer server.Close()
	for _, insecure := range []bool{false, true} {
		client := newProfilePictureHTTPClient(insecure)
		defer client.CloseIdleConnections()
		response, err := client.Get(server.URL)
		if insecure {
			require.NoError(t, err)
			response.Body.Close()
		} else {
			require.Error(t, err)
		}
		require.Equal(t, 10*time.Second, client.Timeout)
		transport := client.Transport.(*http.Transport)
		require.Equal(t, uint16(tls.VersionTLS12), transport.TLSClientConfig.MinVersion)
		require.NotNil(t, transport.Proxy)
		require.NotNil(t, transport.DialContext)
	}
}

func TestProfilePictureClientDoesNotFollowRedirects(t *testing.T) {
	reached := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { reached = true }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer source.Close()
	client := newProfilePictureHTTPClient(false)
	defer client.CloseIdleConnections()
	response, err := client.Get(source.URL)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusFound, response.StatusCode)
	require.False(t, reached)
}

func TestProfilePictureTLSConfiguration(t *testing.T) {
	t.Setenv("OC_INSECURE", "false")
	t.Setenv("GRAPH_EVENTS_TLS_INSECURE", "true")
	var cfg config.Config
	require.NoError(t, envdecode.Decode(&cfg))
	require.True(t, cfg.Events.TLSInsecure)
	require.False(t, cfg.ProfilePictureInsecure)
	t.Setenv("GRAPH_PROFILE_PICTURE_INSECURE", "true")
	require.NoError(t, envdecode.Decode(&cfg))
	require.True(t, cfg.ProfilePictureInsecure)
	t.Setenv("OC_INSECURE", "true")
	t.Setenv("GRAPH_PROFILE_PICTURE_INSECURE", "false")
	require.NoError(t, envdecode.Decode(&cfg))
	require.False(t, cfg.ProfilePictureInsecure)
}
