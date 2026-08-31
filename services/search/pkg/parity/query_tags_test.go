package parity

import (
	"strings"

	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func tagsGroup() queryGroup {
	longTag := strings.Repeat("z", 300) + "needle"

	return queryGroup{
		name: "tags",
		fixtures: []search.Resource{
			fixtureDoc("invoice.txt", withTags("foo-bar")),
			fixtureDoc("memo.txt", withTags("foo")),
			fixtureDoc("spaced.txt", withTags("spaced tag")),
			fixtureFolder("project", withTags("work")),
			fixtureDoc("draft.txt", withParent("1$1!project"), withPath("./project/draft.txt")),
			fixtureDoc("longtag.txt", withTags(longTag)),
		},
		cases: []queryCase{
			{id: 1, query: `name:"*foo-bar*"`},
			{id: 2, query: `tag:("foo-bar")`, want: []string{"invoice.txt"}},
			{id: 3, query: `tag:("foo")`, want: []string{"memo.txt"}},
			{id: 4, query: `tag:("FOO-BAR")`, want: []string{"invoice.txt"}},
			{id: 5, query: `tag:("*foo*")`, want: []string{"invoice.txt", "memo.txt"}},
			{id: 6, query: `tag:("spaced tag")`, want: []string{"spaced.txt"}},
			{id: 7, query: `tag:("*paced ta*")`, want: []string{"spaced.txt"}},
			{id: 8, query: `tag:("work")`, want: []string{"project"}},
			{id: 9, query: `tag:("` + longTag + `")`, want: []string{"longtag.txt"}, engineOverrides: map[string]override{"opensearch": override{}}},
		},
	}
}
