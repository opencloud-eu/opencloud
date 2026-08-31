package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func stressGroup() queryGroup {
	return queryGroup{
		name: "stress",
		fixtures: []search.Resource{
			fixtureDoc("quarterly report.docx",
				withMime("application/vnd.openxmlformats-officedocument.wordprocessingml.document"),
				withTags("final"), withSize(2000)),
			fixtureDoc("draft report.txt", withTags("draft"), withSize(50), withMtime("2020-01-01T00:00:00Z")),
			fixtureDoc("photo.jpg", withMime("image/jpeg"), withTags("final")),
			fixtureDoc("notes.md", withMime("text/markdown"), isHidden()),
			fixtureFolder("archive"),
		},
		cases: []queryCase{
			{id: 1, query: `name:"*report*" AND mediatype:document`, want: []string{"quarterly report.docx", "draft report.txt"}},
			{id: 2, query: `name:"*report*" AND NOT tag:("draft")`, want: []string{"quarterly report.docx"}, engineOverrides: map[string]override{"bleve": override{want: []string{"draft report.txt", "quarterly report.docx"}}}},
			{id: 3, query: `(tag:("final") OR tag:("draft")) AND mediatype:image`, want: []string{"photo.jpg"}},
			{id: 4, query: `mediatype:document AND mtime>"2021-01-01T00:00:00Z"`, want: []string{"quarterly report.docx", "notes.md"}},
			{id: 5, query: `name:"*report*" AND size>100`, want: []string{"quarterly report.docx"}, engineOverrides: map[string]override{"bleve": override{}, "opensearch": override{}}},
			{id: 6, query: `tag:("final") AND NOT mediatype:folder`, want: []string{"quarterly report.docx", "photo.jpg"}},
			{id: 7, query: `hidden:true AND name:"*notes*"`, want: []string{"notes.md"}, engineOverrides: map[string]override{"bleve": override{}}},
			{id: 8, query: `name:quarterly report`, want: []string{"quarterly report.docx"}, engineOverrides: map[string]override{"bleve": override{}, "opensearch": override{}}},
			{id: 9, query: `name:"quarterly report"`},
			{id: 10, query: `name:"quarterly report.docx"`, want: []string{"quarterly report.docx"}},
			{id: 11, query: `NOT tag:("draft")`, want: []string{"quarterly report.docx", "photo.jpg", "notes.md", "archive"}, engineOverrides: map[string]override{"bleve": override{want: []string{"archive", "draft report.txt", "notes.md", "photo.jpg", "quarterly report.docx"}}}},
			{id: 12, query: `tag:("final") OR hidden:true`, want: []string{"quarterly report.docx", "photo.jpg", "notes.md"}, engineOverrides: map[string]override{"bleve": override{want: []string{"photo.jpg", "quarterly report.docx"}}}},
			{id: 13, query: `(name:"*report*" OR name:"*notes*") AND NOT (tag:("draft") OR hidden:true)`, want: []string{"quarterly report.docx"}, engineOverrides: map[string]override{"bleve": override{want: []string{"notes.md", "quarterly report.docx"}}}},
			{id: 14, query: `mediatype:image OR (mediatype:document AND tag:("draft"))`, want: []string{"photo.jpg", "draft report.txt"}},
			{id: 15, query: `NOT (mediatype:folder OR hidden:true)`, want: []string{"quarterly report.docx", "draft report.txt", "photo.jpg"}, engineOverrides: map[string]override{"bleve": override{want: []string{"draft report.txt", "notes.md", "photo.jpg", "quarterly report.docx"}}}},
			{id: 16, query: `name:"*report*" AND (size>100 OR tag:("draft"))`, want: []string{"quarterly report.docx", "draft report.txt"}, engineOverrides: map[string]override{"bleve": override{want: []string{"draft report.txt"}}, "opensearch": override{want: []string{"draft report.txt"}}}},
		},
	}
}
