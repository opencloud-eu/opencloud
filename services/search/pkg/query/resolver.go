package query

import (
	"reflect"
	"strings"
	"sync"

	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// aliases are KQL spellings the derived index can't produce (fields are plural).
var aliases = map[string]string{
	"tag":      "Tags",
	"favorite": "Favorites",
}

// fieldIndex maps a lowercased KQL key to its canonical field name ("" is the
// bare-search default).
var fieldIndex = sync.OnceValue(func() map[string]string {
	idx := mapping.FieldNameIndex(reflect.TypeFor[search.Resource](), search.Resource{}.SearchFieldOverrides())
	for k, v := range aliases {
		idx[k] = v
	}
	idx[""] = idx["name"]
	return idx
})

// caseInsensitiveFields are the fields searched case-insensitively by default,
// derived from the CaseInsensitive overrides.
var caseInsensitiveFields = sync.OnceValue(func() map[string]struct{} {
	out := map[string]struct{}{}
	for field, opts := range (search.Resource{}).SearchFieldOverrides() {
		if opts.CaseInsensitive != nil && *opts.CaseInsensitive {
			out[field] = struct{}{}
		}
	}
	return out
})

// pathFields are hierarchical path fields (TypePath), derived from the overrides.
var pathFields = sync.OnceValue(func() map[string]struct{} {
	out := map[string]struct{}{}
	for field, opts := range (search.Resource{}).SearchFieldOverrides() {
		if opts.Type == mapping.TypePath {
			out[field] = struct{}{}
		}
	}
	return out
})

// fulltextFields are analyzed full-text fields (TypeFulltext), derived from the
// overrides.
var fulltextFields = sync.OnceValue(func() map[string]struct{} {
	out := map[string]struct{}{}
	for field, opts := range (search.Resource{}).SearchFieldOverrides() {
		if opts.Type == mapping.TypeFulltext {
			out[field] = struct{}{}
		}
	}
	return out
})

// ResolveField maps a KQL key to its canonical field name; unknown keys pass through.
func ResolveField(name string) string {
	if v, ok := fieldIndex()[strings.ToLower(name)]; ok {
		return v
	}
	return name
}

// FieldIsCaseInsensitive reports whether a field's default search is case-insensitive.
func FieldIsCaseInsensitive(field string) bool {
	_, ok := caseInsensitiveFields()[field]
	return ok
}

// FieldIsPath reports whether a field is a hierarchical path field.
func FieldIsPath(field string) bool {
	_, ok := pathFields()[field]
	return ok
}

// FieldIsFulltext reports whether a field is an analyzed full-text field.
func FieldIsFulltext(field string) bool {
	_, ok := fulltextFields()[field]
	return ok
}

// geopointFields maps a lowercased KQL key to its indexed geopoint sibling field,
// derived from the TypeGeopoint entries in the resource field overrides (e.g.
// "location" -> "location_geopoint", "journey.start" -> "journey.start_geopoint").
var geopointFields = sync.OnceValue(func() map[string]string {
	out := map[string]string{}
	for key, opts := range (search.Resource{}).SearchFieldOverrides() {
		if opts.Type == mapping.TypeGeopoint {
			out[strings.ToLower(key)] = key + mapping.GeopointSuffix
		}
	}
	return out
})

// ResolveGeoField maps a KQL key to its indexed geopoint field name. ok is false
// when the key is not a geopoint field, so callers can reject geo predicates on
// non-geo fields.
func ResolveGeoField(name string) (string, bool) {
	f, ok := geopointFields()[strings.ToLower(name)]
	return f, ok
}
