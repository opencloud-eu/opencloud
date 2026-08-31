package parity

import (
	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func deleteLifecycle() lifecycleGroup {
	parent, child := fixtureTree()

	return lifecycleGroup{
		name:     "delete",
		fixtures: []search.Resource{parent, child},
		cases: []lifecycleCase{
			{
				id: 1, title: "takes the resource out of the results",
				do:     func(e search.Engine) error { return e.Delete(child.ID) },
				expect: treeIsLeft("parent"),
			},
			{
				id: 2, title: "takes the descendants along",
				do:     func(e search.Engine) error { return e.Delete(parent.ID) },
				expect: treeIsLeft(),
			},
			{
				id: 3, title: "leaves the resource in the index",
				do:           func(e search.Engine) error { return e.Delete(child.ID) },
				wantDocCount: conversions.ToPointer(uint64(2)),
				engineOverrides: map[string]lifecycleOverride{
					"opensearch": {wantDocCount: conversions.ToPointer(uint64(1))},
				},
			},
			{
				id: 4, title: "takes a resource out that was just written",
				do: func(e search.Engine) error {
					fresh := fixtureDoc("fresh.txt", withID("1$1!8"))
					if err := e.Upsert(fresh.ID, fresh); err != nil {
						return err
					}

					return e.Delete(fresh.ID)
				},
				expect: []expectation{{`name:"*fresh*"`, nil}},
			},
		},
	}
}
