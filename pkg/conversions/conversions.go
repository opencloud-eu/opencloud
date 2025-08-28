package conversions

import (
	"encoding/json"
)

func To[T any](v any) (T, error) {
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
