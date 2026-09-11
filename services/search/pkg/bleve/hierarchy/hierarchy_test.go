package hierarchy_test

import (
	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/registry"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve/hierarchy"
)

func terms(ts analysis.TokenStream) []string {
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		out = append(out, string(t.Term))
	}
	return out
}

func tokenize(config map[string]any, input string) []string {
	tok, err := hierarchy.Constructor(config, registry.NewCache())
	Expect(err).ToNot(HaveOccurred())
	return terms(tok.Tokenize([]byte(input)))
}

var _ = Describe("hierarchy tokenizer", func() {
	path := map[string]any{"delimiter": "/"}
	geohash := map[string]any{"tag_depth": true}

	DescribeTable("emits every prefix up to a level boundary",
		func(config map[string]any, input string, want []string) {
			Expect(tokenize(config, input)).To(Equal(want))
		},
		Entry("relative path", path, "./a/b.txt", []string{".", "./a", "./a/b.txt"}),
		Entry("space root", path, ".", []string{"."}),
		Entry("trailing delimiter is not a level", path, "./a/", []string{".", "./a"}),
		Entry("delimiter only", path, "/", []string{}),
		Entry("leading delimiter", path, "/abs/x", []string{"/abs", "/abs/x"}),
		Entry("double delimiter", path, "./a//b", []string{".", "./a", "./a//b"}),
		Entry("spaces and special characters stay literal", path, "./odd name*[1]/f:x?.txt",
			[]string{".", "./odd name*[1]", "./odd name*[1]/f:x?.txt"}),
		Entry("empty input", path, "", []string{}),
		Entry("geohash, one level per byte, depth tagged", geohash, "u4pru",
			[]string{"1/u", "2/u4", "3/u4p", "4/u4pr", "5/u4pru"}),
	)

	It("keeps byte offsets on the source value", func() {
		tok, err := hierarchy.Constructor(path, registry.NewCache())
		Expect(err).ToNot(HaveOccurred())
		ts := tok.Tokenize([]byte("./a/b"))
		Expect(ts).To(HaveLen(3))
		Expect(ts[2].Start).To(Equal(0))
		Expect(ts[2].End).To(Equal(5))
		Expect(ts[2].Position).To(Equal(3))
	})
})
