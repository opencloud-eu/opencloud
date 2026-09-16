package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// Open extensions are addressed as extensions.<name>.<property>. Every
// property is indexed in the sibling of its value's kind, and the literal of a
// restriction picks the sibling: a range needs a typed literal, an equality
// asks every sibling the literal fits.
func openextGroup() queryGroup {
	const project = "com.example.project"

	return queryGroup{
		name: "openext",
		fixtures: []search.Resource{
			fixtureDoc("plan.txt", withOpenExtension(project,
				`{"state":"Open","priority":3,"done":false,"due":"2026-10-01T00:00:00Z","due@odata.type":"#DateTimeOffset","tags":["Urgent","customer"],"site":{"latitude":52.5,"longitude":13.4},"site@odata.type":"#microsoft.graph.geoCoordinates"}`)),
			fixtureDoc("draft.txt", withOpenExtension(project,
				`{"state":"open","priority":1.5,"done":true,"due":"2026-12-24T00:00:00Z","due@odata.type":"#DateTimeOffset"}`)),
			fixtureDoc("legacy.txt", withOpenExtension(project, `{"state":"closed","priority":"3","due":"2026-10-01T00:00:00Z"}`)),
			fixtureDoc("other.txt", withOpenExtension("com.example.other", `{"state":"open"}`)),
			fixtureDoc("plain.txt"),
		},
		cases: []queryCase{
			{id: 1, query: `extensions.com.example.project.state:open`, want: []string{"plan.txt", "draft.txt"}},
			{id: 2, query: `extensions.com.example.project.state=Open`, want: []string{"plan.txt"}},
			{id: 3, query: `extensions.com.example.project.state=open`, want: []string{"draft.txt"}},
			{id: 4, query: `extensions.com.example.project.state:op*`, want: []string{"plan.txt", "draft.txt"}},
			{id: 5, query: `Extensions.com.example.project.state:closed`, want: []string{"legacy.txt"}},
			{id: 6, query: `extensions.com.example.project.priority>2`, want: []string{"plan.txt"}},
			{id: 7, query: `extensions.com.example.project.priority<2`, want: []string{"draft.txt"}},
			{id: 8, query: `extensions.com.example.project.priority:3`, want: []string{"plan.txt", "legacy.txt"}},
			{id: 9, query: `extensions.com.example.project.priority:"3"`, want: []string{"plan.txt", "legacy.txt"}},
			{id: 10, query: `extensions.com.example.project.done:true`, want: []string{"draft.txt"}},
			{id: 11, query: `extensions.com.example.project.done:false`, want: []string{"plan.txt"}},
			{id: 12, query: `extensions.com.example.project.due>2026-11-01T00:00:00Z`, want: []string{"draft.txt"}},
			{id: 13, query: `extensions.com.example.project.due<2026-11-01T00:00:00Z`, want: []string{"plan.txt"}},
			// a date-time literal is typed by the parser and asks the date sibling
			// only; legacy.txt holds the same text as a plain string
			{id: 14, query: `extensions.com.example.project.due:2026-10-01T00:00:00Z`, want: []string{"plan.txt"}},
			{id: 15, query: `extensions.com.example.project.tags:urgent`, want: []string{"plan.txt"}},
			{id: 16, query: `extensions.com.example.project.tags:customer AND extensions.com.example.project.state:open`, want: []string{"plan.txt"}},
			{id: 17, query: `extensions.com.example.project.state:open OR extensions.com.example.other.state:open`, want: []string{"plan.txt", "draft.txt", "other.txt"}},
			{id: 18, query: `NOT extensions.com.example.project.state:open`, want: []string{"legacy.txt", "other.txt", "plain.txt"}},
			{id: 19, query: `extensions.com.example.project.missing:x`},
			{id: 20, query: `extensions.com.example.unknown.state:open`},
		},
	}
}
