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

// Regression: the ScienceMesh subject showed the literal {InitiatorName} placeholder.
func TestScienceMeshSubjectNamesInitiator(t *testing.T) {
	vars := map[string]string{
		"ShareSharer":     "Anna",
		"InitiatorName":   "Anna",
		"ShareSharerMail": "anna@example.org",
		"ShareLink":       "https://cloud.example.org/accept",
		"Token":           "token",
		"ProviderDomain":  "cloud.example.org",
	}
	for locale, want := range map[string]string{
		"en": "ScienceMesh: Anna wants to collaborate with you",
		"es": "ScienceMesh: Anna quiere colaborar contigo",
	} {
		msg, err := RenderEmailTemplate(ScienceMeshInviteTokenGenerated, locale, "en", "", "", vars)
		if err != nil {
			t.Fatal(err)
		}
		if msg.Subject != want {
			t.Errorf("%s: got subject %q, want %q", locale, msg.Subject, want)
		}
	}
}

// Regression: translating an empty string returns the header of the .po file, so emails
// without a call to action got "Project-Id-Version: …" in every translated language.
func TestEmptyStringsAreNotTranslated(t *testing.T) {
	vars := map[string]string{"ShareGrantee": "Ben", "ShareSharer": "Anna", "ShareFolder": "Q3 Report"}

	msg, err := RenderGroupedEmailTemplate(GroupedDaily, map[string]string{"DisplayName": "Ben"}, "de", "en", "", "",
		[]MessageTemplate{ShareRemoved}, []map[string]string{vars}, nil)
	if err != nil {
		t.Fatal(err)
	}
	single, err := RenderEmailTemplate(ShareRemoved, "de", "en", "", "", vars)
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"digest text": msg.TextBody,
		"digest html": msg.HTMLBody,
		"single text": single.TextBody,
		"single html": single.HTMLBody,
	} {
		if strings.Contains(body, "Project-Id-Version") {
			t.Errorf("%s contains the header of the translation file:\n%s", name, body)
		}
	}
}
