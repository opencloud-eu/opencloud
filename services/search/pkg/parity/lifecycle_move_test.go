package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func moveLifecycle() lifecycleGroup {
	parent, child := fixtureTree()
	big := fixtureFolder("big", withID("1$1!4"))
	inBig := fixtureDoc("f1.txt", withID("1$1!5"), withParent(big.ID), withPath("./big/f1.txt"))
	inSibling := fixtureDoc("x.txt", withID("1$1!6"), withParent("1$1!big2"), withPath("./big2/x.txt"))
	odd := fixtureFolder("odd name (1)", withID("1$1!7"))
	inOdd := fixtureDoc("f:x+y.txt", withID("1$1!8"), withParent(odd.ID), withPath("./odd name (1)/f:x+y.txt"))

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
			{
				id: 3, title: "leaves a sibling folder that shares the prefix alone",
				fixtures: []search.Resource{big, inBig, inSibling},
				do:       func(e search.Engine) error { return e.Move(big.ID, big.ParentID, "./moved") },
				expect: []expectation{
					{`path:"./moved"`, []string{"moved", "f1.txt"}},
					{`path:"./big"`, nil},
					{`path:"./big2"`, []string{"x.txt"}},
				},
			},
			{
				id: 4, title: "carries the descendants of a path with special characters",
				fixtures: []search.Resource{odd, inOdd},
				do:       func(e search.Engine) error { return e.Move(odd.ID, odd.ParentID, "./odd name (2)") },
				expect: []expectation{
					{`path:"./odd name (2)"`, []string{"odd name (2)", "f:x+y.txt"}},
					{`path:"./odd name (1)"`, nil},
				},
			},
		},
	}
}
