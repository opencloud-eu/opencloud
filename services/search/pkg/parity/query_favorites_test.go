package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func favoritesGroup() queryGroup {
	return queryGroup{
		name: "favorites",
		fixtures: []search.Resource{
			fixtureDoc("starred.txt", withFavorite("A1B2-Upper")),
			fixtureDoc("plain.txt"),
			fixtureFolder("keepsakes", withFavorite("A1B2-Upper")),
			fixtureDoc("photo.jpg", withParent("1$1!keepsakes"), withPath("./keepsakes/photo.jpg"), withMime("image/jpeg")),
		},
		cases: []queryCase{
			{id: 1, query: `Favorites:"A1B2-Upper"`, want: []string{"starred.txt", "keepsakes"}},
			{id: 2, query: `favorite:"A1B2-Upper"`, want: []string{"starred.txt", "keepsakes"}},
			{id: 3, query: `Favorites:"somebody-else"`},
		},
	}
}
