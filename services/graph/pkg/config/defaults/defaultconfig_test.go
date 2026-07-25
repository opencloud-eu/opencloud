package defaults

import (
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// TestEnsureDefaultsDerivesRootUnderSubpath guards against a regression
// where the OC_URL-derived subpath was never prepended to a service's own,
// non-empty default HTTP.Root (here "/graph") -- see the identical test in
// services/proxy for the "/" case. Without the prepend, graph's own mux
// stayed gated at "/graph" while the proxy forwarded the full, prefixed
// path (e.g. "/test/opencloud/graph/..."), so nothing matched under a
// subpath deployment.
func TestEnsureDefaultsDerivesRootUnderSubpath(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.HTTP.Root != "/graph" {
		t.Fatalf("test assumes DefaultConfig's baseline HTTP.Root is \"/graph\", got %q -- update this test", cfg.HTTP.Root)
	}

	cfg.Commons = &shared.Commons{OpenCloudURL: "https://host.example/test/opencloud"}

	EnsureDefaults(cfg)

	if want := "/test/opencloud/graph"; cfg.HTTP.Root != want {
		t.Errorf("HTTP.Root = %q, want %q", cfg.HTTP.Root, want)
	}

	// EnsureDefaults runs more than once against the same *config.Config in
	// practice; it must not prepend the subpath a second time.
	EnsureDefaults(cfg)
	if want := "/test/opencloud/graph"; cfg.HTTP.Root != want {
		t.Errorf("HTTP.Root after a second EnsureDefaults call = %q, want %q (derivation should be idempotent)", cfg.HTTP.Root, want)
	}
}
