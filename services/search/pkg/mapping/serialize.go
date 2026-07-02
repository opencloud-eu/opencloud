package mapping

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
)

// PrepareForIndex converts v to the flat map[string]any the backend index
// clients expect, via a json round-trip (conversions.To). overrides is
// reserved for type-specific adaptations wired in by follow-up features.
func PrepareForIndex(v any, overrides map[string]FieldOpts) (map[string]any, error) {
	out, err := conversions.To[map[string]any](v)
	if err != nil {
		return nil, fmt.Errorf("mapping: prepare %T: %w", v, err)
	}
	return out, nil
}
