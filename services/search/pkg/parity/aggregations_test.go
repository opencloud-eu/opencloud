package parity

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// aggCase is one aggregation request both engines have to answer alike. The
// answer is rendered to strings (one per non-empty bucket or metric), so the
// matrix machinery can carry it like any query answer.
type aggCase struct {
	id              int
	query           string
	aggs            []*searchService.AggregationOption
	reads           string
	want            []string
	wantError       bool
	engineOverrides map[string]override
}

func (c aggCase) label() string { return fmt.Sprintf("AGG-%02d", c.id) }

func withYear(name string, year int32) search.Resource {
	return fixtureDoc(name, withMime("audio/mpeg"), withAudio(&libregraph.Audio{Year: libregraph.PtrInt32(year)}))
}

func withTaken(name, taken string) search.Resource {
	t, err := time.Parse(time.RFC3339, taken)
	if err != nil {
		panic(err)
	}
	return fixtureDoc(name, withMime("image/jpeg"), withPhoto(&libregraph.Photo{TakenDateTime: &t}))
}

func song(name, artist, album string, year int32) search.Resource {
	return fixtureDoc(name, withMime("audio/mpeg"), withAudio(&libregraph.Audio{
		Artist: libregraph.PtrString(artist),
		Album:  libregraph.PtrString(album),
		Year:   libregraph.PtrInt32(year),
	}))
}

func aggregationFixtures() []search.Resource {
	return []search.Resource{
		// years: 1971, 1975, 1982, 1999, 2001, 2005, 2009
		song("a.mp3", "Pink Floyd", "The Wall", 1971),
		song("b.mp3", "Pink Floyd", "The Wall", 1975),
		song("c.mp3", "Motörhead", "Bomber", 1982),
		song("d.mp3", "Motörhead", "Bomber", 1999),
		song("e.mp3", "Motörhead", "Ace of Spades", 2001),
		withYear("f.mp3", 2005),
		withYear("g.mp3", 2009),
		withTaken("a.jpg", "2018-08-11T09:15:00Z"),
		withTaken("b.jpg", "2018-08-11T19:42:00Z"),
		withTaken("c.jpg", "2018-09-01T12:00:00Z"),
		withTaken("d.jpg", "2021-08-11T08:00:00Z"),
	}
}

func aggregationCases() []aggCase {
	return []aggCase{
		{id: 1, query: "mediatype:audio", reads: "term buckets on audio.artist",
			aggs: []*searchService.AggregationOption{{Field: "audio.artist", Size: 10}},
			want: []string{"audio.artist Pink Floyd=2", "audio.artist Motörhead=3"}},
		{id: 2, query: "mediatype:audio", reads: "no aggregations requested"},
		{id: 3, query: "mediatype:audio", reads: "artist and album buckets in one request",
			aggs: []*searchService.AggregationOption{{Field: "audio.artist"}, {Field: "audio.album"}},
			want: []string{
				"audio.artist Pink Floyd=2", "audio.artist Motörhead=3",
				"audio.album The Wall=2", "audio.album Bomber=2", "audio.album Ace of Spades=1",
			}},
	}
}

// renderAggregations flattens an answer into comparable strings; buckets with
// no hits are dropped, the engines differ in whether they emit them at all.
// A nested result renders under its bucket, joined by " / ".
func renderAggregations(resp *searchService.SearchIndexResponse, err error) []string {
	if err != nil {
		return []string{"error"}
	}

	out := []string{}
	for _, a := range resp.Aggregations {
		out = append(out, renderAggregation("", a)...)
	}

	return out
}

func renderAggregation(prefix string, a *searchService.AggregationResult) []string {
	if a.MetricKind != searchService.MetricKind_METRIC_KIND_UNSPECIFIED {
		kind := strings.ToLower(strings.TrimPrefix(a.MetricKind.String(), "METRIC_KIND_"))
		if a.MetricKind == searchService.MetricKind_METRIC_KIND_AVG {
			return []string{prefix + fmt.Sprintf("%s avg sum=%v count=%d", a.Field, a.Sum, a.Count)}
		}
		return []string{prefix + fmt.Sprintf("%s %s=%v", a.Field, kind, a.Value)}
	}

	out := []string{}
	for _, b := range a.Buckets {
		if b.Count == 0 {
			continue
		}
		line := prefix + fmt.Sprintf("%s %s=%d", a.Field, b.Key, b.Count)
		out = append(out, line)
		for _, sub := range b.SubAggregations {
			out = append(out, renderAggregation(line+" / ", sub)...)
		}
	}

	return out
}

var _ = Describe("Aggregations", func() {
	Describe("aggregations", Ordered, ContinueOnFailure, func() {
		var engines []testEngine

		BeforeAll(func() {
			engines = newEngines("opencloud-test-engine-parity-aggregations", aggregationFixtures())
		})

		for caseAt, c := range aggregationCases() {
			row := matrixRow{
				Section: "Aggregations", Group: "aggregations", ID: c.label(),
				Query: c.query, Reads: c.reads,
				Want: c.want, Overrides: renderOverrides(c.engineOverrides),
				GroupAt: 100, CaseAt: caseAt,
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

						resp, err := e.backend.Search(context.Background(), &searchService.SearchIndexRequest{
							Query:        c.query,
							Aggregations: c.aggs,
						})
						answer := renderAggregations(resp, err)
						recordAnswer(row, name, answer)

						_, overridden := c.engineOverrides[name]
						if !overridden && !c.wantError {
							Expect(err).NotTo(HaveOccurred(), "the aggregation has to answer")
						}

						expectAnswer(name, answer, override{want: c.want}, c.engineOverrides)
					})
				}
			})
		}
	})
})
