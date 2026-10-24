package parity

import (
	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func casePathLifecycle() lifecycleGroup {
	folder := fixtureFolder("Documents", withID("1$1!2"), withPath("./Documents"))
	picture := fixtureDoc("Picture.jpg",
		withID("1$1!3"),
		withParent(folder.ID),
		withMime("image/jpeg"),
		withPath("./Documents/Picture.jpg"),
	)

	left := func(names ...string) []expectation {
		return []expectation{
			{`path:"./Documents"`, names},
		}
	}

	return lifecycleGroup{
		name:     "casepath",
		fixtures: []search.Resource{folder, picture},
		cases: []lifecycleCase{
			{
				id: 1, title: "takes the descendants along when deleting",
				do:     func(e search.Engine) error { return e.Delete(folder.ID) },
				expect: left(),
			},
			{
				id: 2, title: "takes the descendants along when moving",
				do: func(e search.Engine) error { return e.Move(folder.ID, folder.ParentID, "./Other Documents") },
				expect: []expectation{
					{`path:"./Other Documents"`, []string{"Other Documents", "Picture.jpg"}},
					{`path:"./Documents"`, nil},
				},
			},
			{
				id: 3, title: "reaches the descendants when purging",
				do:           func(e search.Engine) error { return e.Purge(folder.ID, false) },
				expect:       left(),
				wantDocCount: conversions.ToPointer(uint64(0)),
			},
		},
	}
}
