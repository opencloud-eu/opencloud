package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func invalidGroup() queryGroup {
	return queryGroup{
		name: "invalid",
		fixtures: []search.Resource{
			fixtureDoc("alpha.txt"),
		},
		cases: []queryCase{
			{id: 1, query: `AND mediatype:document`, wantBadRequest: true},
			{id: 2, query: `mediatype:document AND`, want: []string{"alpha.txt"}},
		},
	}
}
