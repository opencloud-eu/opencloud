package aggregation

import (
	"fmt"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

// ValidateOptions holds aggregations to what the index can answer: a field it
// knows, of a type whose values are buckets. fieldType returns the mapping
// type of a field, empty for one a client may not aggregate on.
func ValidateOptions(opts []*searchsvc.AggregationOption, fieldType func(string) string) error {
	for _, opt := range opts {
		field, t := opt.GetField(), fieldType(opt.GetField())
		if t == "" {
			return fmt.Errorf("unknown aggregation field %q", field)
		}
		if err := validateTermsField(field, t); err != nil {
			return err
		}
	}
	return nil
}

// validateTermsField allows the types whose values are buckets as they are.
// The values of a date are instants, no bucket each.
func validateTermsField(field, fieldType string) error {
	switch fieldType {
	case mapping.TypeKeyword, mapping.TypeBool, mapping.TypeNumeric:
		return nil
	case mapping.TypeDatetime:
		return fmt.Errorf("terms aggregation is not supported on date field %q", field)
	}
	return fmt.Errorf("terms aggregation is not supported on field %q", field)
}
