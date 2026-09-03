//go:build vectors

package bleve_test

import (
	"context"
	"fmt"
	"os"
	"strings"

	bleveSearch "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/index/scorch"
	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	bleveQuery "github.com/opencloud-eu/opencloud/services/search/pkg/query/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// axisVector returns a one-hot vector of the schema dimensionality: distinct
// axes are orthogonal (cosine 0), same axes identical (cosine 1).
func axisVector(axis int) []float32 {
	v := make([]float32, content.ImageVectorDims)
	v[axis] = 1
	return v
}

// fakeVectorizer maps query texts to fixed vectors.
type fakeVectorizer struct{ byText map[string][]float32 }

func (f fakeVectorizer) VectorizeText(_ context.Context, text string) ([]float32, error) {
	v, ok := f.byText[strings.ToLower(text)]
	if !ok {
		return nil, fmt.Errorf("no vector for %q", text)
	}
	return v, nil
}

var _ = Describe("Semantic search (vectors build)", func() {
	var (
		eng *bleve.Backend

		doSearch = func(query string) *searchsvc.SearchIndexResponse {
			res, err := eng.Search(context.Background(), &searchsvc.SearchIndexRequest{
				Query: query,
				Ref: &searchmsg.Reference{
					ResourceId: &searchmsg.ResourceID{StorageId: "1", SpaceId: "2", OpaqueId: "2"},
				},
			})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			return res
		}

		names = func(res *searchsvc.SearchIndexResponse) []string {
			out := make([]string, 0, len(res.Matches))
			for _, m := range res.Matches {
				out = append(out, m.GetEntity().GetName())
			}
			return out
		}
	)

	BeforeEach(func() {
		mapping, err := bleve.NewMapping()
		Expect(err).ToNot(HaveOccurred())

		tmpDir, err := os.MkdirTemp("", "bleve-semantic-test-")
		Expect(err).ToNot(HaveOccurred())
		idx, err := bleveSearch.NewUsing(tmpDir, mapping, scorch.Name, bleveSearch.Config.DefaultKVStore, nil)
		Expect(err).ToNot(HaveOccurred())
		// close before removing: the background persister still writes into
		// the store directory otherwise
		DeferCleanup(func() error {
			if err := idx.Close(); err != nil {
				return err
			}
			return os.RemoveAll(tmpDir)
		})

		eng = bleve.NewBackend(idx, bleveQuery.DefaultCreator, log.Logger{}, bleve.WithTextVectorizer(fakeVectorizer{
			byText: map[string][]float32{
				"cat": axisVector(0),
				"dog": axisVector(1),
			},
		}))

		upsert := func(id, path, name string, vector []float32) {
			r := search.Resource{
				ID:       id,
				RootID:   "1$2!2",
				ParentID: "1$2!2",
				Path:     path,
				Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE),
				Document: content.Document{Name: name, MimeType: "image/jpeg", ImageVector: vector},
			}
			Expect(eng.Upsert(id, r)).To(Succeed())
		}
		upsert("1$2!10", "./cat.jpg", "cat.jpg", axisVector(0))
		upsert("1$2!11", "./dog.jpg", "dog.jpg", axisVector(1))
		Expect(eng.Upsert("1$2!12", search.Resource{
			ID: "1$2!12", RootID: "1$2!2", ParentID: "1$2!2", Path: "./note.txt",
			Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE),
			Document: content.Document{Name: "note.txt", MimeType: "text/plain"},
		})).To(Succeed())
	})

	It("ranks the nearest image first on a purely semantic query", func() {
		res := doSearch(`semantic:"cat"`)
		Expect(len(res.Matches)).To(BeNumerically(">=", 1))
		Expect(res.Matches[0].GetEntity().GetName()).To(Equal("cat.jpg"))
	})

	It("combines the semantic clause with a filter part", func() {
		res := doSearch(`Name:dog* AND semantic:"cat"`)
		Expect(names(res)).To(Equal([]string{"dog.jpg"}))
	})

	It("keeps the vector across move and restore round-trips", func() {
		Expect(eng.Move("1$2!10", "1$2!2", "./moved-cat.jpg")).To(Succeed())
		res := doSearch(`semantic:"cat"`)
		Expect(res.Matches[0].GetEntity().GetName()).To(Equal("moved-cat.jpg"))

		Expect(eng.Delete("1$2!10")).To(Succeed())
		res = doSearch(`semantic:"cat"`)
		for _, n := range names(res) {
			Expect(n).ToNot(Equal("moved-cat.jpg"))
		}

		Expect(eng.Restore("1$2!10")).To(Succeed())
		res = doSearch(`semantic:"cat"`)
		Expect(res.Matches[0].GetEntity().GetName()).To(Equal("moved-cat.jpg"))
	})

	It("rejects semantic queries without a vectorizer", func() {
		bare := bleve.NewBackend(nil, bleveQuery.DefaultCreator, log.Logger{})
		_, err := bare.Search(context.Background(), &searchsvc.SearchIndexRequest{Query: `semantic:"cat"`})
		Expect(err).To(MatchError(ContainSubstring("not configured")))
	})
})
