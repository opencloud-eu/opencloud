package parity

import (
	"context"
	"fmt"
	"strings"

	bleveSearch "github.com/blevesearch/bleve/v2"
	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/internal/opensearchtest"
	bleveEngine "github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	bleveQuery "github.com/opencloud-eu/opencloud/services/search/pkg/query/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var engineNames = []string{"bleve", "opensearch"}

type testEngine struct {
	name        string
	backend     search.Engine
	settle      func()
	unavailable string
}

// newEngines builds one backend per engine over the same fixtures. Call it
// from a BeforeAll: the cleanups it defers run when the container ends.
func newEngines(index string, fixtures []search.Resource) []testEngine {
	GinkgoHelper()

	return []testEngine{
		newBleve(fixtures),
		newOpenSearch(index, fixtures),
	}
}

// newEngine builds just the one engine a spec is about; its cleanup runs when
// that spec ends.
func newEngine(name, index string, fixtures []search.Resource) testEngine {
	GinkgoHelper()

	switch name {
	case "bleve":
		return newBleve(fixtures)
	case "opensearch":
		return newOpenSearch(index, fixtures)
	}

	Fail("no engine named " + name)
	return testEngine{}
}

func engineNamed(engines []testEngine, name string) testEngine {
	GinkgoHelper()

	for _, e := range engines {
		if e.name == name {
			return e
		}
	}

	Fail("no engine named " + name)
	return testEngine{}
}

func newBleve(fixtures []search.Resource) testEngine {
	GinkgoHelper()

	mapping, err := bleveEngine.NewMapping()
	Expect(err).NotTo(HaveOccurred(), "failed to build the bleve mapping")

	idx, err := bleveSearch.NewMemOnly(mapping)
	Expect(err).NotTo(HaveOccurred(), "failed to create the bleve index")
	DeferCleanup(func() { _ = idx.Close() })

	backend := bleveEngine.NewBackend(idx, bleveQuery.DefaultCreator, log.Logger{})
	load(backend, fixtures)

	return testEngine{name: "bleve", backend: backend, settle: func() {}}
}

func newOpenSearch(index string, fixtures []search.Resource) testEngine {
	GinkgoHelper()

	tc := opensearchtest.NewDefaultTestClient(GinkgoTB(), openSearchClient)

	if err := tc.IndicesReset(context.Background(), []string{index}); err != nil {
		return testEngine{name: "opensearch", unavailable: err.Error()}
	}

	backend, err := opensearch.NewBackend(index, tc.Client())
	if err != nil {
		return testEngine{name: "opensearch", unavailable: err.Error()}
	}

	DeferCleanup(func() {
		Expect(tc.IndicesDelete(context.Background(), []string{index})).To(Succeed())
	})

	// every write waits for the next refresh (Refresh: wait_for), which is a
	// second apart by default; the test index can refresh far more often
	_, err = tc.Client().Indices.Settings.Put(context.Background(), opensearchgoAPI.SettingsPutReq{
		Indices: []string{index},
		Body:    strings.NewReader(`{"index": {"refresh_interval": "50ms"}}`),
	})
	Expect(err).NotTo(HaveOccurred(), "failed to speed up the refresh of %s", index)

	settle := func() { tc.Require.IndicesRefresh([]string{index}, nil) }
	load(backend, fixtures)
	settle()

	return testEngine{name: "opensearch", backend: backend, settle: settle}
}

// load writes the fixtures the way the service does, in one batch.
func load(backend search.Engine, fixtures []search.Resource) {
	GinkgoHelper()

	if len(fixtures) == 0 {
		return
	}

	batch, err := backend.NewBatch(len(fixtures) + 1)
	Expect(err).NotTo(HaveOccurred(), "failed to open a batch for the fixtures")
	for _, doc := range fixtures {
		Expect(batch.Upsert(doc.ID, doc)).To(Succeed(), "upsert %s", doc.ID)
	}
	Expect(batch.Push()).To(Succeed(), "failed to write the fixtures")
}

func badRequestAnswer(err error) []string {
	if err == nil {
		return []string{"no error"}
	}

	return []string{"bad request"}
}

func reads(read func(*searchMessage.Match) string) func(*searchService.SearchIndexResponse) []string {
	return readsMany(func(m *searchMessage.Match) []string { return []string{read(m)} })
}

func readsMany(read func(*searchMessage.Match) []string) func(*searchService.SearchIndexResponse) []string {
	return func(resp *searchService.SearchIndexResponse) []string {
		if len(resp.Matches) != 1 {
			return []string{fmt.Sprintf("%d matches", len(resp.Matches))}
		}

		return read(resp.Matches[0])
	}
}

func resourceID(id *searchMessage.ResourceID) string {
	return storagespace.FormatResourceID(&sprovider.ResourceId{
		StorageId: id.GetStorageId(),
		SpaceId:   id.GetSpaceId(),
		OpaqueId:  id.GetOpaqueId(),
	})
}

// ask runs a query and returns the matched names; an engine error is the
// answer "error", so a query the engine rejects still shows in the matrix.
func ask(e search.Engine, request *searchService.SearchIndexRequest) ([]string, error) {
	resp, err := e.Search(context.Background(), request)
	if err != nil {
		return []string{"error"}, err
	}

	names := make([]string, 0, len(resp.Matches))
	for _, m := range resp.Matches {
		names = append(names, m.Entity.GetName())
	}

	return names, nil
}

// matchNames is ConsistOf for a wanted list, spelled out for the empty case so
// a "no match" expectation reads as such in the failure.
func matchNames(want []string) types.GomegaMatcher {
	if len(want) == 0 {
		return BeEmpty()
	}

	return ConsistOf(want)
}
