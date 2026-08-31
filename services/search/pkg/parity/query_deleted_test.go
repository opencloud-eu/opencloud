package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func deletedGroup() queryGroup {
	return queryGroup{
		name: "deleted",
		fixtures: []search.Resource{
			fixtureDoc("trashed.txt", isDeleted()),
			fixtureDoc("kept.txt"),
			fixtureFolder("bin", isDeleted()),
			fixtureDoc("receipt.txt", withParent("1$1!bin"), withPath("./bin/receipt.txt"), isDeleted()),
			fixtureFolder("shelf"),
			fixtureDoc("book.txt", withParent("1$1!shelf"), withPath("./shelf/book.txt")),
		},
		cases: []queryCase{
			{id: 1, query: `name:"*trashed*"`},
			{id: 2, query: `name:"*.txt"`, want: []string{"kept.txt", "book.txt"}},
			{id: 3, query: `name:"*receipt*"`},
			{id: 4, query: `path:"./bin"`},
			{id: 5, query: `path:"./shelf"`, want: []string{"shelf", "book.txt"}, engineOverrides: map[string]override{"bleve": override{want: []string{"shelf"}}}},
		},
	}
}
