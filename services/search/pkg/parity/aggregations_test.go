package parity

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// aggCase is one aggregation request both engines have to answer alike. The
// answer is rendered to strings (one per non-empty bucket or metric), so the
// matrix machinery can carry it like any query answer.
type aggCase struct {
	id    int
	query string
	aggs  []*searchService.AggregationOption
	// filters narrow the matches to buckets; the answer then starts with the
	// names of the matches
	filters         []*searchService.AggregationFilter
	reads           string
	want            []string
	wantCount       *int
	wantBadRequest  bool
	engineOverrides map[string]override
	// pageSize, when set, prefixes the answer with "<returned> of <total> matches"
	pageSize int32
	// listMatches puts the names of the matches into the answer and holds the
	// engines to their order
	listMatches bool
}

// aggGroup is a set of cases over one set of fixtures.
type aggGroup struct {
	name     string
	fixtures []search.Resource
	cases    []aggCase
}

func aggregationGroups() []aggGroup {
	return []aggGroup{
		{name: "aggregations", fixtures: aggregationFixtures(), cases: aggregationCases()},
		{name: "cardinality", fixtures: cardinalityFixtures(), cases: cardinalityCases()},
	}
}

func (c aggCase) label() string { return fmt.Sprintf("AGG-%02d", c.id) }

func untaggedSong(name string, year int32) search.Resource {
	return fixtureDoc(name, withMime("audio/mpeg"), withAudio(&libregraph.Audio{Year: libregraph.PtrInt32(year)}))
}

func photo(name, taken string) search.Resource {
	t, err := time.Parse(time.RFC3339, taken)
	if err != nil {
		panic(err)
	}
	return fixtureDoc(name, withMime("image/jpeg"), withPhoto(&libregraph.Photo{TakenDateTime: &t}))
}

func song(name, artist, album string, year int32, opts ...fixtureOption) search.Resource {
	return fixtureDoc(name, append([]fixtureOption{withMime("audio/mpeg"), withAudio(&libregraph.Audio{
		Artist: libregraph.PtrString(artist),
		Album:  libregraph.PtrString(album),
		Year:   libregraph.PtrInt32(year),
	})}, opts...)...)
}

func withDrm(r search.Resource, drm bool) search.Resource {
	r.Audio.HasDrm = &drm
	return r
}

func aggregationFixtures() []search.Resource {
	return []search.Resource{
		// years: 1971, 1975, 1982, 1999, 2001, 2005, 2009
		withDrm(song("a.mp3", "Saxon", "Wheels of Steel", 1971, withTags("live", "remaster")), true),
		withDrm(song("b.mp3", "Saxon", "Wheels of Steel", 1975, withTags("live")), false),
		song("c.mp3", "Motörhead", "Bomber", 1982),
		song("d.mp3", "Motörhead", "Bomber", 1999),
		song("e.mp3", "Motörhead", "Ace of Spades", 2001),
		untaggedSong("f.mp3", 2005),
		untaggedSong("g.mp3", 2009),
		photo("a.jpg", "2018-08-11T09:15:00Z"),
		photo("b.jpg", "2018-08-11T19:42:00Z"),
		photo("c.jpg", "2018-09-01T12:00:00Z"),
		photo("d.jpg", "2021-08-11T08:00:00Z"),
		// no audio mime type: stays out of the mediatype:audio cases
		fixtureDoc("live.txt", withAudio(&libregraph.Audio{Artist: libregraph.PtrString(`Wh*t? "Live"`)})),
	}
}

