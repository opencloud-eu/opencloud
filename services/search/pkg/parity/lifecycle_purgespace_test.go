package parity

import (
	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func purgeSpaceLifecycle() lifecycleGroup {
	inSpace := fixtureDoc("inSpace.txt", withID("1$1!4"))
	elsewhere := fixtureDoc("elsewhere.txt", withID("2$2!4"), withRoot("2$2!1"))

	return lifecycleGroup{
		name:     "purgespace",
		fixtures: []search.Resource{inSpace, elsewhere},
		cases: []lifecycleCase{
			{
				id: 1, title: "leaves the other space alone",
				do: func(e search.Engine) error { return e.PurgeSpace(inSpace.RootID) },
				expect: []expectation{
					{`name:"*inSpace*"`, nil},
					{`name:"*elsewhere*"`, []string{"elsewhere.txt"}},
				},
			},
			{
				id: 2, title: "takes a space out that holds more than one round",
				fixtures: append(fixtureBulk(60), elsewhere),
				do:       func(e search.Engine) error { return e.PurgeSpace(inSpace.RootID) },
				expect: []expectation{
					{`name:"*bulk*"`, nil},
					{`name:"*elsewhere*"`, []string{"elsewhere.txt"}},
				},
				wantDocCount: conversions.ToPointer(uint64(1)),
			},
		},
	}
}
