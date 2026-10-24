package parity

import (
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func fieldsGroup() queryGroup {
	return queryGroup{
		name: "fields",
		fixtures: []search.Resource{
			fixtureDoc("small.txt", withSize(42)),
			fixtureDoc("old.txt", withMtime("2020-01-01T00:00:00Z")),
			fixtureDoc("known.txt", withID("1$1!23")),
			fixtureDoc("cased.txt", withID("1$1!AB-23")),
			fixtureDoc("hidden.txt", isHidden()),
			fixtureDoc("plain.txt"),
			fixtureFolder("box"),
			fixtureDoc("boxed.txt", withParent("1$1!box"), withPath("./box/boxed.txt")),
			fixtureDoc("song.mp3", withMime("audio/mpeg"), withAudio(&libregraph.Audio{Artist: libregraph.PtrString("Some Artist"), Duration: libregraph.PtrInt64(200)})),
		},
		cases: []queryCase{
			{id: 1, query: `size:42`, want: []string{"small.txt"}},
			{id: 2, query: `mtime<"2021-01-01T00:00:00Z"`, want: []string{"old.txt"}},
			{id: 3, query: `id:"1$1!23"`, want: []string{"known.txt"}},
			{id: 4, query: `hidden:true`, want: []string{"hidden.txt"}},
			{id: 5, query: `type:file`, want: []string{"small.txt", "old.txt", "known.txt", "cased.txt", "hidden.txt", "plain.txt", "boxed.txt", "song.mp3"}},
			{id: 6, query: `type:folder`, want: []string{"box"}},
			{id: 7, query: `unknown:field`},
			{id: 8, query: `type:File`, want: []string{"small.txt", "old.txt", "known.txt", "cased.txt", "hidden.txt", "plain.txt", "boxed.txt", "song.mp3"}},
			{id: 9, query: `type:FOLDER`, want: []string{"box"}},
			{id: 10, query: `hidden:TRUE`, want: []string{"hidden.txt"}},
			{id: 11, query: `id:"1$1!AB-23"`, want: []string{"cased.txt"}},
			{id: 12, query: `id:"1$1!ab-23"`},
			// a facet value keeps its case, the field is not marked lowercase
			{id: 13, query: `audio.artist:"Some Artist"`, want: []string{"song.mp3"}},
			{id: 14, query: `audio.artist:"some artist"`, want: []string{"song.mp3"}}, // facets search case-insensitively
			{id: 15, query: `audio.duration>100`},                                     // number queries are gated to Size and Type on both engines
		},
	}
}
