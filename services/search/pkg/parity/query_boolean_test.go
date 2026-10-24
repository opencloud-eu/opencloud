package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func booleanGroup() queryGroup {
	return queryGroup{
		name: "boolean",
		fixtures: []search.Resource{
			fixtureDoc("alpha.txt", withTags("red")),
			fixtureDoc("beta.txt", withTags("blue")),
			fixtureDoc("gamma.md", withMime("text/markdown")),
		},
		cases: []queryCase{
			{id: 1, query: `name:"*alpha*" AND name:"*txt*"`, want: []string{"alpha.txt"}},
			{id: 2, query: `name:"*alpha*" OR name:"*beta*"`, want: []string{"alpha.txt", "beta.txt"}},
			{id: 3, query: `name:"*a*" AND NOT name:"*alpha*"`, want: []string{"beta.txt", "gamma.md"}},
			{id: 4, query: `name:"*a*" AND tag:("red")`, want: []string{"alpha.txt"}},
			{id: 5, query: `(name:"*alpha*" OR name:"*beta*") AND mediatype:text/plain`, want: []string{"alpha.txt", "beta.txt"}},
			{id: 6, query: `name:"*a*" AND mediatype:text/markdown`, want: []string{"gamma.md"}},
		},
	}
}
