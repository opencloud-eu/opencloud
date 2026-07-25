package svc

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/idp/pkg/config"
)

// TestIndexTemplateSubstitution guards against a regression where each
// bytes.Replace call in Index() re-read from the original template instead
// of chaining off the previous result, silently discarding earlier
// replacements (only the last one, __PASSWORD_RESET_LINK__, would ever
// survive). It also covers that __PATH_PREFIX__ includes IDP_URI_BASE_PATH
// when OpenCloud is deployed under a subpath.
func TestIndexTemplateSubstitution(t *testing.T) {
	const tmpl = `<div id="root" data-path-prefix="__PATH_PREFIX__" ` +
		`passwort-reset-link="__PASSWORD_RESET_LINK__" ` +
		`style="background-image: url(__BG_IMG_URL__)"></div>` +
		`<meta property="csp-nonce" content="__CSP_NONCE__">`

	fsys := fstest.MapFS{
		"identifier/index.html": &fstest.MapFile{Data: []byte(tmpl)},
	}

	idp := IDP{
		logger: log.NewLogger(),
		assets: http.FS(fsys),
		config: &config.Config{
			IDP: config.Settings{URIBasePath: "/test/opencloud"},
			Asset: config.Asset{
				LoginBackgroundUrl: "https://example.com/bg.png",
			},
			Service: config.Service{
				PasswordResetURI: "https://example.com/reset",
			},
		},
	}

	w := httptest.NewRecorder()
	idp.Index().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	body, err := io.ReadAll(w.Result().Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}
	got := string(body)

	for placeholder, want := range map[string]string{
		"__PATH_PREFIX__":         "/test/opencloud/signin/v1",
		"__BG_IMG_URL__":          "https://example.com/bg.png",
		"__PASSWORD_RESET_LINK__": "https://example.com/reset",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected substituted value %q (for %s) in response, got: %s", want, placeholder, got)
		}
		if strings.Contains(got, placeholder) {
			t.Errorf("placeholder %s was left unsubstituted in response: %s", placeholder, got)
		}
	}
	if strings.Contains(got, "__CSP_NONCE__") {
		t.Errorf("placeholder __CSP_NONCE__ was left unsubstituted in response: %s", got)
	}
}
