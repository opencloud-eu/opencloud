package parity

import (
	"github.com/opencloud-eu/opencloud/pkg/conversions"
	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func samenameGroup() queryGroup {
	folder := fixtureFolder("doc", withID("1$1!2"))
	under := func(path string) *searchMessage.Reference {
		return &searchMessage.Reference{
			ResourceId: &searchMessage.ResourceID{StorageId: "1", SpaceId: "1", OpaqueId: "1"},
			Path:       path,
		}
	}

	return queryGroup{
		name: "samename",
		fixtures: []search.Resource{
			folder,
			fixtureDoc("doc.pdf", withID("1$1!3")),
			fixtureDoc("file.pdf", withID("1$1!4")),
			fixtureDoc("doc.pdf", withID("1$1!5"), withParent(folder.ID), withPath("./doc/doc.pdf")),
			fixtureDoc("file.pdf", withID("1$1!6"), withParent(folder.ID), withPath("./doc/file.pdf")),
		},
		cases: []queryCase{
			{id: 1, query: `name:"*doc*"`, wantCount: conversions.ToPointer(3)},
			{id: 2, query: `name:"*doc*"`, ref: under("./doc"), wantCount: conversions.ToPointer(2)},
			{id: 3, query: `name:"*file*"`, wantCount: conversions.ToPointer(2)},
			{id: 4, query: `name:"*file*"`, ref: under("./doc"), wantCount: conversions.ToPointer(1)},
		},
	}
}
