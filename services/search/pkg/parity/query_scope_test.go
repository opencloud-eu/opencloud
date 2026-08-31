package parity

import (
	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func scopeGroup() queryGroup {
	space := func(path string) *searchMessage.Reference {
		return &searchMessage.Reference{
			ResourceId: &searchMessage.ResourceID{StorageId: "1", SpaceId: "1", OpaqueId: "1"},
			Path:       path,
		}
	}

	return queryGroup{
		name: "scope",
		fixtures: []search.Resource{
			fixtureFolder("parent"),
			fixtureDoc("child.pdf", withPath("./parent/child.pdf")),
			fixtureDoc("outside.txt"),
			fixtureDoc("elsewhere.txt", withRoot("2$2!1")),
		},
		cases: []queryCase{
			{id: 1, query: `*`, want: []string{"parent", "child.pdf", "outside.txt", "elsewhere.txt"}},
			{id: 2, query: `*`, ref: space(""), want: []string{"parent", "child.pdf", "outside.txt"}},
			{id: 3, query: `*`, ref: space("./parent"), want: []string{"parent", "child.pdf"}},
			{id: 4, query: `*`, ref: space("./parent/child.pdf"), want: []string{"child.pdf"}},
			{id: 5, query: `name:"*elsewhere*"`, ref: space(""), want: nil},
		},
	}
}