func aggregationCases() []aggCase {
	ranges := func(rs ...*searchService.BucketRange) *searchService.BucketDefinition {
		return &searchService.BucketDefinition{Ranges: rs}
	}
	metric := func(field string, kind searchService.MetricKind) *searchService.AggregationOption {
		return &searchService.AggregationOption{Field: field, MetricDefinition: &searchService.MetricDefinition{Kind: kind}}
	}
	terms := func(field string, keys ...string) *searchService.AggregationFilter {
		return &searchService.AggregationFilter{Field: field, Terms: keys}
	}
	within := func(field string, rs ...*searchService.BucketRange) *searchService.AggregationFilter {
		return &searchService.AggregationFilter{Field: field, Ranges: rs}
	}

	return []aggCase{
		{id: 1, query: "mediatype:audio", reads: "term buckets on audio.artist",
			aggs: []*searchService.AggregationOption{{Field: "audio.artist", Size: 10}},
			want: []string{"audio.artist Saxon=2", "audio.artist Motörhead=3"}},
		{id: 2, query: "mediatype:audio", reads: "no aggregations requested"},
		{id: 3, query: "mediatype:audio", reads: "artist and album buckets in one request",
			aggs: []*searchService.AggregationOption{{Field: "audio.artist"}, {Field: "audio.album"}},
			want: []string{
				"audio.artist Saxon=2", "audio.artist Motörhead=3",
				"audio.album Wheels of Steel=2", "audio.album Bomber=2", "audio.album Ace of Spades=1",
			}},
		{id: 4, query: "mediatype:audio", reads: "audio.year buckets per decade",
			aggs: []*searchService.AggregationOption{{Field: "audio.year", BucketDefinition: ranges(
				&searchService.BucketRange{From: "1970", To: "1980"},
				&searchService.BucketRange{From: "1980", To: "1990"},
				&searchService.BucketRange{From: "1990", To: "2000"},
				&searchService.BucketRange{From: "2000", To: "2010"},
			)}},
			want: []string{"audio.year 1970..1980=2", "audio.year 1980..1990=1", "audio.year 1990..2000=1", "audio.year 2000..2010=3"}},
		{id: 5, query: "mediatype:audio", reads: "open-ended audio.year ranges",
			aggs: []*searchService.AggregationOption{{Field: "audio.year", BucketDefinition: ranges(
				&searchService.BucketRange{To: "1990"},
				&searchService.BucketRange{From: "2000"},
			)}},
			want: []string{"audio.year ..1990=3", "audio.year 2000..=3"}},
		{id: 6, query: "mediatype:audio", reads: "top-level metrics on audio.year",
			aggs: []*searchService.AggregationOption{
				metric("audio.year", searchService.MetricKind_METRIC_KIND_SUM),
				metric("audio.year", searchService.MetricKind_METRIC_KIND_MIN),
				metric("audio.year", searchService.MetricKind_METRIC_KIND_MAX),
				metric("audio.year", searchService.MetricKind_METRIC_KIND_AVG),
			},
			want: []string{"audio.year sum=13942", "audio.year min=1971", "audio.year max=2009", "audio.year avg sum=13942 count=7"}},
		{id: 7, query: "mediatype:image", reads: "photo.takenDateTime buckets per date range, an empty range counts zero",
			aggs: []*searchService.AggregationOption{{Field: "photo.takenDateTime", BucketDefinition: ranges(
				&searchService.BucketRange{From: "2018-08-11T00:00:00Z", To: "2018-08-12T00:00:00Z"},
				&searchService.BucketRange{From: "2018-08-01T00:00:00Z", To: "2018-09-01T00:00:00Z"},
				&searchService.BucketRange{From: "2021-01-01T00:00:00Z", To: "2022-01-01T00:00:00Z"},
				&searchService.BucketRange{From: "2023-01-01T00:00:00Z", To: "2024-01-01T00:00:00Z"},
			)}},
			want: []string{
				"photo.takenDateTime 2018-08-11T00:00:00Z..2018-08-12T00:00:00Z=2",
				"photo.takenDateTime 2018-08-01T00:00:00Z..2018-09-01T00:00:00Z=2",
				"photo.takenDateTime 2021-01-01T00:00:00Z..2022-01-01T00:00:00Z=1",
				"photo.takenDateTime 2023-01-01T00:00:00Z..2024-01-01T00:00:00Z=0",
			}},
		{id: 8, query: "mediatype:image", reads: "open-ended date ranges",
			aggs: []*searchService.AggregationOption{{Field: "photo.takenDateTime", BucketDefinition: ranges(
				&searchService.BucketRange{To: "2019-01-01T00:00:00Z"},
				&searchService.BucketRange{From: "2019-01-01T00:00:00Z"},
			)}},
			want: []string{"photo.takenDateTime ..2019-01-01T00:00:00Z=3", "photo.takenDateTime 2019-01-01T00:00:00Z..=1"}},
		{id: 9, query: "mediatype:image", reads: "malformed date range bound",
			aggs: []*searchService.AggregationOption{{Field: "photo.takenDateTime", BucketDefinition: ranges(
				&searchService.BucketRange{From: "2018-08-11T00:00:00Z", To: "not-a-date"},
			)}},
			wantBadRequest: true},
		{id: 10, query: "mediatype:audio", reads: "album buckets nested in artist buckets",
			aggs: []*searchService.AggregationOption{{Field: "audio.artist", SubAggregations: []*searchService.AggregationOption{{Field: "audio.album"}}}},
			want: []string{
				"audio.artist Saxon=2", "audio.artist Saxon=2 / audio.album Wheels of Steel=2",
				"audio.artist Motörhead=3", "audio.artist Motörhead=3 / audio.album Bomber=2", "audio.artist Motörhead=3 / audio.album Ace of Spades=1",
			}},
		{id: 11, query: "mediatype:audio", reads: "sum and avg of audio.year per artist",
			aggs: []*searchService.AggregationOption{{Field: "audio.artist", SubAggregations: []*searchService.AggregationOption{
				metric("audio.year", searchService.MetricKind_METRIC_KIND_SUM),
				metric("audio.year", searchService.MetricKind_METRIC_KIND_AVG),
			}}},
			want: []string{
				"audio.artist Saxon=2", "audio.artist Saxon=2 / audio.year sum=3946", "audio.artist Saxon=2 / audio.year avg sum=3946 count=2",
				"audio.artist Motörhead=3", "audio.artist Motörhead=3 / audio.year sum=5982", "audio.artist Motörhead=3 / audio.year avg sum=5982 count=3",
			}},
		{id: 12, query: "mediatype:audio", reads: "artist buckets nested in audio.year decades",
			aggs: []*searchService.AggregationOption{{Field: "audio.year", BucketDefinition: ranges(
				&searchService.BucketRange{From: "1970", To: "1980"},
				&searchService.BucketRange{From: "1980", To: "1990"},
				&searchService.BucketRange{From: "1990", To: "2000"},
				&searchService.BucketRange{From: "2000", To: "2010"},
			), SubAggregations: []*searchService.AggregationOption{{Field: "audio.artist"}}}},
			want: []string{
				"audio.year 1970..1980=2", "audio.year 1970..1980=2 / audio.artist Saxon=2",
				"audio.year 1980..1990=1", "audio.year 1980..1990=1 / audio.artist Motörhead=1",
				"audio.year 1990..2000=1", "audio.year 1990..2000=1 / audio.artist Motörhead=1",
				"audio.year 2000..2010=3", "audio.year 2000..2010=3 / audio.artist Motörhead=1",
			}},
		{id: 13, query: "mediatype:audio", reads: "max audio.year per album per artist, three levels",
			aggs: []*searchService.AggregationOption{{Field: "audio.artist", SubAggregations: []*searchService.AggregationOption{
				{Field: "audio.album", SubAggregations: []*searchService.AggregationOption{
					metric("audio.year", searchService.MetricKind_METRIC_KIND_MAX),
				}},
			}}},
			want: []string{
				"audio.artist Saxon=2", "audio.artist Saxon=2 / audio.album Wheels of Steel=2", "audio.artist Saxon=2 / audio.album Wheels of Steel=2 / audio.year max=1975",
				"audio.artist Motörhead=3", "audio.artist Motörhead=3 / audio.album Bomber=2", "audio.artist Motörhead=3 / audio.album Bomber=2 / audio.year max=1999",
				"audio.artist Motörhead=3 / audio.album Ace of Spades=1", "audio.artist Motörhead=3 / audio.album Ace of Spades=1 / audio.year max=2001",
			}},
		{id: 14, query: "mediatype:image", reads: "MimeType buckets nested in open-ended date ranges",
			aggs: []*searchService.AggregationOption{{Field: "photo.takenDateTime", BucketDefinition: ranges(
				&searchService.BucketRange{To: "2019-01-01T00:00:00Z"},
				&searchService.BucketRange{From: "2019-01-01T00:00:00Z"},
			), SubAggregations: []*searchService.AggregationOption{{Field: "MimeType"}}}},
			want: []string{
				"photo.takenDateTime ..2019-01-01T00:00:00Z=3", "photo.takenDateTime ..2019-01-01T00:00:00Z=3 / MimeType image/jpeg=3",
				"photo.takenDateTime 2019-01-01T00:00:00Z..=1", "photo.takenDateTime 2019-01-01T00:00:00Z..=1 / MimeType image/jpeg=1",
			},
			// OpenSearch maps MimeType as wildcard, which serves no doc values to
			// aggregate on; the follow-up is an aggregatable keyword sibling
			engineOverrides: map[string]override{"opensearch": {want: []string{"error"}}}},
		{id: 15, query: "mediatype:audio", reads: "nested aggregations cover every match on a page of one",
			pageSize: 1,
			aggs: []*searchService.AggregationOption{
				{Field: "audio.artist", SubAggregations: []*searchService.AggregationOption{{Field: "audio.album"}}},
				metric("audio.year", searchService.MetricKind_METRIC_KIND_SUM),
			},
			want: []string{
				"1 of 7 matches",
				"audio.artist Saxon=2", "audio.artist Saxon=2 / audio.album Wheels of Steel=2",
				"audio.artist Motörhead=3", "audio.artist Motörhead=3 / audio.album Bomber=2", "audio.artist Motörhead=3 / audio.album Ace of Spades=1",
				"audio.year sum=13942",
			}},
		// 16 and 19 pinned malformed bounds twice over, 17 is the cardinality case below
		{id: 18, query: "mediatype:audio", reads: "two range aggregations and a metric on audio.year stay apart",
			aggs: []*searchService.AggregationOption{
				{Field: "audio.year", BucketDefinition: ranges(&searchService.BucketRange{To: "1980"}, &searchService.BucketRange{From: "1980"})},
				{Field: "audio.year", BucketDefinition: ranges(&searchService.BucketRange{To: "2000"}, &searchService.BucketRange{From: "2000"})},
				metric("audio.year", searchService.MetricKind_METRIC_KIND_MAX),
			},
			want: []string{
				"audio.year ..1980=2", "audio.year 1980..=5",
				"audio.year ..2000=4", "audio.year 2000..=3",
				"audio.year max=2009",
			}},
		{id: 20, query: "mediatype:audio", reads: "album buckets in audio.year ranges in artist buckets",
			aggs: []*searchService.AggregationOption{{Field: "audio.artist", SubAggregations: []*searchService.AggregationOption{{
				Field:            "audio.year",
				BucketDefinition: ranges(&searchService.BucketRange{To: "1980"}, &searchService.BucketRange{From: "1980"}),
				SubAggregations:  []*searchService.AggregationOption{{Field: "audio.album"}},
			}}}},
			want: []string{
				"audio.artist Saxon=2",
				"audio.artist Saxon=2 / audio.year ..1980=2", "audio.artist Saxon=2 / audio.year ..1980=2 / audio.album Wheels of Steel=2",
				"audio.artist Saxon=2 / audio.year 1980..=0",
				"audio.artist Motörhead=3",
				"audio.artist Motörhead=3 / audio.year ..1980=0",
				"audio.artist Motörhead=3 / audio.year 1980..=3",
				"audio.artist Motörhead=3 / audio.year 1980..=3 / audio.album Bomber=2", "audio.artist Motörhead=3 / audio.year 1980..=3 / audio.album Ace of Spades=1",
			}},
		{id: 21, query: "mediatype:image", reads: "a metric without a single value has none",
			aggs: []*searchService.AggregationOption{
				metric("audio.year", searchService.MetricKind_METRIC_KIND_SUM),
				metric("audio.year", searchService.MetricKind_METRIC_KIND_MIN),
				metric("audio.year", searchService.MetricKind_METRIC_KIND_AVG),
			},
			want: []string{"audio.year sum none", "audio.year min none", "audio.year avg none"}},
		{id: 22, query: "mediatype:audio", reads: "filtered to the Saxon bucket, as many matches as the bucket counted",
			filters: []*searchService.AggregationFilter{terms("audio.artist", "Saxon")},
			aggs:    []*searchService.AggregationOption{{Field: "audio.artist"}},
			want:    []string{"a.mp3", "b.mp3", "audio.artist Saxon=2"}},
		{id: 23, query: "mediatype:audio", reads: "filtered to saxon, a bucket key matches case-sensitively",
			filters: []*searchService.AggregationFilter{terms("audio.artist", "saxon")}},
		{id: 24, query: "mediatype:audio", reads: "filtered to the Saxon or the Motörhead bucket",
			filters: []*searchService.AggregationFilter{terms("audio.artist", "Saxon", "Motörhead")},
			want:    []string{"a.mp3", "b.mp3", "c.mp3", "d.mp3", "e.mp3"}},
		{id: 25, query: "mediatype:audio", reads: "filtered to audio.year 1980-1990",
			filters: []*searchService.AggregationFilter{within("audio.year", &searchService.BucketRange{From: "1980", To: "1990"})},
			want:    []string{"c.mp3"}},
		{id: 26, query: "mediatype:audio", reads: "filtered to audio.year 1982-1999, from-inclusive and to-exclusive",
			filters: []*searchService.AggregationFilter{within("audio.year", &searchService.BucketRange{From: "1982", To: "1999"})},
			want:    []string{"c.mp3"}},
		{id: 27, query: "mediatype:audio", reads: "filtered to audio.year 2000-",
			filters: []*searchService.AggregationFilter{within("audio.year", &searchService.BucketRange{From: "2000"})},
			want:    []string{"e.mp3", "f.mp3", "g.mp3"}},
		{id: 28, query: "mediatype:audio", reads: "filtered to audio.year -1980 or 2005-",
			filters: []*searchService.AggregationFilter{within("audio.year", &searchService.BucketRange{To: "1980"}, &searchService.BucketRange{From: "2005"})},
			want:    []string{"a.mp3", "b.mp3", "f.mp3", "g.mp3"}},
		{id: 29, query: "mediatype:image", reads: "filtered to photo.takenDateTime on 2018-08-11",
			filters: []*searchService.AggregationFilter{within("photo.takenDateTime", &searchService.BucketRange{From: "2018-08-11T00:00:00Z", To: "2018-08-12T00:00:00Z"})},
			want:    []string{"a.jpg", "b.jpg"}},
		{id: 30, query: "mediatype:audio", reads: "filtered to the Motörhead bucket and to audio.year 1990-2010",
			filters: []*searchService.AggregationFilter{
				terms("audio.artist", "Motörhead"),
				within("audio.year", &searchService.BucketRange{From: "1990", To: "2010"}),
			},
			want: []string{"d.mp3", "e.mp3"}},
		{id: 31, query: "*", reads: `filtered to the bucket Wh*t? "Live", a key is data`,
			filters: []*searchService.AggregationFilter{terms("audio.artist", `Wh*t? "Live"`)},
			want:    []string{"live.txt"}},
		{id: 32, query: "mediatype:audio", reads: "filtered to Sax*, a key is no wildcard pattern",
			filters: []*searchService.AggregationFilter{terms("audio.artist", "Sax*")}},
		{id: 33, query: "mediatype:audio", reads: "filtered to a malformed range",
			filters:        []*searchService.AggregationFilter{within("audio.year", &searchService.BucketRange{From: "1980 OR Name:*", To: "1990"})},
			wantBadRequest: true},
		{id: 34, query: "mediatype:audio", reads: "term buckets on the numeric audio.year",
			aggs: []*searchService.AggregationOption{{Field: "audio.year"}},
			want: []string{
				"audio.year 1971=1", "audio.year 1975=1", "audio.year 1982=1", "audio.year 1999=1",
				"audio.year 2001=1", "audio.year 2005=1", "audio.year 2009=1",
			}},
		{id: 35, query: "mediatype:audio", reads: "audio.year buckets nested in artist buckets",
			aggs: []*searchService.AggregationOption{{Field: "audio.artist", SubAggregations: []*searchService.AggregationOption{{Field: "audio.year"}}}},
			want: []string{
				"audio.artist Saxon=2", "audio.artist Saxon=2 / audio.year 1971=1", "audio.artist Saxon=2 / audio.year 1975=1",
				"audio.artist Motörhead=3", "audio.artist Motörhead=3 / audio.year 1982=1",
				"audio.artist Motörhead=3 / audio.year 1999=1", "audio.artist Motörhead=3 / audio.year 2001=1",
			}},
		{id: 36, query: "mediatype:audio", reads: "filtered to the audio.year bucket 1982",
			filters: []*searchService.AggregationFilter{terms("audio.year", "1982")},
			want:    []string{"c.mp3"}},
		{id: 37, query: "mediatype:audio", reads: "term buckets on the bool audio.hasDrm, spelled true and false",
			aggs: []*searchService.AggregationOption{{Field: "audio.hasDrm"}},
			want: []string{"audio.hasDrm true=1", "audio.hasDrm false=1"}},
		{id: 38, query: "mediatype:audio", reads: "filtered to the audio.hasDrm bucket true",
			filters: []*searchService.AggregationFilter{terms("audio.hasDrm", "true")},
			want:    []string{"a.mp3"}},
		{id: 39, query: "mediatype:audio", reads: "no bucket for the empty Title of every match",
			aggs: []*searchService.AggregationOption{{Field: "Title"}}},
		{id: 40, query: "mediatype:audio", reads: "no bucket for the empty Title, with sub-aggregations neither",
			aggs: []*searchService.AggregationOption{{Field: "Title", SubAggregations: []*searchService.AggregationOption{{Field: "audio.artist"}}}}},
		{id: 41, query: "mediatype:audio", reads: "matches of the same score in the order of their ids, page after page",
			pageSize: 3, listMatches: true,
			want: []string{"3 of 7 matches", "a.mp3", "b.mp3", "c.mp3"}},
		{id: 45, query: "mediatype:audio", reads: "filtered to the Tags bucket live, a filter narrows and does not rank",
			filters: []*searchService.AggregationFilter{terms("Tags", "live")}, listMatches: true,
			want: []string{"a.mp3", "b.mp3"}},
	}
}

