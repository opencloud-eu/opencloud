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
	ranges := func(rs ...*searchService.BucketRange) *searchService.BucketDefinition {
		return &searchService.BucketDefinition{Ranges: rs}
	}

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
		{id: 4, query: "mediatype:audio", reads: "audio.year buckets per decade",
			aggs: []*searchService.AggregationOption{{Field: "audio.year", BucketDefinition: ranges(
				&searchService.BucketRange{From: "1970", To: "1980"},
				&searchService.BucketRange{From: "1980", To: "1990"},
				&searchService.BucketRange{From: "1990", To: "2000"},
				&searchService.BucketRange{From: "2000", To: "2010"},
			)}},
			want: []string{"audio.year 1970-1980=2", "audio.year 1980-1990=1", "audio.year 1990-2000=1", "audio.year 2000-2010=3"}},
		{id: 5, query: "mediatype:audio", reads: "open-ended audio.year ranges",
			aggs: []*searchService.AggregationOption{{Field: "audio.year", BucketDefinition: ranges(
				&searchService.BucketRange{To: "1990"},
				&searchService.BucketRange{From: "2000"},
			)}},
			want: []string{"audio.year -1990=3", "audio.year 2000-=3"}},
		{id: 6, query: "mediatype:audio", reads: "top-level metrics on audio.year",
			aggs: []*searchService.AggregationOption{
				{Field: "audio.year", MetricKind: searchService.MetricKind_METRIC_KIND_SUM},
				{Field: "audio.year", MetricKind: searchService.MetricKind_METRIC_KIND_MIN},
				{Field: "audio.year", MetricKind: searchService.MetricKind_METRIC_KIND_MAX},
				{Field: "audio.year", MetricKind: searchService.MetricKind_METRIC_KIND_AVG},
			},
			want: []string{"audio.year sum=13942", "audio.year min=1971", "audio.year max=2009", "audio.year avg sum=13942 count=7"}},
		{id: 7, query: "mediatype:image", reads: "photo.takenDateTime buckets per date range",
			aggs: []*searchService.AggregationOption{{Field: "photo.takenDateTime", BucketDefinition: ranges(
				&searchService.BucketRange{From: "2018-08-11T00:00:00Z", To: "2018-08-12T00:00:00Z"},
				&searchService.BucketRange{From: "2018-08-01", To: "2018-09-01"},
				&searchService.BucketRange{From: "2021-01-01", To: "2022-01-01"},
				&searchService.BucketRange{From: "2023-01-01", To: "2024-01-01"},
			)}},
			want: []string{
				"photo.takenDateTime 2018-08-11T00:00:00Z-2018-08-12T00:00:00Z=2",
				"photo.takenDateTime 2018-08-01-2018-09-01=2",
				"photo.takenDateTime 2021-01-01-2022-01-01=1",
			}},
		{id: 8, query: "mediatype:image", reads: "open-ended date ranges",
			aggs: []*searchService.AggregationOption{{Field: "photo.takenDateTime", BucketDefinition: ranges(
				&searchService.BucketRange{To: "2019-01-01"},
				&searchService.BucketRange{From: "2019-01-01"},
			)}},
			want: []string{"photo.takenDateTime -2019-01-01=3", "photo.takenDateTime 2019-01-01-=1"}},
		{id: 9, query: "mediatype:image", reads: "malformed date range bound",
			aggs: []*searchService.AggregationOption{{Field: "photo.takenDateTime", BucketDefinition: ranges(
				&searchService.BucketRange{From: "2018-08-11T00:00:00Z", To: "not-a-date"},
			)}},
			wantError: true, want: []string{"error"}},
	}
}

// renderAggregations flattens an answer into comparable strings; buckets with
// no hits are dropped, the engines differ in whether they emit them at all.
func renderAggregations(resp *searchService.SearchIndexResponse, err error) []string {
	if err != nil {
		return []string{"error"}
	}

	out := []string{}
	for _, a := range resp.Aggregations {
		if a.MetricKind != searchService.MetricKind_METRIC_KIND_UNSPECIFIED {
			kind := strings.ToLower(strings.TrimPrefix(a.MetricKind.String(), "METRIC_KIND_"))
			if a.MetricKind == searchService.MetricKind_METRIC_KIND_AVG {
				out = append(out, fmt.Sprintf("%s avg sum=%v count=%d", a.Field, a.Sum, a.Count))
				continue
			}
			out = append(out, fmt.Sprintf("%s %s=%v", a.Field, kind, a.Value))
			continue
		}

		for _, b := range a.Buckets {
			if b.Count == 0 {
				continue
			}
			out = append(out, fmt.Sprintf("%s %s=%d", a.Field, b.Key, b.Count))
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
