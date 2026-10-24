package parity

import (
	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func batchLifecycle() lifecycleGroup {
	parent, child := fixtureTree()

	added := fixtureDoc("added.pdf", withID("1$1!7"), withParent(parent.ID), withPath("./parent/added.pdf"))
	other := fixtureDoc("other.pdf", withID("1$1!8"), withParent(parent.ID), withPath("./parent/other.pdf"))

	return lifecycleGroup{
		name:     "batch",
		fixtures: []search.Resource{parent, child},
		cases: []lifecycleCase{
			{
				id: 1, title: "holds what it was given until it is pushed",
				do: func(e search.Engine) error {
					batch, err := e.NewBatch(100)
					if err != nil {
						return err
					}

					return batch.Upsert(added.ID, added)
				},
				expect: []expectation{{`name:"*added*"`, nil}},
			},
			{
				id: 2, title: "writes what it was given once it is pushed",
				do: func(e search.Engine) error {
					batch, err := e.NewBatch(100)
					if err != nil {
						return err
					}

					if err := batch.Upsert(added.ID, added); err != nil {
						return err
					}

					return batch.Push()
				},
				expect:       []expectation{{`name:"*added*"`, []string{"added.pdf"}}},
				wantDocCount: conversions.ToPointer(uint64(3)),
			},
			{
				id: 3, title: "takes a resource out the same way a delete does",
				do: func(e search.Engine) error {
					batch, err := e.NewBatch(100)
					if err != nil {
						return err
					}

					if err := batch.Delete(child.ID); err != nil {
						return err
					}

					return batch.Push()
				},
				expect: treeIsLeft("parent"),
			},
			{
				id: 4, title: "moves a resource the same way a move does",
				do: func(e search.Engine) error {
					batch, err := e.NewBatch(100)
					if err != nil {
						return err
					}

					if err := batch.Move(parent.ID, parent.ParentID, "./my/newname"); err != nil {
						return err
					}

					return batch.Push()
				},
				expect: []expectation{
					{`path:"./my/newname/child.pdf"`, []string{"child.pdf"}},
					{`path:"./parent/child.pdf"`, nil},
				},
			},
			{
				id: 5, title: "keeps what another batch holds out of its push",
				do: func(e search.Engine) error {
					first, err := e.NewBatch(100)
					if err != nil {
						return err
					}

					second, err := e.NewBatch(100)
					if err != nil {
						return err
					}

					if err := first.Upsert(added.ID, added); err != nil {
						return err
					}

					if err := second.Upsert(other.ID, other); err != nil {
						return err
					}

					return first.Push()
				},
				expect: []expectation{
					{`name:"*added*"`, []string{"added.pdf"}},
					{`name:"*other*"`, nil},
				},
			},
		},
	}
}