// renderAggregations flattens an answer into comparable strings, one per
// bucket or metric. A nested result renders under its bucket, joined by " / ".
func renderAggregations(resp *searchService.SearchIndexResponse, err error) []string {
	if err != nil {
		if _, ok := err.(errtypes.BadRequest); ok {
			return []string{"bad request"}
		}
		return []string{"error"}
	}

	out := []string{}
	for _, a := range resp.Aggregations {
		out = append(out, renderAggregation("", a)...)
	}

	return out
}

func renderAggregation(prefix string, a *searchService.AggregationResult) []string {
	if m := a.GetMetric(); m != nil {
		kind := strings.ToLower(strings.TrimPrefix(m.GetKind().String(), "METRIC_KIND_"))
		value, ok := aggregation.MetricValue(m)
		switch {
		case !ok:
			return []string{prefix + fmt.Sprintf("%s %s none", a.Field, kind)}
		case m.GetKind() == searchService.MetricKind_METRIC_KIND_AVG:
			return []string{prefix + fmt.Sprintf("%s avg sum=%v count=%d", a.Field, m.GetSum(), m.GetCount())}
		}
		return []string{prefix + fmt.Sprintf("%s %s=%v", a.Field, kind, value)}
	}

	out := []string{}
	for _, b := range a.Buckets {
		line := prefix + fmt.Sprintf("%s %s=%d", a.Field, b.Key, b.Count)
		out = append(out, line)
		for _, sub := range b.SubAggregations {
			out = append(out, renderAggregation(line+" / ", sub)...)
		}
	}

	return out
}

