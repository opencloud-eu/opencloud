package parity

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"

	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// sortCase holds the engines to the order of their matches.
type sortCase struct {
	id             int
	query          string
	orderBy        []*searchService.SortProperty
	reads          string
	want           []string
	wantBadRequest bool
}

type sortGroup struct {
	name     string
	fixtures []search.Resource
	cases    []sortCase
}

func (c sortCase) label() string { return fmt.Sprintf("SORT-%02d", c.id) }

func asc(name string) *searchService.SortProperty {
	return &searchService.SortProperty{Name: name}
}

func desc(name string) *searchService.SortProperty {
	return &searchService.SortProperty{Name: name, IsDescending: true}
}

func track(name string, size uint64, mtime string, duration int64, year int32) search.Resource {
	return fixtureDoc(name, withMime("audio/mpeg"), withSize(size), withMtime(mtime),
		withAudio(&libregraph.Audio{Duration: libregraph.PtrInt64(duration), Year: libregraph.PtrInt32(year)}))
}

func takenPhoto(name string, taken string) search.Resource {
	t, err := time.Parse(time.RFC3339, taken)
	if err != nil {
		panic(err)
	}
	return fixtureDoc(name, withMime("image/jpeg"), withPhoto(&libregraph.Photo{TakenDateTime: &t}))
}

func sortGroups() []sortGroup {
	return []sortGroup{
		{
			name: "sorting",
			fixtures: []search.Resource{
				track("b.mp3", 300, "2024-03-01T00:00:00Z", 200000, 1999),
				track("a.mp3", 100, "2024-01-01T00:00:00Z", 300000, 2005),
				track("C.mp3", 200, "2024-02-01T00:00:00Z", 100000, 1999),
				fixtureDoc("d.mp3", withMime("audio/mpeg"), withSize(200), withMtime("2024-04-01T00:00:00Z")),
				takenPhoto("x.jpg", "2020-06-01T12:00:00Z"),
				takenPhoto("y.jpg", "2010-06-01T12:00:00Z"),
				fixtureDoc("z.jpg", withMime("image/jpeg")),
				fixtureDoc("apple.txt"),
				fixtureDoc("Zebra.txt"),
				fixtureDoc("Übung.txt"),
			},
			cases: []sortCase{
				{id: 1, query: `mediatype:audio`, orderBy: []*searchService.SortProperty{asc("size")},
					reads: "by size, a tie in the order of the ids",
					want:  []string{"a.mp3", "C.mp3", "d.mp3", "b.mp3"}},
				{id: 2, query: `mediatype:audio`, orderBy: []*searchService.SortProperty{desc("size")},
					reads: "by size descending, a tie still in the order of the ids",
					want:  []string{"b.mp3", "C.mp3", "d.mp3", "a.mp3"}},
				{id: 3, query: `mediatype:audio`, orderBy: []*searchService.SortProperty{desc("lastModifiedDateTime")},
					reads: "newest first",
					want:  []string{"d.mp3", "b.mp3", "C.mp3", "a.mp3"}},
				{id: 4, query: `mediatype:audio`, orderBy: []*searchService.SortProperty{asc("audio.duration")},
					reads: "shortest track first, a file without a duration last",
					want:  []string{"C.mp3", "b.mp3", "a.mp3", "d.mp3"}},
				{id: 5, query: `mediatype:audio`, orderBy: []*searchService.SortProperty{desc("audio.duration")},
					reads: "longest track first, a file without a duration still last",
					want:  []string{"a.mp3", "b.mp3", "C.mp3", "d.mp3"}},
				{id: 6, query: `mediatype:audio`, orderBy: []*searchService.SortProperty{asc("audio.year"), desc("audio.duration")},
					reads: "by year, within a year the longest track first",
					want:  []string{"b.mp3", "C.mp3", "a.mp3", "d.mp3"}},
				{id: 7, query: `name:"*.txt"`, orderBy: []*searchService.SortProperty{asc("name")},
					reads: "by name in code point order: upper case before lower case, an umlaut after z",
					want:  []string{"Zebra.txt", "apple.txt", "Übung.txt"}},
				{id: 8, query: `mediatype:image`, orderBy: []*searchService.SortProperty{desc("photo.takenDateTime")},
					reads: "latest photo first, a file without a date last",
					want:  []string{"x.jpg", "y.jpg", "z.jpg"}},
				{id: 9, query: `mediatype:image`, orderBy: []*searchService.SortProperty{asc("photo.takenDateTime")},
					reads: "earliest photo first, a file without a date still last",
					want:  []string{"y.jpg", "x.jpg", "z.jpg"}},
				{id: 10, query: `mediatype:audio`, orderBy: []*searchService.SortProperty{asc("Tags")},
					reads: "a field with several values is not sortable", wantBadRequest: true},
				{id: 11, query: `mediatype:audio`, orderBy: []*searchService.SortProperty{asc("nonexistent")},
					reads: "an unknown field is not sortable", wantBadRequest: true},
			},
		},
	}
}

var _ = Describe("Sorting", func() {
	for groupAt, group := range sortGroups() {
		Describe(group.name, Ordered, ContinueOnFailure, func() {
			var engines []testEngine

			BeforeAll(func() {
				engines = newEngines("opencloud-test-engine-parity-sort-"+group.name, group.fixtures)
			})

			for caseAt, c := range group.cases {
				row := matrixRow{
					Section: "Sorting", Group: group.name, ID: c.label(),
					Query: c.query, Reads: c.reads,
					Want: c.want, WantBadRequest: c.wantBadRequest, Ordered: true,
					GroupAt: 200 + groupAt, CaseAt: caseAt,
				}
				planRow(row)

				Describe(c.label()+" "+c.reads, func() {
					for _, name := range engineNames {
						It("on "+name, func() {
							e := engineNamed(engines, name)
							if e.unavailable != "" {
								recordSkip(row, name)
								Skip(e.unavailable)
							}

							resp, err := e.backend.Search(context.Background(), &search.IndexQuery{Query: c.query, OrderBy: c.orderBy})
							if c.wantBadRequest {
								recordAnswer(row, name, badRequestAnswer(err))
								Expect(err).To(BeAssignableToTypeOf(errtypes.BadRequest("")))
								return
							}

							Expect(err).NotTo(HaveOccurred(), "the query has to answer")
							answer := make([]string, 0, len(resp.Matches))
							for _, m := range resp.Matches {
								answer = append(answer, m.GetEntity().GetName())
							}
							recordAnswer(row, name, answer)
							Expect(answer).To(Equal(c.want))
						})
					}
				})
			}
		})
	}
})
