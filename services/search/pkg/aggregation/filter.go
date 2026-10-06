package aggregation

import (
	"fmt"
	"strconv"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

// Buckets are the buckets an aggregation filter selects, read as the type of
// its field.
type Buckets struct {
	Terms  []Term
	Ranges Ranges
}

// Term is a bucket key of a terms aggregation. A numeric or bool field also
// carries the key as its value.
type Term struct {
	Key    string
	Number *float64
	Bool   *bool
}

// ParseFilter reads the buckets of a filter on a field of the given mapping
// type and rejects what no aggregation on that field could have answered.
func ParseFilter(f *searchsvc.AggregationFilter, fieldType string) (Buckets, error) {
	field := f.GetField()
	if fieldType == "" {
		return Buckets{}, fmt.Errorf("unknown aggregation filter field %q", field)
	}
	if len(f.GetTerms())+len(f.GetRanges()) == 0 {
		return Buckets{}, fmt.Errorf("aggregation filter on field %q selects no bucket", field)
	}
	ranges, err := ParseRanges(field, f.GetRanges())
	if err != nil {
		return Buckets{}, err
	}
	if err := validateRangeField(field, fieldType, ranges); err != nil {
		return Buckets{}, err
	}
	buckets := Buckets{Ranges: ranges}
	if len(f.GetTerms()) == 0 {
		return buckets, nil
	}
	if err := validateTermsField(field, fieldType); err != nil {
		return Buckets{}, err
	}
	for _, key := range f.GetTerms() {
		term, err := parseTerm(fieldType, key)
		if err != nil {
			return Buckets{}, fmt.Errorf("invalid bucket key on field %q: %w", field, err)
		}
		buckets.Terms = append(buckets.Terms, term)
	}
	return buckets, nil
}

func parseTerm(fieldType, key string) (Term, error) {
	term := Term{Key: key}
	switch fieldType {
	case mapping.TypeNumeric:
		// the key as a bucket spells it, no other spelling of the number
		v, err := parseNumber(key)
		if err != nil || v == nil || NumberKey(*v) != key {
			return Term{}, fmt.Errorf("%q is no bucket key of a number", key)
		}
		term.Number = v
	case mapping.TypeBool:
		v, err := strconv.ParseBool(key)
		if err != nil || key != BoolKey(v) {
			return Term{}, fmt.Errorf("%q is neither true nor false", key)
		}
		term.Bool = &v
	}
	return term, nil
}
