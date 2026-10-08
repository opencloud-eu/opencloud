package aggregation

import (
	"fmt"
	"math"
	"strconv"
	"time"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
)

// Ranges are the parsed ranges of one aggregation or filter, all numeric or
// all dates. A range is from-inclusive and to-exclusive.
type Ranges struct {
	Numeric []NumericRange
	Dates   []DateRange
}

// NumericRange has a nil bound on an open side.
type NumericRange struct {
	Key      string
	From, To *float64
}

// DateRange has a zero time on an open side.
type DateRange struct {
	Key      string
	From, To time.Time
}

// Contains is from-inclusive and to-exclusive.
func (r NumericRange) Contains(v float64) bool {
	return (r.From == nil || v >= *r.From) && (r.To == nil || v < *r.To)
}

// Contains is from-inclusive and to-exclusive.
func (r DateRange) Contains(t time.Time) bool {
	return (r.From.IsZero() || !t.Before(r.From)) && (r.To.IsZero() || t.Before(r.To))
}

// RangeKey is the bucket key of a range, "from..to" with an open side left
// empty; the dots keep a negative bound readable.
func RangeKey(r *searchsvc.BucketRange) string {
	return r.GetFrom() + ".." + r.GetTo()
}

// ParseRanges accepts decimal numbers or RFC3339 dates as bounds, one kind for
// all ranges, and at least one bound per range.
func ParseRanges(field string, ranges []*searchsvc.BucketRange) (Ranges, error) {
	var out Ranges
	seen := make(map[string]struct{}, len(ranges))
	for _, r := range ranges {
		key := RangeKey(r)
		if r.GetFrom() == "" && r.GetTo() == "" {
			return Ranges{}, fmt.Errorf("range on field %q needs a from or a to bound", field)
		}
		if _, ok := seen[key]; ok {
			return Ranges{}, fmt.Errorf("duplicate range %q on field %q", key, field)
		}
		seen[key] = struct{}{}

		from, fromErr := parseNumber(r.GetFrom())
		to, toErr := parseNumber(r.GetTo())
		if fromErr == nil && toErr == nil {
			out.Numeric = append(out.Numeric, NumericRange{Key: key, From: from, To: to})
			continue
		}
		start, startErr := parseDate(r.GetFrom())
		end, endErr := parseDate(r.GetTo())
		if startErr == nil && endErr == nil {
			out.Dates = append(out.Dates, DateRange{Key: key, From: start, To: end})
			continue
		}
		return Ranges{}, fmt.Errorf("invalid range %q on field %q: bounds are numbers or RFC3339 dates", key, field)
	}
	if len(out.Numeric) > 0 && len(out.Dates) > 0 {
		return Ranges{}, fmt.Errorf("ranges on field %q mix numbers and dates", field)
	}
	return out, nil
}

func parseNumber(s string) (*float64, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return nil, fmt.Errorf("%q is not a finite number", s)
	}
	return &v, nil
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, s)
}
