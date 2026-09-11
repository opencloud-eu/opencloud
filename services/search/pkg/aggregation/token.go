// Package aggregation encodes and decodes the aggregationFilterToken exchanged
// with clients. A terms token is the bucket key as lowercase hex of its UTF-8
// bytes, prefixed with U+01C2 twice and wrapped in double quotes (the same
// encoding MS Graph uses); a range token is range(from,to) with min/max for open
// bounds. Decoding turns a {field}:{token} filter into a KQL fragment the search
// engines parse and then force to an exact, case-sensitive match.
package aggregation

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// termPrefix marks a hex-encoded terms token (U+01C2 LATIN LETTER ALVEOLAR
// CLICK, twice).
const termPrefix = "ǂǂ"

// EncodeTermsToken encodes a terms bucket key as an aggregationFilterToken: the
// key as lowercase hex of its UTF-8 bytes, prefixed with termPrefix and wrapped
// in double quotes. The quotes are part of the token value.
func EncodeTermsToken(key string) string {
	return `"` + termPrefix + hex.EncodeToString([]byte(key)) + `"`
}

// EncodeRangeToken encodes a range bucket as range(from,to). An empty bound is
// open and written as min (lower) or max (upper).
func EncodeRangeToken(from, to string) string {
	if from == "" {
		from = "min"
	}
	if to == "" {
		to = "max"
	}
	return "range(" + from + "," + to + ")"
}

// DecodeAggregationFilter turns a {field}:{token} aggregation filter into a KQL
// fragment. Terms and or() tokens become field:"value" restrictions (to be
// matched exactly and case-sensitively by the caller); range() becomes a
// numeric/date range. Tokens that are not server-shaped are rejected.
func DecodeAggregationFilter(filter string) (string, error) {
	field, token, ok := strings.Cut(filter, ":")
	if !ok || field == "" || token == "" {
		return "", fmt.Errorf("invalid aggregation filter %q", filter)
	}
	switch {
	case strings.HasPrefix(token, "or("):
		return decodeOr(field, token)
	case strings.HasPrefix(token, "range("):
		return decodeRange(field, token)
	default:
		v, err := decodeTerm(token)
		if err != nil {
			return "", err
		}
		frag, err := kqlTerm(field, v)
		if err != nil {
			return "", err
		}
		return frag, nil
	}
}

// decodeTerm strips the quotes and termPrefix and hex-decodes a terms token.
func decodeTerm(token string) (string, error) {
	if len(token) < 2 || token[0] != '"' || token[len(token)-1] != '"' {
		return "", fmt.Errorf("invalid terms token %q", token)
	}
	inner := token[1 : len(token)-1]
	if !strings.HasPrefix(inner, termPrefix) {
		return "", fmt.Errorf("invalid terms token %q", token)
	}
	b, err := hex.DecodeString(strings.TrimPrefix(inner, termPrefix))
	if err != nil {
		return "", fmt.Errorf("invalid terms token %q: %w", token, err)
	}
	return string(b), nil
}

// decodeRange turns range(from,to) into a KQL comparison; open bounds (min/max)
// are dropped.
func decodeRange(field, token string) (string, error) {
	inner, ok := trimCall(token, "range")
	if !ok {
		return "", fmt.Errorf("invalid range token %q", token)
	}
	from, to, ok := strings.Cut(inner, ",")
	if !ok {
		return "", fmt.Errorf("invalid range token %q", token)
	}
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	var parts []string
	if from != "" && from != "min" {
		parts = append(parts, field+">="+from)
	}
	if to != "" && to != "max" {
		parts = append(parts, field+"<="+to)
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("range token %q has no bounds", token)
	}
	return "(" + strings.Join(parts, " AND ") + ")", nil
}

// decodeOr turns or("token","token",...) into an OR group of terms.
func decodeOr(field, token string) (string, error) {
	inner, ok := trimCall(token, "or")
	if !ok {
		return "", fmt.Errorf("invalid or token %q", token)
	}
	// terms tokens are quote-wrapped lowercase hex, so they never contain a
	// comma; a plain split is safe.
	parts := make([]string, 0)
	for _, t := range strings.Split(inner, ",") {
		v, err := decodeTerm(strings.TrimSpace(t))
		if err != nil {
			return "", err
		}
		frag, err := kqlTerm(field, v)
		if err != nil {
			return "", err
		}
		parts = append(parts, frag)
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("empty or token %q", token)
	}
	return "(" + strings.Join(parts, " OR ") + ")", nil
}

// trimCall unwraps name(inner); ok is false when token is not name(...).
func trimCall(token, name string) (string, bool) {
	if !strings.HasPrefix(token, name+"(") || !strings.HasSuffix(token, ")") {
		return "", false
	}
	return token[len(name)+1 : len(token)-1], true
}

// kqlTerm builds a field:"value" restriction. KQL quoted strings have no escape
// syntax, so a value containing a double quote cannot be expressed and is
// rejected rather than emitted as broken KQL.
func kqlTerm(field, value string) (string, error) {
	if strings.Contains(value, `"`) {
		return "", fmt.Errorf("aggregation value %q contains an unsupported double quote", value)
	}
	return field + `:"` + value + `"`, nil
}
