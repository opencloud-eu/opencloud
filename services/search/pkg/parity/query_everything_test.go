package parity

import (
	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func everythingGroup() queryGroup {
	return queryGroup{
		name: "everything",
		fixtures: []search.Resource{
			fixtureDoc("alpha.txt"),
			fixtureDoc("beta.txt"),
			fixtureFolder("box"),
		},
		cases: []queryCase{
			{id: 1, query: `*`, want: []string{"alpha.txt", "beta.txt", "box"}},
			{id: 2, query: `name:"*"`, want: []string{"alpha.txt", "beta.txt", "box"}},
			{id: 3, query: `*`, limit: 2, wantCount: conversions.ToPointer(2)},
			{id: 4, query: `*`, limit: -1, wantCount: conversions.ToPointer(3)},
		},
	}
}
