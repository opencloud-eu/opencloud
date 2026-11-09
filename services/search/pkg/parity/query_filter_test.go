package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func filterGroup() queryGroup {
	return queryGroup{
		name: "filter",
		fixtures: []search.Resource{
			fixtureFolder("alice", withID("1$1!1"), withPath(".")),
			fixtureFolder(".space", withPath("./.space")),
			fixtureDoc("readme.md", withPath("./.space/readme.md")),
			fixtureFolder(".spaces", withPath("./.spaces")),
			fixtureDoc("notes.md"),
		},
		cases: []queryCase{
			{id: 1, query: `path:"./.space"`, want: []string{".space", "readme.md"}},
			{id: 2, query: `(name:"*.md") AND (NOT path:"./.space" AND NOT id:"none$none!none")`, want: []string{"notes.md"}},
			{id: 3, query: `(name:"alice" OR name:".spaces") AND (NOT path:"./.space" AND NOT id:"1$1!1")`, want: []string{".spaces"}},
			{id: 4, query: `(name:"alice") AND (NOT path:"./.space" AND NOT id:"none$none!none")`, want: []string{"alice"}},
		},
	}
}
