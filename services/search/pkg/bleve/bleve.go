package bleve

import (
	"regexp"

	bleveSearch "github.com/blevesearch/bleve/v2/search"
	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var queryEscape = regexp.MustCompile(`([` + regexp.QuoteMeta(`+=&|><!(){}[]^\"~*?:\/`) + `\-\s])`)

func getFieldValue[T any](m map[string]any, key string) (out T) {
	val, ok := m[key]
	if !ok {
		return
	}

	out, _ = val.(T)

	return
}

func resourceIDtoSearchID(id storageProvider.ResourceId) *searchMessage.ResourceID {
	return &searchMessage.ResourceID{
		StorageId: id.GetStorageId(),
		SpaceId:   id.GetSpaceId(),
		OpaqueId:  id.GetOpaqueId()}
}

func getFieldSliceValue[T any](m map[string]any, key string) (out []T) {
	iv := getFieldValue[any](m, key)
	add := func(v any) {
		cv, ok := v.(T)
		if !ok {
			return
		}

		out = append(out, cv)
	}

	// bleve tend to convert []string{"foo"} to type string if slice contains only one value
	// bleve: []string{"foo"} -> "foo"
	// bleve: []string{"foo", "bar"} -> []string{"foo", "bar"}
	switch v := iv.(type) {
	case T:
		add(v)
	case []any:
		for _, rv := range v {
			add(rv)
		}
	}

	return
}

func getFragmentValue(m bleveSearch.FieldFragmentMap, key string, idx int) string {
	val, ok := m[key]
	if !ok {
		return ""
	}

	if len(val) <= idx {
		return ""
	}

	return val[idx]
}

// hitToFacet builds a search Entity facet *T from a bleve hit's fields under the
// given key prefix. Nil when the hit has no such fields.
func hitToFacet[T any](fields map[string]any, prefix string) *T {
	return mapping.DeserializeAt[T](fields, prefix)
}

// matchToResource reconstructs a search.Resource from a bleve hit. Used by
// the Move / Delete / Restore / Purge paths that round-trip a record through
// the index. Always returns a non-nil *Resource: Deserialize is fail-soft
// for per-field parse errors, so corrupted hit values surface as zero
// values on individual fields instead of dropping the whole record.
func matchToResource(match *bleveSearch.DocumentMatch) *search.Resource {
	return mapping.Deserialize[search.Resource](match.Fields)
}

func escapeQuery(s string) string {
	return queryEscape.ReplaceAllString(s, "\\$1")
}
