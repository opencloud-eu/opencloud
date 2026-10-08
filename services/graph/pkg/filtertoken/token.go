// Package filtertoken encodes and decodes the MS Graph aggregationFilterToken:
// quoted hex terms tokens behind a U+01C2 pair, and range(from, to) range tokens.
package filtertoken

import (
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"
)

// U+01C2 LATIN LETTER ALVEOLAR CLICK, twice
const termPrefix = "ǂǂ"

const (
	openLower = "min"
	openUpper = "max"
	// MS Graph appends this to an open upper bound; the bound itself is
	// still max, so the marker carries no extra information.
	openUpperMarker = `to="le"`
)

// Filter is a decoded aggregationFilters entry: the buckets of one field a
// match has to be in, any of them.
type Filter struct {
	Field  string
	Terms  []string
	Ranges []Range
}

// Range has an empty bound on an open side.
type Range struct {
	From, To string
}

// EncodeTermsToken hex-encodes the bucket key; the quotes are part of the token value.
func EncodeTermsToken(key string) string {
	return `"` + termPrefix + hex.EncodeToString([]byte(key)) + `"`
}

// EncodeRangeToken uses the MS Graph spelling: open bounds are min/max, an open
// upper bound carries to="le".
func EncodeRangeToken(from, to string) string {
	if from == "" {
		from = openLower
	}
	if to == "" {
		return "range(" + from + ", " + openUpper + ", " + openUpperMarker + ")"
	}
	return "range(" + from + ", " + to + ")"
}

// DecodeFilter reads {field}:{token}, the token being a terms token, a range
// token or an or(...) of them. Anything not shaped like a server-issued token
// is rejected; the bounds of a range are returned as written.
func DecodeFilter(filter string) (Filter, error) {
	field, token, ok := strings.Cut(filter, ":")
	if !ok || field == "" || token == "" {
		return Filter{}, fmt.Errorf("invalid aggregation filter %q", filter)
	}
	f := Filter{Field: field}
	tokens := []string{token}
	if inner, ok := trimCall(token, "or"); ok {
		tokens = splitArguments(inner)
	}
	for _, t := range tokens {
		if inner, ok := trimCall(t, "range"); ok {
			r, err := decodeRange(inner)
			if err != nil {
				return Filter{}, fmt.Errorf("invalid range token %q: %w", t, err)
			}
			f.Ranges = append(f.Ranges, r)
			continue
		}
		term, err := decodeTerm(t)
		if err != nil {
			return Filter{}, err
		}
		f.Terms = append(f.Terms, term)
	}
	// the server issues one kind of token per field
	if len(f.Terms) > 0 && len(f.Ranges) > 0 {
		return Filter{}, fmt.Errorf("invalid aggregation filter %q: terms and ranges in one or()", filter)
	}
	return f, nil
}

func decodeTerm(token string) (string, error) {
	inner, ok := strings.CutPrefix(token, `"`+termPrefix)
	if ok {
		inner, ok = strings.CutSuffix(inner, `"`)
	}
	// the encoder writes lowercase hex, and no bucket has an empty key
	if !ok || inner == "" || inner != strings.ToLower(inner) {
		return "", fmt.Errorf("invalid terms token %q", token)
	}
	b, err := hex.DecodeString(inner)
	if err != nil || !utf8.Valid(b) {
		return "", fmt.Errorf("invalid terms token %q", token)
	}
	return string(b), nil
}

func decodeRange(arguments string) (Range, error) {
	segments := splitArguments(arguments)
	if len(segments) == 3 && segments[1] == openUpper && segments[2] == openUpperMarker {
		segments = segments[:2]
	}
	if len(segments) != 2 || segments[0] == "" || segments[1] == "" {
		return Range{}, fmt.Errorf("expected a lower and an upper bound")
	}
	var r Range
	if segments[0] != openLower {
		r.From = segments[0]
	}
	if segments[1] != openUpper {
		r.To = segments[1]
	}
	if r.From == "" && r.To == "" {
		return Range{}, fmt.Errorf("no bounds")
	}
	return r, nil
}

func trimCall(token, name string) (string, bool) {
	if !strings.HasPrefix(token, name+"(") || !strings.HasSuffix(token, ")") {
		return "", false
	}
	return token[len(name)+1 : len(token)-1], true
}

// splitArguments splits on the commas outside of parentheses and trims the
// optional whitespace around each argument.
func splitArguments(s string) []string {
	var out []string
	depth, start := 0, 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	return append(out, strings.TrimSpace(s[start:]))
}
