package query

import (
	"strconv"
	"strings"
	"time"

	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

// Nothing in a request says which kind an extension property has, so the
// literal decides: a range targets the sibling of its typed literal, an
// equality asks every sibling the literal fits. Both compilers follow this plan.

// OpenExtensionTerm is one sibling an equality is asked on.
type OpenExtensionTerm struct {
	Field   string
	Sibling string

	String string
	Number float64
	Bool   bool
	Time   time.Time
}

// IsOpenExtensionField reports whether a KQL key addresses an extension property.
func IsOpenExtensionField(key string) bool {
	return mapping.IsOpenExtensionQueryField(key)
}

// OpenExtensionField is the indexed field name of the sibling for a KQL key.
func OpenExtensionField(key, sibling string) string {
	field, _ := mapping.OpenExtensionFieldFromQuery(key, sibling)
	return field
}

// OpenExtensionEquality asks the string sibling, case-insensitively unless exact,
// and every other sibling the value reads as.
func OpenExtensionEquality(key, value string, exact bool) []OpenExtensionTerm {
	term := func(sibling string) OpenExtensionTerm {
		return OpenExtensionTerm{Field: OpenExtensionField(key, sibling), Sibling: sibling}
	}

	str := term(mapping.SiblingLower)
	str.String = strings.ToLower(value)
	if exact {
		str = term(mapping.SiblingKeyword)
		str.String = value
	}
	plan := []OpenExtensionTerm{str}

	if f, err := strconv.ParseFloat(value, 64); err == nil {
		num := term(mapping.SiblingNumber)
		num.Number = f
		plan = append(plan, num)
	}
	if b, err := strconv.ParseBool(value); err == nil && (value == "true" || value == "false") {
		bl := term(mapping.SiblingBool)
		bl.Bool = b
		plan = append(plan, bl)
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		dt := term(mapping.SiblingDate)
		dt.Time = t
		plan = append(plan, dt)
	}
	return plan
}
