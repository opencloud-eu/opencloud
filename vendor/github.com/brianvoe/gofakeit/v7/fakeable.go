package gofakeit

import (
	"errors"
	"fmt"
	"reflect"
)

// Fakeable is an interface that can be implemented by a type to provide a custom fake value.
type Fakeable interface {
	// Fake returns a fake value for the type.
	Fake(faker *Faker) (any, error)
}

func isFakeable(t reflect.Type) bool {
	fakeableTyp := reflect.TypeOf((*Fakeable)(nil)).Elem()

	return t.Implements(fakeableTyp) || reflect.PointerTo(t).Implements(fakeableTyp)
}

func callFake(faker *Faker, v reflect.Value, possibleKinds ...reflect.Kind) (any, error) {
	f, ok := v.Addr().Interface().(Fakeable)
	if !ok {
		return nil, errors.New("not a Fakeable type")
	}

	fakedValue, err := f.Fake(faker)
	if err != nil {
		return nil, fmt.Errorf("error calling Fake: %w", err)
	}
	k := reflect.TypeOf(fakedValue).Kind()
	if !containsKind(possibleKinds, k) {
		return nil, fmt.Errorf("returned value kind %q is not amongst the valid ones: %v", k, possibleKinds)
	}

	switch k {
	case reflect.String:
		return reflect.ValueOf(fakedValue).String(), nil
	case reflect.Bool:
		return reflect.ValueOf(fakedValue).Bool(), nil
	case reflect.Int:
		return int(reflect.ValueOf(fakedValue).Int()), nil
	case reflect.Int8:
		return int8(reflect.ValueOf(fakedValue).Int()), nil
	case reflect.Int16:
		return int16(reflect.ValueOf(fakedValue).Int()), nil
	case reflect.Int32:
		return int32(reflect.ValueOf(fakedValue).Int()), nil
	case reflect.Int64:
		return int64(reflect.ValueOf(fakedValue).Int()), nil
	case reflect.Uint:
		return uint(reflect.ValueOf(fakedValue).Uint()), nil
	case reflect.Uint8:
		return uint8(reflect.ValueOf(fakedValue).Uint()), nil
	case reflect.Uint16:
		return uint16(reflect.ValueOf(fakedValue).Uint()), nil
	case reflect.Uint32:
		return uint32(reflect.ValueOf(fakedValue).Uint()), nil
	case reflect.Uint64:
		return uint64(reflect.ValueOf(fakedValue).Uint()), nil
	case reflect.Float32:
		return float32(reflect.ValueOf(fakedValue).Float()), nil
	case reflect.Float64:
		return float64(reflect.ValueOf(fakedValue).Float()), nil
	case reflect.Slice, reflect.Array:
		return reflect.ValueOf(fakedValue).Interface(), nil
	case reflect.Map:
		return reflect.ValueOf(fakedValue).Interface(), nil
	case reflect.Struct:
		return reflect.ValueOf(fakedValue).Interface(), nil

	default:
		return nil, fmt.Errorf("unsupported type %q", k)
	}
}

func containsKind(possibleKinds []reflect.Kind, kind reflect.Kind) bool {
	for _, k := range possibleKinds {
		if k == kind {
			return true
		}
	}
	return false
}
