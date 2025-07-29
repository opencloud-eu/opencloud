package opensearch

import (
	"encoding/json"
	"reflect"

	"dario.cat/mergo"
)

func isEmpty(x any) bool {
	switch {
	case x == nil:
		return true
	case reflect.ValueOf(x).Kind() == reflect.Bool:
		return false
	case reflect.DeepEqual(x, reflect.Zero(reflect.TypeOf(x)).Interface()):
		return true
	case reflect.ValueOf(x).Kind() == reflect.Map && reflect.ValueOf(x).Len() == 0:
		return true
	default:
		return false
	}
}

func merge[T any](options ...T) T {
	mapOptions := make(map[string]any)

	for _, option := range options {
		data, err := convert[map[string]any](option)
		if err != nil {
			continue
		}

		_ = mergo.Merge(&mapOptions, data)
	}

	data, _ := convert[T](mapOptions)

	return data
}

func convert[T any](v any) (T, error) {
	var t T

	if v == nil {
		return t, nil
	}

	j, err := json.Marshal(v)
	if err != nil {
		return t, err
	}

	if err := json.Unmarshal(j, &t); err != nil {
		return t, err
	}

	return t, nil
}
