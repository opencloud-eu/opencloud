package mapping

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
)

// PrepareForIndex converts v to the flat map[string]any the backend index
// clients expect: a json round-trip (conversions.To) plus type-specific
// adaptations (currently geopoint siblings). Pass the same overrides as the
// *BuildMapping calls so the document and the mapping stay in sync.
func PrepareForIndex(v any, overrides map[string]FieldOpts) (map[string]any, error) {
	out, err := conversions.To[map[string]any](v)
	if err != nil {
		return nil, fmt.Errorf("mapping: prepare %T: %w", v, err)
	}
	if out == nil {
		return out, nil
	}
	addGeopointSiblings(out, overrides)
	return out, nil
}
