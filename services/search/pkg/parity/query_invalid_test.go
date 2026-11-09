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
			{id: 2, query: `mediatype:document AND`, wantBadRequest: true},
			{id: 3, query: `mediatype:document NOT`, wantBadRequest: true},
			{id: 4, query: `mediatype:document AND OR name:alpha`, wantBadRequest: true},
			{id: 5, query: `(mediatype:document`, wantBadRequest: true},
			{id: 6, query: `mediatype:document) OR (name:alpha`, wantBadRequest: true},
		},
	}
}
