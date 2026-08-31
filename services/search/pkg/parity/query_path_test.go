package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func pathGroup() queryGroup {
	return queryGroup{
		name: "path",
		fixtures: []search.Resource{
			fixtureFolder("parent"),
			fixtureDoc("child.jpg", withMime("image/jpeg"), withPath("./parent/child.jpg")),
			fixtureFolder("docs-lower", withPath("./documents")),
			fixtureFolder("docs-upper", withPath("./DOCUMENTS")),
			fixtureFolder("docs-mixed", withPath("./Documents")),
		},
		cases: []queryCase{
			{id: 1, query: `path:"./parent"`, want: []string{"parent", "child.jpg"}, engineOverrides: map[string]override{"bleve": override{want: []string{"parent"}}}},
			{id: 2, query: `path:"./parent/child.jpg"`, want: []string{"child.jpg"}},
			{id: 3, query: `path:"./Parent"`, engineOverrides: map[string]override{"opensearch": override{want: []string{"child.jpg", "parent"}}}},
			{id: 4, query: `path:"*child*"`, want: []string{"child.jpg"}},
			{id: 5, query: `path:"./documents"`, want: []string{"docs-lower"}, engineOverrides: map[string]override{"opensearch": override{want: []string{"docs-lower", "docs-mixed", "docs-upper"}}}},
			{id: 6, query: `path:"./DOCUMENTS"`, want: []string{"docs-upper"}, engineOverrides: map[string]override{"opensearch": override{want: []string{"docs-lower", "docs-mixed", "docs-upper"}}}},
			{id: 7, query: `path:"./Documents"`, want: []string{"docs-mixed"}, engineOverrides: map[string]override{"opensearch": override{want: []string{"docs-lower", "docs-mixed", "docs-upper"}}}},
			{id: 8, query: `path:"./parent/"`, want: []string{"parent", "child.jpg"}, engineOverrides: map[string]override{"bleve": override{}, "opensearch": override{}}},
		},
	}
}
