package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func extractionGroup() queryGroup {
	return queryGroup{
		name: "extraction",
		fixtures: []search.Resource{
			fixtureDoc("given-up.pdf", withExtractionFailure(3, "")),
			fixtureDoc("waiting.png", withExtractionFailure(1, "2030-01-01T00:00:00Z")),
			fixtureDoc("extracted.txt"),
		},
		cases: []queryCase{
			{id: 1, query: `extractionfailure.count>=3`, want: []string{"given-up.pdf"}},
			{id: 2, query: `ExtractionFailure.Count>=1`, want: []string{"given-up.pdf", "waiting.png"}},
			{id: 3, query: `extractionfailure.retryat>"2029-01-01T00:00:00Z"`, want: []string{"waiting.png"}},
			{id: 4, query: `extractionfailure.retryat<"2029-01-01T00:00:00Z"`},
		},
	}
}
