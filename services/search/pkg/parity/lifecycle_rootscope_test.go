package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func rootScopeLifecycle() lifecycleGroup {
	const shared = "./same/path.txt"

	target := fixtureDoc("target.txt", withID("1$1!3"), withPath(shared))
	twin := fixtureDoc("twin.txt", withID("2$2!3"), withRoot("2$2!1"), withParent("2$2!2"), withPath(shared))

	deleted := func(r search.Resource) search.Resource {
		r.Deleted = true

		return r
	}

	return lifecycleGroup{
		name:     "rootscope",
		fixtures: []search.Resource{target, twin},
		cases: []lifecycleCase{
			{
				id: 1, title: "deletes only the one in the target root",
				do: func(e search.Engine) error { return e.Delete(target.ID) },
				expect: []expectation{
					{`name:"*target*"`, nil},
					{`name:"*twin*"`, []string{"twin.txt"}},
				},
			},
			{
				id: 2, title: "restores only the one in the target root",
				fixtures: []search.Resource{deleted(target), deleted(twin)},
				do:       func(e search.Engine) error { return e.Restore(target.ID) },
				expect: []expectation{
					{`name:"*target*"`, []string{"target.txt"}},
					{`name:"*twin*"`, nil},
				},
			},
			{
				id: 4, title: "purges only the one in the target root",
				do: func(e search.Engine) error { return e.Purge(target.ID, false) },
				expect: []expectation{
					{`name:"*target*"`, nil},
					{`name:"*twin*"`, []string{"twin.txt"}},
				},
			},
			{
				id: 3, title: "moves only the one in the target root",
				do: func(e search.Engine) error { return e.Move(target.ID, target.ParentID, "./moved.txt") },
				expect: []expectation{
					{`path:"./moved.txt"`, []string{"moved.txt"}},
					{`path:"` + shared + `"`, []string{"twin.txt"}},
				},
			},
		},
	}
}
