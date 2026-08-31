package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func contentGroup() queryGroup {
	return queryGroup{
		name: "content",
		fixtures: []search.Resource{
			fixtureDoc("monthly.txt", withContent("the monthly reports are due")),
			fixtureDoc("links.txt", withContent("see https://opencloud.example.com/help or write to alan@example.org")),
		},
		cases: []queryCase{
			{id: 1, query: `Content:report`},
			{id: 2, query: `Content:REPORTS`, want: []string{"monthly.txt"}},
			{id: 3, query: `Content:"monthly reports"`, want: []string{"monthly.txt"}},
			{id: 4, query: `Content:"reports monthly"`},
			{id: 5, query: `Content:report*`, want: []string{"monthly.txt"}},
			{id: 6, query: `Content:*eport*`, want: []string{"monthly.txt"}},
			{id: 7, query: `Content:month*`, want: []string{"monthly.txt"}},
			{id: 8, query: `Content:"https://opencloud.example.com/help"`, want: []string{"links.txt"}},
			{id: 9, query: `Content:"alan@example.org"`, want: []string{"links.txt"}},
			{id: 10, query: `Content:opencloud`, want: []string{"links.txt"}},
		},
	}
}
