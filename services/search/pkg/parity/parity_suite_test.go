package parity

import (
	"context"
	"encoding/json"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/search/internal/opensearchtest"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
)

var (
	// openSearchClient points every process at the one OpenSearch started by
	// process 1; the container handle stays there.
	openSearchClient config.EngineOpenSearchClient
	stopOpenSearch   func()
)

func TestEngineParity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Search Engine Parity Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	cfg, done, err := opensearchtest.SetupTests(context.Background())
	Expect(err).NotTo(HaveOccurred(), "failed to set up the OpenSearch test container")
	stopOpenSearch = done

	client, err := json.Marshal(cfg.Engine.OpenSearch.Client)
	Expect(err).NotTo(HaveOccurred())

	return client
}, func(client []byte) {
	Expect(json.Unmarshal(client, &openSearchClient)).To(Succeed())
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	if stopOpenSearch != nil {
		stopOpenSearch()
	}
})

var _ = ReportAfterSuite("engine parity matrix", func(report Report) {
	writeMatrix(report)
})
