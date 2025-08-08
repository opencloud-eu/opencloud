package osu

import (
	"encoding/json"
	"fmt"
	"reflect"

	"dario.cat/mergo"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
)

type Builder interface {
	json.Marshaler
	Map() (map[string]any, error)
}

func newBase(v ...any) (map[string]any, error) {
	base := make(map[string]any)
	for _, value := range v {
		data, err := conversions.To[map[string]any](value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert value to map: %w", err)
		}

		if isEmpty(data) {
			continue
		}

		if err := mergo.Merge(&base, data); err != nil {
			return nil, fmt.Errorf("failed to merge value into base: %w", err)
		}
	}

	return base, nil
}

func applyValue[T any](target map[string]any, key string, v T) {
	if target == nil || isEmpty(key) || isEmpty(v) {
		return
	}

	target[key] = v
}

func applyValues[T any](target map[string]any, values map[string]T) {
	if target == nil || isEmpty(values) {
		return
	}

	for k, v := range values {
		applyValue[T](target, k, v)
	}
}

func applyBuilder(target map[string]any, key string, builder Builder) error {
	if target == nil || isEmpty(key) || isEmpty(builder) {
		return nil
	}

	data, err := builder.Map()
	if err != nil {
		return fmt.Errorf("failed to map builder %s: %w", key, err)
	}

	if !isEmpty(data) {
		target[key] = data
	}

	return nil
}

func applyBuilders(target map[string]any, key string, bs ...Builder) error {
	if target == nil || isEmpty(key) || isEmpty(bs) {
		return nil
	}

	builders := make([]map[string]any, 0, len(bs))
	for _, builder := range bs {
		data, err := builder.Map()
		switch {
		case err != nil:
			return fmt.Errorf("failed to map builder %s: %w", key, err)
		case isEmpty(data):
			continue
		default:
			builders = append(builders, data)
		}
	}

	if len(builders) > 0 {
		target[key] = builders
	}

	return nil
}

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
		data, err := conversions.To[map[string]any](option)
		if err != nil {
			continue
		}

		_ = mergo.Merge(&mapOptions, data)
	}

	data, _ := conversions.To[T](mapOptions)

	return data
}
