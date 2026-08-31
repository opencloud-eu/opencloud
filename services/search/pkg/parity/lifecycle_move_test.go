package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func moveLifecycle() lifecycleGroup {
	parent, child := fixtureTree()

	return lifecycleGroup{
		name:     "move",
		fixtures: []search.Resource{parent, child},
		cases: []lifecycleCase{
			{
				id: 1, title: "carries the descendants to the new path",
				do: func(e search.Engine) error {
					return e.Move(parent.ID, parent.ParentID, "./my/newname")
				},
				expect: []expectation{
					{`path:"./my/newname/child.pdf"`, []string{"child.pdf"}},
					{`path:"./parent/child.pdf"`, nil},
				},
			},
			{
				id: 2, title: "through the trash and back leaves the flag behind",
				do: func(e search.Engine) error {
					if err := e.Move(parent.ID, parent.ParentID, "./.trash/parent"); err != nil {
						return err
					}

					return e.Move(parent.ID, parent.ParentID, "./parent")
				},
				expect: []expectation{{`hidden:true`, nil}},
			},
		},
	}
}
