package parity

import (
	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func idempotencyLifecycle() lifecycleGroup {
	parent, child := fixtureTree()
	missing := "1$1!gone"

	return lifecycleGroup{
		name:     "idempotency",
		fixtures: []search.Resource{parent, child},
		cases: []lifecycleCase{
			{
				id: 1, title: "deleting the same resource twice is not an error",
				do: func(e search.Engine) error {
					if err := e.Delete(child.ID); err != nil {
						return err
					}

					return e.Delete(child.ID)
				},
				expect: treeIsLeft("parent"),
			},
			{
				id: 2, title: "deleting a resource the index does not have reports it",
				do:      func(e search.Engine) error { return e.Delete(missing) },
				expect:  treeIsLeft("parent", "child.pdf"),
				wantErr: true,
			},
			{
				id: 3, title: "restoring a resource that was never deleted leaves it alone",
				do:     func(e search.Engine) error { return e.Restore(child.ID) },
				expect: treeIsLeft("parent", "child.pdf"),
			},
			{
				id: 4, title: "purging the same resource twice reports the second one",
				do: func(e search.Engine) error {
					if err := e.Purge(child.ID, false); err != nil {
						return err
					}

					return e.Purge(child.ID, false)
				},
				expect:       treeIsLeft("parent"),
				wantDocCount: conversions.ToPointer(uint64(1)),
				wantErr:      true,
			},
			{
				id: 5, title: "purging a resource the index does not have reports it",
				do:           func(e search.Engine) error { return e.Purge(missing, false) },
				expect:       treeIsLeft("parent", "child.pdf"),
				wantDocCount: conversions.ToPointer(uint64(2)),
				wantErr:      true,
			},
			{
				id: 6, title: "moving a resource onto its own path leaves it where it is",
				do: func(e search.Engine) error {
					return e.Move(parent.ID, parent.ParentID, parent.Path)
				},
				expect: []expectation{
					{`path:"./parent/child.pdf"`, []string{"child.pdf"}},
					{`name:"*parent*"`, []string{"parent"}},
				},
			},
			{
				id: 7, title: "purging a whole space that holds nothing is not an error",
				do:           func(e search.Engine) error { return e.PurgeSpace("9$9!9") },
				expect:       treeIsLeft("parent", "child.pdf"),
				wantDocCount: conversions.ToPointer(uint64(2)),
			},
		},
	}
}
