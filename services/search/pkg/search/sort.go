package search

import (
	"reflect"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	searchmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
)

// Sorting support for search results (graph sortProperties / proto order_by).
//
// A field is sortable when both of the following hold:
//   - it is indexed as a scalar (string, number, bool or time), so the
//     engines can sort on it natively; multivalued fields like Tags are not
//     sortable
//   - it is carried on the match entity, so the service layer can read the
//     sort key when merging the per-space result streams
//
// Both sets are derived by reflection (index side: the Resource type, match
// side: the Entity proto), so new facet fields become sortable automatically.

// sortIndexAliases maps the graph-facing names of top-level fields to their
// index field names. Facet fields (photo.*, audio.*, ...) share the same
// dotted names in both worlds and need no alias. Top-level fields are only
// exposed under these graph names; internal fields like RootID or Deleted
// stay unsortable.
var sortIndexAliases = map[string]string{
	"name":                 "Name",
	"size":                 "Size",
	"lastModifiedDateTime": "Mtime",
	"mimeType":             "MimeType",
}

// entityJSONAliases maps graph-facing names to the Entity proto's JSON names
// where the two disagree.
var entityJSONAliases = map[string]string{
	"lastModifiedDateTime": "lastModifiedTime",
}

// IsSortableField reports whether results can be sorted by the field.
func IsSortableField(name string) bool {
	_, ok := SortIndexField(name)
	return ok
}

// SortIndexField translates a graph sortProperties name into the index field
// name to sort on, reporting whether the field is sortable at all.
func SortIndexField(name string) (string, bool) {
	field := name
	if alias, ok := sortIndexAliases[name]; ok {
		field = alias
	} else if !strings.Contains(name, ".") {
		return "", false
	}
	if !sortableIndexFields[field] {
		return "", false
	}
	if !entityFieldResolvable(name) {
		return "", false
	}
	return field, true
}

// CompareMatches orders match a relative to b according to orderBy: -1 when a
// comes first, 1 when b comes first, 0 when the sort keys tie (callers fall
// back to the score). Matches missing a sort key sort after those that have
// it, regardless of direction.
func CompareMatches(a, b *searchmsg.Match, orderBy []*searchsvc.SortProperty) int {
	for _, sp := range orderBy {
		ka := matchSortKey(a, sp.GetName())
		kb := matchSortKey(b, sp.GetName())
		if !ka.present && !kb.present {
			continue
		}
		if !ka.present {
			return 1
		}
		if !kb.present {
			return -1
		}
		c := 0
		switch {
		case ka.isString:
			c = strings.Compare(ka.str, kb.str)
		case ka.num < kb.num:
			c = -1
		case ka.num > kb.num:
			c = 1
		}
		if c == 0 {
			continue
		}
		if sp.GetIsDescending() {
			c = -c
		}
		return c
	}
	return 0
}

// sortableIndexFields is the set of scalar indexed fields, keyed by index
// field name.
var sortableIndexFields = buildSortableFieldSet()

func buildSortableFieldSet() map[string]bool {
	out := map[string]bool{}
	collectScalarFields(out, "", reflect.TypeOf(Resource{}))
	return out
}

func collectScalarFields(out map[string]bool, prefix string, t reflect.Type) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous {
			collectScalarFields(out, prefix, f.Type)
			continue
		}
		path := prefix + jsonFieldName(f)
		ft := f.Type
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		switch ft.Kind() {
		case reflect.String, reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			out[path] = true
		case reflect.Struct:
			if ft == timeType {
				out[path] = true
				continue
			}
			collectScalarFields(out, path+".", ft)
		}
	}
}

func entityPath(name string) []string {
	if alias, ok := entityJSONAliases[name]; ok {
		name = alias
	}
	return strings.Split(name, ".")
}

const timestampFullName = protoreflect.FullName("google.protobuf.Timestamp")

// entityFieldResolvable reports whether the graph field name resolves to a
// scalar (or timestamp) field on the match entity.
func entityFieldResolvable(name string) bool {
	md := (&searchmsg.Entity{}).ProtoReflect().Descriptor()
	segments := entityPath(name)
	for i, seg := range segments {
		fd := md.Fields().ByJSONName(seg)
		if fd == nil || fd.IsList() || fd.IsMap() {
			return false
		}
		if i < len(segments)-1 {
			if fd.Kind() != protoreflect.MessageKind {
				return false
			}
			md = fd.Message()
			continue
		}
		switch fd.Kind() {
		case protoreflect.StringKind, protoreflect.BoolKind,
			protoreflect.Int32Kind, protoreflect.Int64Kind,
			protoreflect.Sint32Kind, protoreflect.Sint64Kind,
			protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind,
			protoreflect.Uint32Kind, protoreflect.Uint64Kind,
			protoreflect.Fixed32Kind, protoreflect.Fixed64Kind,
			protoreflect.FloatKind, protoreflect.DoubleKind:
			return true
		case protoreflect.MessageKind:
			return fd.Message().FullName() == timestampFullName
		}
		return false
	}
	return false
}

// sortKey is the comparable value of a sort field on a concrete match.
type sortKey struct {
	present  bool
	isString bool
	str      string
	num      float64
}

// matchSortKey extracts the sort key for the graph field name from a match by
// walking the entity proto along the field's JSON names.
func matchSortKey(m *searchmsg.Match, name string) sortKey {
	entity := m.GetEntity()
	if entity == nil {
		return sortKey{}
	}
	msg := entity.ProtoReflect()
	segments := entityPath(name)
	for i, seg := range segments {
		fd := msg.Descriptor().Fields().ByJSONName(seg)
		if fd == nil || fd.IsList() || fd.IsMap() {
			return sortKey{}
		}
		if i < len(segments)-1 {
			if fd.Kind() != protoreflect.MessageKind || !msg.Has(fd) {
				return sortKey{}
			}
			msg = msg.Get(fd).Message()
			continue
		}
		if fd.HasPresence() && !msg.Has(fd) {
			return sortKey{}
		}
		v := msg.Get(fd)
		switch fd.Kind() {
		case protoreflect.StringKind:
			return sortKey{present: true, isString: true, str: v.String()}
		case protoreflect.BoolKind:
			num := 0.0
			if v.Bool() {
				num = 1.0
			}
			return sortKey{present: true, num: num}
		case protoreflect.Int32Kind, protoreflect.Int64Kind,
			protoreflect.Sint32Kind, protoreflect.Sint64Kind,
			protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind:
			return sortKey{present: true, num: float64(v.Int())}
		case protoreflect.Uint32Kind, protoreflect.Uint64Kind,
			protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
			return sortKey{present: true, num: float64(v.Uint())}
		case protoreflect.FloatKind, protoreflect.DoubleKind:
			return sortKey{present: true, num: v.Float()}
		case protoreflect.MessageKind:
			if fd.Message().FullName() != timestampFullName {
				return sortKey{}
			}
			ts := v.Message()
			seconds := ts.Get(ts.Descriptor().Fields().ByName("seconds")).Int()
			nanos := ts.Get(ts.Descriptor().Fields().ByName("nanos")).Int()
			return sortKey{present: true, num: float64(seconds) + float64(nanos)/1e9}
		}
		return sortKey{}
	}
	return sortKey{}
}
