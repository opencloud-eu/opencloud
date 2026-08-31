package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func mediatypeGroup() queryGroup {
	return queryGroup{
		name: "mediatype",
		fixtures: []search.Resource{
			fixtureDoc("notes.md", withMime("text/markdown")),
			fixtureDoc("photo.jpg", withMime("image/jpeg")),
			fixtureFolder("albums"),
			fixtureFolder("drafts"),
		},
		cases: []queryCase{
			{id: 1, query: `mediatype:text/markdown`, want: []string{"notes.md"}},
			{id: 2, query: `mediatype:TEXT/MARKDOWN`, want: []string{"notes.md"}, engineOverrides: map[string]override{"bleve": override{}}},
			{id: 3, query: `mediatype:image/jpeg`, want: []string{"photo.jpg"}},
			{id: 4, query: `mediatype:*jpeg`, want: []string{"photo.jpg"}},
			{id: 5, query: `mediatype:image`, want: []string{"photo.jpg"}},
			{id: 6, query: `mediatype:folder`, want: []string{"albums", "drafts"}},
		},
	}
}
