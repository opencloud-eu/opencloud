package email

import "testing"

// Regression for #2804: escapeStringMap used to mutate its input map, and the
// recipient render loop reuses that map across iterations.
func TestEscapeStringMapDoesNotMutateInput(t *testing.T) {
	const raw = "Test & Demo"
	input := map[string]string{"SpaceName": raw}

	first := escapeStringMap(input)
	second := escapeStringMap(input)

	if got, want := first["SpaceName"], "Test &amp; Demo"; got != want {
		t.Errorf("first call: got %q, want %q", got, want)
	}
	if first["SpaceName"] != second["SpaceName"] {
		t.Errorf("escapeStringMap not idempotent on shared input: first=%q second=%q",
			first["SpaceName"], second["SpaceName"])
	}
	if input["SpaceName"] != raw {
		t.Errorf("escapeStringMap mutated its input: got %q, want %q",
			input["SpaceName"], raw)
	}
}
