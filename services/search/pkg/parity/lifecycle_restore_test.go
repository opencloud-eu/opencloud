package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func restoreLifecycle() lifecycleGroup {
	parent, child := fixtureTree()
	secret := fixtureDoc("file.txt", withID("1$1!5"), withPath("./.secret/file.txt"), isHidden())

	return lifecycleGroup{
		name:     "restore",
		fixtures: []search.Resource{parent, child},
		cases: []lifecycleCase{
			{
				id: 1, title: "brings the descendants back",
				do: func(e search.Engine) error {
					if err := e.Delete(parent.ID); err != nil {
						return err
					}

					return e.Restore(parent.ID)
				},
				expect: treeIsLeft("parent", "child.pdf"),
			},
			{
				id: 2, title: "leaves the hidden flag alone",
				fixtures: []search.Resource{secret},
				do: func(e search.Engine) error {
					if err := e.Delete(secret.ID); err != nil {
						return err
					}

					return e.Restore(secret.ID)
				},
				expect: []expectation{{`hidden:true`, []string{"file.txt"}}},
			},
		},
	}
}
