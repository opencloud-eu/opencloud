package mapping

import (
	"reflect"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
)

// FromProto builds a *T (a libregraph facet) from the equally shaped proto
// facet message. The bridge runs over proto3 JSON, which renders int64 as
// strings, so leaves that fail the typed set are re-parsed from the string.
// Fail-soft like the deserializers: nil for nil input or an empty message.
func FromProto[T any, M any](m *M) *T {
	if m == nil {
		return nil
	}
	t := reflect.TypeFor[T]()
	fields, err := conversions.To[map[string]any](m)
	if err != nil || len(fields) == 0 {
		return nil
	}
	out := reflect.New(t)
	if !fillStruct(out.Elem(), fields, "", setValueLenient) {
		return nil
	}
	return out.Interface().(*T)
}

func setValueLenient(v reflect.Value, raw any) error {
	err := setValue(v, raw)
	if err != nil {
		if s, ok := raw.(string); ok {
			return setValueFromString(v, s)
		}
	}
	return err
}
