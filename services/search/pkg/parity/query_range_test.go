package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func rangeGroup() queryGroup {
	return queryGroup{
		name: "range",
		fixtures: []search.Resource{
			fixtureDoc("small.txt", withSize(50)),
			fixtureDoc("big.txt", withSize(500)),
			fixtureDoc("ancient.txt", withSize(10), withMtime("2020-01-01T00:00:00Z")),
		},
		cases: []queryCase{
			{id: 1, query: `size>100`, want: []string{"big.txt"}},
			{id: 2, query: `size<100`, want: []string{"small.txt", "ancient.txt"}},
			{id: 3, query: `mtime>"2021-01-01T00:00:00Z"`, want: []string{"small.txt", "big.txt"}},
			{id: 4, query: `mtime<"2021-01-01T00:00:00Z"`, want: []string{"ancient.txt"}},
			{id: 5, query: `Mtime:"today"`, want: []string{"small.txt", "big.txt"}},
			{id: 6, query: `Mtime:"yesterday"`},
			{id: 7, query: `mtime>2021`},
			{id: 8, query: `name>100`},
			{id: 9, query: `size>=50`, want: []string{"small.txt", "big.txt"}},
			{id: 10, query: `size<=50`, want: []string{"small.txt", "ancient.txt"}},
			{id: 11, query: `size>100 AND size<1000`, want: []string{"big.txt"}},
			{id: 12, query: `mtime>=2021-01-01`, want: []string{"small.txt", "big.txt"}},
			// a number on a numeric field matches as a whole, with : and with =
			{id: 13, query: `size:50`, want: []string{"small.txt"}},
			{id: 14, query: `size=50`, want: []string{"small.txt"}},
			{id: 15, query: `size:1.5`},
			{id: 16, query: `size:5*`},
			// not KQL: a colon before the comparison makes it the text ">100",
			// which no number is. Old iOS clients send this form.
			{id: 17, query: `size:>100`},
			{id: 18, query: `size:<100`},
			{id: 19, query: `size:>=100`},
			{id: 20, query: `mtime:>"2021-01-01T00:00:00Z"`},
			{id: 21, query: `mtime:>2021-01-01`},
			// the KQL range operator .. is not supported
			{id: 22, query: `size:50..500`},
			{id: 23, query: `mtime:2019-01-01..2020-12-31`},
		},
	}
}
