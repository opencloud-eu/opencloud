// Package mapping builds search index mappings for bleve and OpenSearch from
// a Go struct via reflection. Field names come from json tags; the caller
// provides overrides for fields that need a specific type or analyzer.
package mapping

// Field type constants used in FieldOpts.Type. An empty Type means the type
// is inferred from the Go field via reflection.
const (
	TypeKeyword  = "keyword"
	TypeFulltext = "fulltext"
	TypePath     = "path"
	TypeWildcard = "wildcard"
	TypeNumeric  = "numeric"
	TypeDatetime = "datetime"
	TypeBool     = "bool"
	TypeObject   = "object"
	TypeGeopoint = "geopoint"
)

// FieldOpts overrides the default type inference for a struct field. Keys in
// the override map are json-tag names (e.g. "Name", "location", "audio.artist"),
// not Go field names.
type FieldOpts struct {
	// Type is one of the Type* constants. Empty means "infer from Go type".
	Type string

	// Analyzer is the name of a custom analyzer registered on the bleve
	// IndexMapping (e.g. "lowercaseKeyword", "fulltext"). For OpenSearch it
	// becomes the analyzer attribute on the field.
	Analyzer string

	// IncludeInAll controls bleve's _all field inclusion. Nil means "use the
	// bleve default for this field type". Has no effect on OpenSearch.
	IncludeInAll *bool
}
