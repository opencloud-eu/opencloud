package email

import (
	"strings"
	"testing"
)

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

// Regression for #3830: translating an empty string returns the header of the .po file, so emails
// without a call to action got "Project-Id-Version: …" in every translated language.
func TestEmptyStringsAreNotTranslated(t *testing.T) {
	vars := map[string]string{"ShareGrantee": "Ben", "ShareFolder": "Q3 Report", "ExpiredAt": "2026-10-27 10:00:00"}
	msg, err := RenderEmailTemplate(ShareExpired, "de", "en", "", "", vars)
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"text": msg.TextBody, "html": msg.HTMLBody} {
		if strings.Contains(body, "Project-Id-Version") {
			t.Errorf("%s body contains the header of the translation file:\n%s", name, body)
		}
	}
}
