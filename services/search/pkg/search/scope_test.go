package search_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var _ = DescribeTable("ExtractScope",
	func(query, wantQuery, wantScope string) {
		nodes, scope, err := search.ExtractScope(parse(query).Nodes)
		Expect(err).ToNot(HaveOccurred())
		Expect(scope).To(Equal(wantScope))
		if wantQuery == "" {
			Expect(nodes).To(BeEmpty())
			return
		}
		Expect(render(nodes)).To(Equal(render(parse(wantQuery).Nodes)))
	},
	Entry("at the end", `name:"*file*" AND tag:foo AND scope:1$2!3`, `name:"*file*" AND tag:foo`, `1$2!3`),
	Entry("at the start", `scope:1$2!3 AND name:x`, `name:x`, `1$2!3`),
	Entry("in the middle", `name:x AND scope:1$2!3 AND tag:y`, `name:x AND tag:y`, `1$2!3`),
	Entry("joined implicitly", `name:x scope:1$2!3`, `name:x`, `1$2!3`),
	Entry("joined with OR", `name:x OR scope:1$2!3`, `name:x`, `1$2!3`),
	Entry("negated", `name:x AND NOT scope:1$2!3`, `name:x`, `1$2!3`),
	Entry("negated at the start", `NOT scope:1$2!3 AND name:x`, `name:x`, `1$2!3`),
	Entry("alone", `scope:1$2!3`, ``, `1$2!3`),
	Entry("no scope", `name:x AND tag:y`, `name:x AND tag:y`, ``),
	Entry("a quoted value is not a scope", `name:"scope:1$2!3"`, `name:"scope:1$2!3"`, ``),
)

var _ = DescribeTable("ExtractScope rejects a nested scope",
	func(query string) {
		_, _, err := search.ExtractScope(parse(query).Nodes)
		Expect(err).To(MatchError(search.ErrNestedScope))
	},
	Entry("inside a group", `name:x AND (scope:1$2!3 OR tag:y)`),
	Entry("inside a nested group", `name:x AND (tag:y OR (scope:1$2!3))`),
	Entry("after a top-level scope", `scope:1$2!3 AND (scope:1$2!4)`),
)

var _ = DescribeTable("ExpandFavorites",
	func(query, want string) {
		a := parse(query)
		search.ExpandFavorites(a.Nodes, "uid")
		Expect(render(a.Nodes)).To(Equal(render(parse(want).Nodes)))
	},
	Entry("alone", `is:favorite`, `Favorites:uid`),
	Entry("at the start", `is:favorite AND name:x`, `Favorites:uid AND name:x`),
	Entry("in the middle", `name:x AND is:favorite AND tag:y`, `name:x AND Favorites:uid AND tag:y`),
	Entry("negated", `NOT is:favorite`, `NOT Favorites:uid`),
	Entry("inside a group", `name:x AND (is:favorite OR tag:y)`, `name:x AND (Favorites:uid OR tag:y)`),
	Entry("a quoted value stays text", `name:"is:favorite"`, `name:"is:favorite"`),
)
