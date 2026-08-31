package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func extensionGroup() queryGroup {
	return queryGroup{
		name: "extension",
		fixtures: []search.Resource{
			fixtureDoc("report.txt"),
			fixtureDoc("notes.md", withMime("text/markdown")),
			fixtureFolder("archive"),
		},
		cases: []queryCase{
			{id: 1, query: `txt`, want: []string{"report.txt"}},
			{id: 2, query: `md`, want: []string{"notes.md"}},
			{id: 3, query: `name:"*.txt"`, want: []string{"report.txt"}},
			{id: 4, query: `report`, want: []string{"report.txt"}},
		},
	}
}
