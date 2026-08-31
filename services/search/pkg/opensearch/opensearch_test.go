package opensearch_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	opensearchtest "github.com/opencloud-eu/opencloud/services/search/internal/opensearchtest"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
)

var defaultConfig *config.Config

func TestMain(m *testing.M) {
	cfg, done, err := opensearchtest.SetupTests(context.Background())
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to setup tests:", err)
		os.Exit(1)
		return
	}
	defaultConfig = cfg
	code := m.Run()
	done()
	os.Exit(code)
}
