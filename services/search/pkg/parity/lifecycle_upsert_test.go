package parity

import (
	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func upsertLifecycle() lifecycleGroup {
	parent, child := fixtureTree()

	renamed := child
	renamed.Name = "renamed.pdf"
	renamed.Path = "./parent/renamed.pdf"

	return lifecycleGroup{
		name:     "upsert",
		fixtures: []search.Resource{parent, child},
		cases: []lifecycleCase{
			{
				id: 1, title: "replaces the resource it already knows",
				do:           func(e search.Engine) error { return e.Upsert(renamed.ID, renamed) },
				expect:       append(treeIsLeft("parent"), expectation{`name:"*renamed*"`, []string{"renamed.pdf"}}),
				wantDocCount: conversions.ToPointer(uint64(2)),
			},
		},
	}
}
