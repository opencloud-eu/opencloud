package defaults

import (
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// TestEnsureDefaultsDerivesRootUnderSubpath guards against a regression where
// EnsureDefaults's derivation of HTTP.Root from OC_URL never fired, because
// DefaultConfig sets HTTP.Root to "/" as its own baseline default -- a
// "cfg.HTTP.Root == ''" unset check is never true, so the derivation was
// silently skipped and the deployment's subpath was dropped entirely.
// Only caught by actually deploying under a real subpath and observing every
// request fall through to the web app's catch-all route.
func TestEnsureDefaultsDerivesRootUnderSubpath(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.HTTP.Root != "/" {
		t.Fatalf("test assumes DefaultConfig's baseline HTTP.Root is \"/\", got %q -- update this test", cfg.HTTP.Root)
	}

	cfg.Commons = &shared.Commons{OpenCloudURL: "https://host.example/test/opencloud"}

	EnsureDefaults(cfg)

	if want := "/test/opencloud"; cfg.HTTP.Root != want {
		t.Errorf("HTTP.Root = %q, want %q", cfg.HTTP.Root, want)
	}

	// EnsureDefaults runs more than once against the same *config.Config in
	// practice; it must not prepend the subpath a second time.
	EnsureDefaults(cfg)
	if want := "/test/opencloud"; cfg.HTTP.Root != want {
		t.Errorf("HTTP.Root after a second EnsureDefaults call = %q, want %q (derivation should be idempotent)", cfg.HTTP.Root, want)
	}
}

// TestEnsureDefaultsRootOfDomainUnaffected covers the common case: no
// subpath in OC_URL, HTTP.Root should stay at its "/" baseline.
func TestEnsureDefaultsRootOfDomainUnaffected(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Commons = &shared.Commons{OpenCloudURL: "https://host.example"}

	EnsureDefaults(cfg)

	if cfg.HTTP.Root != "/" {
		t.Errorf("HTTP.Root = %q, want \"/\" (no subpath in OC_URL)", cfg.HTTP.Root)
	}
}
