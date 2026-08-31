package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func titleGroup() queryGroup {
	return queryGroup{
		name: "title",
		fixtures: []search.Resource{
			fixtureDoc("q1.html", withMime("text/html"), withTitle("quarterly report")),
		},
		cases: []queryCase{
			{id: 1, query: `Title:"quarterly report"`, want: []string{"q1.html"}},
			{id: 2, query: `Title:quarterly`, want: []string{"q1.html"}, engineOverrides: map[string]override{"bleve": override{}}},
			{id: 3, query: `Title:QUARTERLY`, want: []string{"q1.html"}, engineOverrides: map[string]override{"bleve": override{}}},
			{id: 4, query: `Title:quarterl*`, want: []string{"q1.html"}},
			{id: 5, query: `Title:"*ly rep*"`, want: []string{"q1.html"}},
			{id: 6, query: `title:quarterly`, want: []string{"q1.html"}, engineOverrides: map[string]override{"bleve": override{}, "opensearch": override{}}},
			{id: 7, query: `Title:"QUARTERLY REPORT"`, want: []string{"q1.html"}, engineOverrides: map[string]override{"bleve": override{}}},
		},
	}
}
