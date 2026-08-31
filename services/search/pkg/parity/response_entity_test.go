package parity

import (
	"fmt"

	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func entityGroup() responseGroup {
	parent := fixtureFolder("parent", withID("1$1!2"))
	doc := fixtureDoc("bar.pdf",
		withID("1$1!3"),
		withParent(parent.ID),
		withPath("./parent/bar.pdf"),
		withMime("application/pdf"),
		withSize(1234),
	)
	notes := fixtureDoc("notes.txt", withID("1$1!4"), withContent("foo bar baz"))

	moved := func(e search.Engine) error { return e.Move(parent.ID, "1$1!9", "./somewhere/newname") }

	return responseGroup{
		name:     "entity",
		fixtures: []search.Resource{parent, doc, notes},
		cases: []responseCase{
			{
				id: 1, query: `name:"bar.pdf"`, reads: "Ref.Path", want: []string{"./parent/bar.pdf"},
				read: reads(func(m *searchMessage.Match) string { return m.GetEntity().GetRef().GetPath() }),
			},
			{
				id: 2, query: `name:"bar.pdf"`, reads: "Name", want: []string{"bar.pdf"},
				read: reads(func(m *searchMessage.Match) string { return m.GetEntity().GetName() }),
			},
			{
				id: 3, query: `name:"bar.pdf"`, reads: "Id", want: []string{"1$1!3"},
				read: reads(func(m *searchMessage.Match) string { return resourceID(m.GetEntity().GetId()) }),
			},
			{
				id: 4, query: `name:"bar.pdf"`, reads: "ParentId", want: []string{"1$1!2"},
				read: reads(func(m *searchMessage.Match) string { return resourceID(m.GetEntity().GetParentId()) }),
			},
			{
				id: 5, query: `name:"bar.pdf"`, reads: "Ref.ResourceId", want: []string{"1$1!1"},
				read: reads(func(m *searchMessage.Match) string { return resourceID(m.GetEntity().GetRef().GetResourceId()) }),
			},
			{
				id: 6, query: `name:"bar.pdf"`, reads: "Size", want: []string{"1234"},
				read: reads(func(m *searchMessage.Match) string { return fmt.Sprint(m.GetEntity().GetSize()) }),
			},
			{
				id: 7, query: `name:"bar.pdf"`, reads: "Type", want: []string{"1"},
				read: reads(func(m *searchMessage.Match) string { return fmt.Sprint(m.GetEntity().GetType()) }),
			},
			{
				id: 8, query: `name:"bar.pdf"`, reads: "MimeType", want: []string{"application/pdf"},
				read: reads(func(m *searchMessage.Match) string { return m.GetEntity().GetMimeType() }),
			},
			{
				id: 9, query: `name:"bar.pdf"`, reads: "Deleted", want: []string{"false"},
				read: reads(func(m *searchMessage.Match) string { return fmt.Sprint(m.GetEntity().GetDeleted()) }),
			},
			{
				id: 10, query: `name:"bar.pdf"`, reads: "Score", want: []string{"above zero"},
				read: reads(func(m *searchMessage.Match) string {
					if m.GetScore() > 0 {
						return "above zero"
					}

					return fmt.Sprint(m.GetScore())
				}),
			},
			{
				id: 11, query: `path:"./parent"`, reads: "TotalMatches", want: []string{"2"}, engineOverrides: map[string]override{"bleve": override{want: []string{"1"}}},
				read: func(resp *searchService.SearchIndexResponse) []string {
					return []string{fmt.Sprint(resp.GetTotalMatches())}
				},
			},
			{
				id: 12, query: `name:"*notes*"`, reads: "Highlights", want: []string{`""`},
				read: reads(func(m *searchMessage.Match) string { return fmt.Sprintf("%q", m.GetEntity().GetHighlights()) }),
			},
			{
				id: 13, query: `content:bar`, reads: "Highlights", want: []string{"foo <mark>bar</mark> baz"},
				read: reads(func(m *searchMessage.Match) string { return m.GetEntity().GetHighlights() }),
			},
			{
				id: 14, do: moved, after: "moved to another parent", query: `name:"newname"`, reads: "ParentId", want: []string{"1$1!9"},
				read: reads(func(m *searchMessage.Match) string { return resourceID(m.GetEntity().GetParentId()) }),
			},
			{
				id: 15, do: moved, after: "moved to another parent", query: `name:"bar.pdf"`, reads: "ParentId", want: []string{"1$1!2"},
				read: reads(func(m *searchMessage.Match) string { return resourceID(m.GetEntity().GetParentId()) }),
			},
			{
				id: 16, do: moved, after: "moved to another parent", query: `name:"bar.pdf"`, reads: "Ref.Path", want: []string{"./somewhere/newname/bar.pdf"},
				read: reads(func(m *searchMessage.Match) string { return m.GetEntity().GetRef().GetPath() }),
			},
		},
	}
}
