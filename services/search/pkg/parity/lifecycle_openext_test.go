package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// The extension siblings follow the stored value: a re-typed property leaves
// its old sibling, and the operations that re-index a resource from the index
// (Move) keep the extensions.
func openextLifecycle() lifecycleGroup {
	const project = "com.example.project"
	parent, child := fixtureTree()
	withOpenExtension(project, `{"state":"open","priority":3}`)(&child)

	retyped := child
	retyped.OpenExtensions = nil
	withOpenExtension(project, `{"state":"open","priority":"high"}`)(&retyped)

	dropped := child
	dropped.OpenExtensions = nil

	return lifecycleGroup{
		name:     "openextops",
		fixtures: []search.Resource{parent, child},
		cases: []lifecycleCase{
			{
				id: 1, title: "re-types a property on upsert",
				do: func(e search.Engine) error { return e.Upsert(retyped.ID, retyped) },
				expect: []expectation{
					{`extensions.com.example.project.priority:high`, []string{"child.pdf"}},
					{`extensions.com.example.project.priority>2`, nil},
					{`extensions.com.example.project.state:open`, []string{"child.pdf"}},
				},
			},
			{
				id: 2, title: "forgets a removed extension on upsert",
				do: func(e search.Engine) error { return e.Upsert(dropped.ID, dropped) },
				expect: []expectation{
					{`extensions.com.example.project.state:open`, nil},
					{`name:child.pdf`, []string{"child.pdf"}},
				},
			},
			{
				id: 3, title: "keeps the extensions through a move",
				do: func(e search.Engine) error { return e.Move(parent.ID, parent.ParentID, "./renamed") },
				expect: []expectation{
					{`extensions.com.example.project.priority>2`, []string{"child.pdf"}},
					{`path:"./renamed/child.pdf"`, []string{"child.pdf"}},
				},
			},
			{
				id: 4, title: "keeps the extensions through the trash and back",
				do: func(e search.Engine) error {
					if err := e.Delete(child.ID); err != nil {
						return err
					}
					return e.Restore(child.ID)
				},
				expect: []expectation{
					{`extensions.com.example.project.state:open`, []string{"child.pdf"}},
				},
			},
		},
	}
}
