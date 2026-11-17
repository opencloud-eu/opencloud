// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// a double quote inside a quoted value is written as two double quotes, a
// backslash and a slash are plain characters (MS-KQL 2.3.1)
func quotingGroup() queryGroup {
	return queryGroup{
		name: "quoting",
		fixtures: []search.Resource{
			fixtureDoc(`Angebot "Quote" Test.txt`, withContent(`Der Kunde sagte "Quotewort" im Gespraech.`)),
			fixtureDoc(`Angebot Quote.txt`, withContent(`Quotewort ohne Anfuehrungszeichen`)),
			fixtureDoc(`back\slash.txt`, withContent(`liegt unter C:\temp\new`)),
			fixtureDoc("hi-tag.txt", withTags(`say "hi"`)),
			fixtureDoc("slash-tag.txt", withTags("a/b")),
			fixtureDoc("root-tag.txt", withTags("/")),
			fixtureDoc("backslash-tag.txt", withTags(`back\slash`)),
		},
		cases: []queryCase{
			{id: 1, query: `name:"*""Quote""*"`, want: []string{`Angebot "Quote" Test.txt`}},
			{id: 2, query: `name:"*Quote*"`, want: []string{`Angebot "Quote" Test.txt`, `Angebot Quote.txt`}},
			{id: 3, query: `name:"Angebot ""Quote"" Test.txt"`, want: []string{`Angebot "Quote" Test.txt`}},
			{id: 4, query: `name="Angebot ""Quote"" Test.txt"`, want: []string{`Angebot "Quote" Test.txt`}},
			// content is matched by its words, the quotes are not part of them
			{id: 5, query: `content:"""Quotewort"""`, want: []string{`Angebot "Quote" Test.txt`, `Angebot Quote.txt`}},
			{id: 6, query: `content:"sagte ""Quotewort"""`, want: []string{`Angebot "Quote" Test.txt`}},
			{id: 7, query: `name:"*back\slash*"`, want: []string{`back\slash.txt`}},
			{id: 8, query: `name:back\slash.txt`, want: []string{`back\slash.txt`}},
			{id: 9, query: `content:"C:\temp\new"`, want: []string{`back\slash.txt`}},
			{id: 10, query: `content:"foo\"`},
			// a backslash escapes nothing: this is name:"*\" AND Quote\ AND "*"
			{id: 11, query: `name:"*\"Quote\"*"`},
			{id: 12, query: `tag:"say ""hi"""`, want: []string{"hi-tag.txt"}},
			{id: 13, query: `tag:"a/b"`, want: []string{"slash-tag.txt"}},
			{id: 14, query: `tag:a/b`, want: []string{"slash-tag.txt"}},
			{id: 15, query: `tag:"/"`, want: []string{"root-tag.txt"}},
			{id: 16, query: `tag:"back\slash"`, want: []string{"backslash-tag.txt"}},
			{id: 17, query: `tag:"*\*"`, want: []string{"backslash-tag.txt"}},
			// a quote can't stand in an unquoted value, nor be left open
			{id: 18, query: `name:Quote"`, wantBadRequest: true},
			{id: 19, query: `content:"foo\""`, wantBadRequest: true},
			{id: 20, query: `name:"""`, wantBadRequest: true},
		},
	}
}