// cardinalityFixtures exceed the composite page size and what OpenSearch
// counts and returns by default (10000 matches).
func cardinalityFixtures() []search.Resource {
	docs := make([]search.Resource, 0, 10050)
	for i := 0; i < 10050; i++ {
		docs = append(docs, song(fmt.Sprintf("card-%05d.mp3", i), fmt.Sprintf("artist-%05d", i), "Singles", int32(1950+i%50)))
	}
	return docs
}

// nested puts a terms aggregation on each field into the one before it.
func nested(fields ...string) *searchService.AggregationOption {
	opt := &searchService.AggregationOption{Field: fields[0]}
	if len(fields) > 1 {
		opt.SubAggregations = []*searchService.AggregationOption{nested(fields[1:]...)}
	}
	return opt
}

func cardinalityCases() []aggCase {
	count := 10050
	return []aggCase{
		{id: 17, query: "mediatype:audio", reads: "one bucket per artist, cardinality above the page size",
			aggs:      []*searchService.AggregationOption{{Field: "audio.artist"}},
			wantCount: &count},
		{id: 42, query: "mediatype:audio", reads: "the total counts every match",
			pageSize: 1,
			want:     []string{"1 of 10050 matches"}},
		{id: 43, query: "mediatype:audio", reads: "a page reaching beyond the first 10000 matches is refused",
			pageSize:       10001,
			wantBadRequest: true},
		{id: 44, query: "mediatype:audio", reads: "more than 65535 buckets in one request are refused",
			aggs:           []*searchService.AggregationOption{nested("audio.artist", "Name", "ID", "RootID", "ParentID", "audio.album", "audio.year")},
			wantBadRequest: true},
	}
}

