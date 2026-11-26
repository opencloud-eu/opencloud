package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestSecurityStrictTransportSecurity(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		headers map[string]string
		want    string
	}{
		{
			name: "https request gets the full header including subdomains",
			url:  "https://cloud.example.com/",
			want: "max-age=315360000; includeSubDomains; preload",
		},
		{
			name: "plain http request gets no header",
			url:  "http://cloud.example.com/",
			want: "",
		},
		{
			name:    "plain http request with X-Forwarded-Proto gets no header",
			url:     "http://cloud.example.com/",
			headers: map[string]string{"X-Forwarded-Proto": "https"},
			want:    "",
		},
	}

	cspConfig := &config.CSP{Directives: map[string][]string{"default-src": {"'none'"}}}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := Security(cspConfig)(next)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, rec.Code, http.StatusOK)
			assert.Equal(t, rec.Header().Get("Strict-Transport-Security"), tt.want)
		})
	}
}

func TestLoadCSPConfig(t *testing.T) {
	// setup test env
	presetYaml := `
directives:
  frame-src:
    - '''self'''
    - 'https://embed.diagrams.net/'
    - 'https://${ONLYOFFICE_DOMAIN|onlyoffice.opencloud.test}/'
    - 'https://${COLLABORA_DOMAIN|collabora.opencloud.test}/'
`

	customYaml := `
directives:
  img-src:
    - '''self'''
    - 'data:'
  frame-src:
    - 'https://some.custom.domain/'
`
	config, err := loadCSPConfig([]byte(presetYaml), []byte(customYaml))
	if err != nil {
		t.Error(err)
	}
	assert.Assert(t, cmp.Contains(config.Directives["frame-src"], "'self'"))
	assert.Assert(t, cmp.Contains(config.Directives["frame-src"], "https://embed.diagrams.net/"))
	assert.Assert(t, cmp.Contains(config.Directives["frame-src"], "https://onlyoffice.opencloud.test/"))
	assert.Assert(t, cmp.Contains(config.Directives["frame-src"], "https://collabora.opencloud.test/"))

	assert.Assert(t, cmp.Contains(config.Directives["img-src"], "'self'"))
	assert.Assert(t, cmp.Contains(config.Directives["img-src"], "data:"))
}
