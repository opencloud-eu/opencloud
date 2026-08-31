package parity

import (
	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func purgeLifecycle() lifecycleGroup {
	parent, child := fixtureTree()

	return lifecycleGroup{
		name:     "purge",
		fixtures: []search.Resource{parent, child},
		cases: []lifecycleCase{
			{
				id: 1, title: "removes one resource",
				do:           func(e search.Engine) error { return e.Purge(child.ID, false) },
				expect:       treeIsLeft("parent"),
				wantDocCount: conversions.ToPointer(uint64(1)),
			},
			{
				id: 2, title: "removes the tree",
				do:           func(e search.Engine) error { return e.Purge(parent.ID, false) },
				expect:       treeIsLeft(),
				wantDocCount: conversions.ToPointer(uint64(0)),
			},
			{
				id: 3, title: "takes only the deleted ones when it is told to",
				do: func(e search.Engine) error {
					if err := e.Delete(child.ID); err != nil {
						return err
					}

					return e.Purge(parent.ID, true)
				},
				expect: treeIsLeft("parent"),
			},
		},
	}
}