var _ = Describe("Aggregations", func() {
	for groupAt, group := range aggregationGroups() {
		Describe(group.name, Ordered, ContinueOnFailure, func() {
			var engines []testEngine

			BeforeAll(func() {
				engines = newEngines("opencloud-test-engine-parity-agg-"+group.name, group.fixtures)
			})

			for caseAt, c := range group.cases {
				row := matrixRow{
					Section: "Aggregations", Group: group.name, ID: c.label(),
					Query: c.query, Reads: c.reads,
					Want: c.want, WantCount: c.wantCount, WantBadRequest: c.wantBadRequest, Overrides: renderOverrides(c.engineOverrides),
					GroupAt: 100 + groupAt, CaseAt: caseAt,
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

							request := &searchService.SearchIndexRequest{
								Query:              c.query,
								Aggregations:       c.aggs,
								AggregationFilters: c.filters,
							}
							if c.pageSize != 0 {
								request.PageSize = conversions.ToPointer(c.pageSize)
							}
							resp, err := e.backend.Search(context.Background(), request)
							answer := renderAggregations(resp, err)
							if err == nil && (len(c.filters) > 0 || c.listMatches) {
								names := make([]string, 0, len(resp.Matches))
								for _, m := range resp.Matches {
									names = append(names, m.GetEntity().GetName())
								}
								answer = append(names, answer...)
							}
							if err == nil && c.pageSize > 0 {
								answer = append([]string{fmt.Sprintf("%d of %d matches", len(resp.Matches), resp.TotalMatches)}, answer...)
							}
							recordAnswer(row, name, answer)

							_, overridden := c.engineOverrides[name]
							if !overridden && !c.wantBadRequest {
								Expect(err).NotTo(HaveOccurred(), "the aggregation has to answer")
							}

							if c.listMatches {
								Expect(answer).To(Equal(c.want))
								return
							}
							expectAnswer(name, answer, override{want: c.want, wantCount: c.wantCount, wantBadRequest: c.wantBadRequest}, c.engineOverrides)
						})
					}
				})
			}
		})
	}
})
