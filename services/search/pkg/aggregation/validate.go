package aggregation

import (
	"fmt"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

// ValidateOptions holds aggregations to what the index can answer: a field it
// knows, a kind that fits the type of the field. fieldType returns the mapping
// type of a field, empty for one a client may not aggregate on.
func ValidateOptions(opts []*searchsvc.AggregationOption, fieldType func(string) string) error {
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
		case KindTerms:
			if err := validateTermsField(field, t); err != nil {
				return err
			}
		}
		if err := ValidateOptions(opt.GetSubAggregations(), fieldType); err != nil {
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
