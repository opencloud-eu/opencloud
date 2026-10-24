package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func visibilityGroup() queryGroup {
	return queryGroup{
		name: "visibility",
		fixtures: []search.Resource{
			fixtureDoc("visible.txt"),
			fixtureDoc("dotfile.txt", isHidden()),
			fixtureFolder(".private", isHidden()),
			fixtureDoc("secret.txt", withParent("1$1!.private"), withPath("./.private/secret.txt"), isHidden()),
		},
		cases: []queryCase{
			{id: 1, query: `hidden:true`, want: []string{"dotfile.txt", ".private", "secret.txt"}},
			{id: 2, query: `hidden:TRUE`, want: []string{"dotfile.txt", ".private", "secret.txt"}},
			{id: 3, query: `hidden:false`, want: []string{"visible.txt"}},
			{id: 4, query: `name:"*secret*"`, want: []string{"secret.txt"}},
			{id: 5, query: `path:"./.private"`, want: []string{".private", "secret.txt"}},
			{id: 6, query: `hidden:banana`},
			{id: 7, query: `hidden:"true"`, want: []string{"dotfile.txt", ".private", "secret.txt"}},
		},
	}
}
