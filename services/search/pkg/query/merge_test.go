package query_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/services/search/pkg/query"
)

func collectStringNodes(nodes []ast.Node) []*ast.StringNode {
	var out []*ast.StringNode
	for _, n := range nodes {
		switch node := n.(type) {
		case *ast.StringNode:
			out = append(out, node)
		case *ast.GroupNode:
			out = append(out, collectStringNodes(node.Nodes)...)
		}
	}
	return out
}

var _ = Describe("MergeFilters", func() {
	It("returns the normalized main query unchanged when there are no filters", func() {
		a, err := query.MergeFilters(kql.Builder{}, `name:"hello"`, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(a.Nodes).To(HaveLen(1))
	})

	It("ANDs a filter in as an exact, case-sensitive match", func() {
		a, err := query.MergeFilters(kql.Builder{}, `name:"hello"`, []string{`Tags:"Pink Floyd"`})
		Expect(err).ToNot(HaveOccurred())

		strs := collectStringNodes(a.Nodes)
		var forced *ast.StringNode
		for _, s := range strs {
			if s.Value == "Pink Floyd" {
				forced = s
			}
		}
		Expect(forced).ToNot(BeNil(), "the decoded filter node should be present")
		Expect(forced.Exact).To(BeTrue())
		Expect(forced.CaseInsensitive).To(BeFalse())
	})

	It("forces every node of an OR filter", func() {
		a, err := query.MergeFilters(kql.Builder{}, `name:"hello"`, []string{`(Tags:"a" OR Tags:"b")`})
		Expect(err).ToNot(HaveOccurred())
		forced := 0
		for _, s := range collectStringNodes(a.Nodes) {
			if s.Value != "a" && s.Value != "b" {
				continue
			}
			forced++
			Expect(s.Exact).To(BeTrue())
			Expect(s.CaseInsensitive).To(BeFalse())
		}
		Expect(forced).To(Equal(2))
	})
})
