package search

import (
	"reflect"
	"sync"

	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

var fieldTypes = sync.OnceValue(func() map[string]string {
	return mapping.FieldTypes(reflect.TypeFor[Resource](), Resource{}.SearchFieldOverrides())
})

// FieldType returns the mapping type of an index field (one of the
// mapping.Type* constants), empty for a field the index does not know.
func FieldType(field string) string {
	return fieldTypes()[field]
}

// AggregatableFieldType is FieldType for the fields a client may name, empty
// for the internal ones.
func AggregatableFieldType(field string) string {
	if resourceFieldOverrides()[field].Internal {
		return ""
	}
	return FieldType(field)
}
