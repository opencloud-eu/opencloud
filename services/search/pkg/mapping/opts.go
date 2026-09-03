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
	TypeVector   = "vector"
)

// LowercaseSuffix names the lowercased sibling of a keyword/path field.
const LowercaseSuffix = "_lowercase"

// WordsSuffix names the word-broken sibling of a keyword field.
const WordsSuffix = "_words"

// WordsAnalyzer names the analyzer both engines register for the words sibling.
const WordsAnalyzer = "words"

// FieldOpts overrides the default type inference for a struct field. Keys in
// the override map are json-tag names (e.g. "Name", "location", "audio.artist"),
// not Go field names.
type FieldOpts struct {
	// Type is one of the Type* constants. Empty means "infer from Go type".
	Type string

	// CaseInsensitive additionally indexes a lowercased <name>_lowercase sibling
	// for case-insensitive search; the case-preserved base is always indexed.
	// On by default for keyword/path fields, KQL searches case-insensitively;
	// false opts a field out (ids, paths).
	CaseInsensitive *bool

	// NoWordBreaker is SharePoint's switch, with its default: a keyword field
	// additionally indexes a <name>_words sibling split into lowercased words
	// (no stemming), so a single word matches a value that contains it,
	// "report" finds "Report.txt"; true opts a field out and leaves it one
	// whole value (tags, ids, paths). The base stays the whole value for
	// returning and aggregating; wildcards and whole-value matches use the
	// _lowercase sibling. Keyword only.
	NoWordBreaker *bool

	// IncludeInAll controls bleve's _all field inclusion. Nil means "use the
	// bleve default for this field type". Has no effect on OpenSearch.
	IncludeInAll *bool

	// Dims is the dimensionality of a TypeVector field. It is index schema:
	// vectors of any other length are not indexed (bleve drops them silently),
	// and changing it requires a new index. Similarity is always cosine.
	Dims int
}

func (o FieldOpts) caseInsensitive() bool { return o.CaseInsensitive == nil || *o.CaseInsensitive }
func (o FieldOpts) wordBroken() bool      { return o.NoWordBreaker == nil || !*o.NoWordBreaker }
