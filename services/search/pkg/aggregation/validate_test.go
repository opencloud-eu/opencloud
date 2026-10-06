package aggregation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

// fieldTypes stands in for the index mapping.
func fieldTypes(field string) string {
	return map[string]string{
		"audio.artist":        mapping.TypeKeyword,
		"audio.album":         mapping.TypeKeyword,
		"audio.hasDrm":        mapping.TypeBool,
		"audio.year":          mapping.TypeNumeric,
		"photo.takenDateTime": mapping.TypeDatetime,
		"Content":             mapping.TypeFulltext,
		"Path":                mapping.TypePath,
		"audio":               mapping.TypeObject,
		"location":            mapping.TypeGeopoint,
	}[field]
}

var _ = Describe("ValidateOptions", func() {
	numeric := &searchsvc.BucketDefinition{Ranges: []*searchsvc.BucketRange{{From: "1980", To: "1990"}}}
	dates := &searchsvc.BucketDefinition{Ranges: []*searchsvc.BucketRange{{From: "2018-08-11T00:00:00Z"}}}
	sum := &searchsvc.MetricDefinition{Kind: searchsvc.MetricKind_METRIC_KIND_SUM}

	DescribeTable("accepts an aggregation that fits its field",
		func(opt *searchsvc.AggregationOption) {
			Expect(aggregation.ValidateOptions([]*searchsvc.AggregationOption{opt}, fieldTypes)).To(Succeed())
		},
		Entry("terms on a keyword", &searchsvc.AggregationOption{Field: "audio.artist"}),
		Entry("terms on a bool", &searchsvc.AggregationOption{Field: "audio.hasDrm"}),
		Entry("terms on a number", &searchsvc.AggregationOption{Field: "audio.year"}),
		Entry("numeric ranges on a number", &searchsvc.AggregationOption{Field: "audio.year", BucketDefinition: numeric}),
		Entry("date ranges on a date", &searchsvc.AggregationOption{Field: "photo.takenDateTime", BucketDefinition: dates}),
		Entry("a metric on a number", &searchsvc.AggregationOption{Field: "audio.year", MetricDefinition: sum}),
		Entry("a tree of them", &searchsvc.AggregationOption{Field: "audio.artist", SubAggregations: []*searchsvc.AggregationOption{
			{Field: "audio.year", BucketDefinition: numeric, SubAggregations: []*searchsvc.AggregationOption{
				{Field: "audio.album"}, {Field: "audio.year", MetricDefinition: sum},
			}},
		}}),
	)

	DescribeTable("rejects an aggregation the index cannot answer",
		func(opt *searchsvc.AggregationOption) {
			Expect(aggregation.ValidateOptions([]*searchsvc.AggregationOption{opt}, fieldTypes)).ToNot(Succeed())
		},
		Entry("an unknown field", &searchsvc.AggregationOption{Field: "audio.nonexistent"}),
		Entry("terms on a date", &searchsvc.AggregationOption{Field: "photo.takenDateTime"}),
		Entry("terms on full text", &searchsvc.AggregationOption{Field: "Content"}),
		Entry("terms on a path", &searchsvc.AggregationOption{Field: "Path"}),
		Entry("terms on a facet object", &searchsvc.AggregationOption{Field: "audio"}),
		Entry("terms on a geopoint", &searchsvc.AggregationOption{Field: "location"}),
		Entry("a metric on a keyword", &searchsvc.AggregationOption{Field: "audio.artist", MetricDefinition: sum}),
		Entry("a metric on a date", &searchsvc.AggregationOption{Field: "photo.takenDateTime", MetricDefinition: sum}),
		Entry("a metric with sub-aggregations", &searchsvc.AggregationOption{Field: "audio.year", MetricDefinition: sum,
			SubAggregations: []*searchsvc.AggregationOption{{Field: "audio.artist"}}}),
		Entry("numeric ranges on a keyword", &searchsvc.AggregationOption{Field: "audio.artist", BucketDefinition: numeric}),
		Entry("numeric ranges on a date", &searchsvc.AggregationOption{Field: "photo.takenDateTime", BucketDefinition: numeric}),
		Entry("date ranges on a number", &searchsvc.AggregationOption{Field: "audio.year", BucketDefinition: dates}),
		Entry("an invalid aggregation two levels down", &searchsvc.AggregationOption{Field: "audio.artist", SubAggregations: []*searchsvc.AggregationOption{
			{Field: "audio.album", SubAggregations: []*searchsvc.AggregationOption{{Field: "photo.takenDateTime"}}},
		}}),
	)
})
