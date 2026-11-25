package aggregation

import (
	"fmt"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

// ValidateOptions holds aggregations to MaxAggregations and MaxDepth and to
// what the index can answer: a field it knows, a kind that fits the type of
// the field. fieldType returns the mapping type of a field, empty for one a
// client may not aggregate on.
func ValidateOptions(opts []*searchsvc.AggregationOption, fieldType func(string) string) error {
	count, depth := shape(opts)
	if count > MaxAggregations {
		return fmt.Errorf("a request takes at most %d aggregations, sub-aggregations included", MaxAggregations)
	}
	if depth > MaxDepth {
		return fmt.Errorf("aggregations nest at most %d levels deep", MaxDepth)
	}
	return validateOptions(opts, fieldType)
}

func shape(opts []*searchsvc.AggregationOption) (count, depth int) {
	for _, opt := range opts {
		subCount, subDepth := shape(opt.GetSubAggregations())
		if KindOf(opt) != KindMetric {
			subDepth++
		}
		count += 1 + subCount
		depth = max(depth, subDepth)
	}
	return count, depth
}

func validateOptions(opts []*searchsvc.AggregationOption, fieldType func(string) string) error {
	for _, opt := range opts {
		field, t := opt.GetField(), fieldType(opt.GetField())
		if t == "" {
			return fmt.Errorf("unknown aggregation field %q", field)
		}
		switch KindOf(opt) {
		case KindMetric:
			if t != mapping.TypeNumeric {
				return fmt.Errorf("metric aggregation needs a numeric field, %q is none", field)
			}
			if len(opt.GetSubAggregations()) > 0 {
				return fmt.Errorf("metric aggregation on %q has no buckets to nest sub-aggregations in", field)
			}
		case KindRange:
			parsed, err := ParseRanges(field, opt.GetBucketDefinition().GetRanges())
			if err != nil {
				return err
			}
			if err := validateRangeField(field, t, parsed); err != nil {
				return err
			}
		case KindGeohash:
			if t != mapping.TypeGeopoint {
				return fmt.Errorf("geohash aggregation needs a geopoint field, %q is none", field)
			}
			if p := opt.GetGeohashDefinition().GetPrecision(); p < 1 || p > MaxGeohashPrecision {
				return fmt.Errorf("geohash precision %d on %q is out of range 1-%d", p, field, MaxGeohashPrecision)
			}
		case KindTerms:
			if err := validateTermsField(field, t); err != nil {
				return err
			}
		}
		if err := validateOptions(opt.GetSubAggregations(), fieldType); err != nil {
			return err
		}
	}
	return nil
}

// validateTermsField allows the types whose values are buckets as they are.
// The values of a date are instants, they aggregate in ranges.
func validateTermsField(field, fieldType string) error {
	switch fieldType {
	case mapping.TypeKeyword, mapping.TypeBool, mapping.TypeNumeric:
		return nil
	case mapping.TypeDatetime:
		return fmt.Errorf("terms aggregation is not supported on date field %q; use bucketDefinition.ranges", field)
	}
	return fmt.Errorf("terms aggregation is not supported on field %q", field)
}

func validateRangeField(field, fieldType string, ranges Ranges) error {
	if len(ranges.Numeric) > 0 && fieldType != mapping.TypeNumeric {
		return fmt.Errorf("numeric ranges need a numeric field, %q is none", field)
	}
	if len(ranges.Dates) > 0 && fieldType != mapping.TypeDatetime {
		return fmt.Errorf("date ranges need a date field, %q is none", field)
	}
	return nil
}

// ValidateFilters holds filters to the same rules as ValidateOptions.
func ValidateFilters(filters []*searchsvc.AggregationFilter, fieldType func(string) string) error {
	for _, f := range filters {
		if _, err := ParseFilter(f, fieldType(f.GetField())); err != nil {
			return err
		}
	}
	return nil
}
