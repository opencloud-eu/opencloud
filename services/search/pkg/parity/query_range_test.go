package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func rangeGroup() queryGroup {
	return queryGroup{
		name: "range",
		fixtures: []search.Resource{
			fixtureDoc("small.txt", withSize(50)),
			fixtureDoc("big.txt", withSize(500)),
			fixtureDoc("ancient.txt", withSize(10), withMtime("2020-01-01T00:00:00Z")),
		},
		cases: []queryCase{
			{id: 1, query: `size>100`, want: []string{"big.txt"}, engineOverrides: map[string]override{"bleve": override{}, "opensearch": override{}}},
			{id: 2, query: `size<100`, want: []string{"small.txt", "ancient.txt"}, engineOverrides: map[string]override{"bleve": override{}, "opensearch": override{}}},
			{id: 3, query: `mtime>"2021-01-01T00:00:00Z"`, want: []string{"small.txt", "big.txt"}},
			{id: 4, query: `mtime<"2021-01-01T00:00:00Z"`, want: []string{"ancient.txt"}},
			{id: 5, query: `Mtime:"today"`, want: []string{"small.txt", "big.txt"}},
			{id: 6, query: `Mtime:"yesterday"`},
			{id: 7, query: `mtime>2021`},
			{id: 8, query: `name>100`},
		},
	}
}
