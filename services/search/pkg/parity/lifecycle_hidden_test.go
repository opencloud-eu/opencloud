package parity

import (
	"path"

	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func hiddenLifecycle() lifecycleGroup {
	group := lifecycleGroup{name: "hidden"}

	for _, move := range []struct {
		id     int
		title  string
		from   string
		target string
		hidden bool
	}{
		{1, "into a dot folder", "./parent", "./.trash/parent", true},
		{2, "into a plain folder", "./parent", "./archive/parent", false},
		{3, "renamed with a leading dot", "./parent", "./.parent", true},
		{4, "out of a dot folder", "./.trash/parent", "./archive/parent", false},
		{5, "renamed without the leading dot", "./.parent", "./parent", false},
		{6, "within the same dot folder", "./.trash/parent", "./.trash/moved", true},
	} {
		parent, child := fixtureTree()
		parent.Path = move.from
		parent.Hidden = search.IsHidden(move.from)
		child.Path = move.from + "/child.pdf"
		child.Hidden = parent.Hidden

		var hidden []string
		if move.hidden {
			hidden = []string{path.Base(move.target), "child.pdf"}
		}

		group.cases = append(group.cases, lifecycleCase{
			id: move.id, title: "follows a move " + move.title,
			fixtures: []search.Resource{parent, child},
			do: func(e search.Engine) error {
				return e.Move(parent.ID, parent.ParentID, move.target)
			},
			expect: []expectation{{`hidden:true`, hidden}},
		})
	}

	return group
}
