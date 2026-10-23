package email

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLayoutParagraphs(t *testing.T) {
	body := "A new link has been requested.\n\nYour PIN: 482193\n\nDo not forward it.\n\nOnly valid for 30 minutes."
	got := layoutParagraphs(body, []paragraphRole{paragraphText, paragraphCode, paragraphText, paragraphNote})

	wantKinds := []string{"text", "code", "text", "note"}
	if len(got) != len(wantKinds) {
		t.Fatalf("got %d paragraphs, want %d", len(got), len(wantKinds))
	}
	for i, k := range wantKinds {
		if got[i].Kind != k {
			t.Errorf("paragraph %d: got kind %q, want %q", i, got[i].Kind, k)
		}
	}
	if got[1].Label != "Your PIN" || got[1].Value != "482193" {
		t.Errorf("code paragraph: got label %q value %q", got[1].Label, got[1].Value)
	}
}

func TestLayoutParagraphsIgnoresLayoutOfOtherLength(t *testing.T) {
	// a translation that merged two paragraphs must not shift the roles
	got := layoutParagraphs("Merged first and second.\n\nThird.", []paragraphRole{paragraphText, paragraphCode, paragraphNote})
	for i, p := range got {
		if p.Kind != "text" {
			t.Errorf("paragraph %d: got kind %q, want text", i, p.Kind)
		}
	}
}

func TestRenderEmailTemplateHTML(t *testing.T) {
	msg, err := RenderEmailTemplate(GuestLinkShareRenewed, "en", "en", "", "", map[string]string{
		"ShareSharer": "Anna <b>Lindqvist</b>",
		"ShareFolder": "Q3 Report",
		"SharePin":    "482193",
		"ShareLink":   "https://cloud.example.com/s/Xk29fQ?a=1&b=2",
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		`<h1 class="oc-text"`,
		"Your new link to 'Q3 Report'",
		"Anna &lt;b&gt;Lindqvist&lt;/b&gt;",
		`>Your PIN</div>`,
		`>482193</div>`,
		`href="https://cloud.example.com/s/Xk29fQ?a=1&amp;b=2"`,
		">Open link</a>",
		"Or open this link:",
		"The link and the PIN are only valid for 30 minutes.",
		`src="cid:opencloud-logo.png"`,
		`src="cid:opencloud-logo-dark.png"`,
		"prefers-color-scheme: dark",
	} {
		if !strings.Contains(msg.HTMLBody, want) {
			t.Errorf("html body is missing %q", want)
		}
	}
	if strings.Contains(msg.HTMLBody, "<b>Lindqvist</b>") {
		t.Error("html body contains unescaped user input")
	}
	for _, name := range []string{"opencloud-logo.png", "opencloud-logo-dark.png"} {
		if len(msg.AttachInline[name]) == 0 {
			t.Errorf("inline attachment %q is missing", name)
		}
	}
	// the plain text email keeps the full message body and call to action
	if !strings.Contains(msg.TextBody, "Your PIN: 482193") || !strings.Contains(msg.TextBody, "Click here to view it: https://cloud.example.com/s/Xk29fQ?a=1&b=2") {
		t.Errorf("unexpected text body:\n%s", msg.TextBody)
	}
}

func TestRenderEmailTemplateLegacyCustomFolder(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		_textTemplate:       "{{ .Greeting }} {{ .MessageBody }}",
		_htmlLegacyTemplate: "<p>{{ .Greeting }}</p><p>{{ .MessageBody }}</p><p>{{ .CallToAction }}</p>",
	} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, imgDir), 0o700); err != nil {
		t.Fatal(err)
	}

	msg, err := RenderEmailTemplate(ShareCreated, "en", "en", dir, "", map[string]string{
		"ShareGrantee": "Ben",
		"ShareSharer":  "Anna",
		"ShareFolder":  "Q3 Report",
		"ShareLink":    "https://cloud.example.com/files/shares/with-me",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `<p>Hello Ben</p><p>Anna has shared "Q3 Report" with you.</p><p>Click here to view it: <a href="https://cloud.example.com/files/shares/with-me">https://cloud.example.com/files/shares/with-me</a></p>`
	if msg.HTMLBody != want {
		t.Errorf("got html body\n%s\nwant\n%s", msg.HTMLBody, want)
	}
}

func TestRenderGroupedEmailTemplateTimeline(t *testing.T) {
	cet := time.FixedZone("CET", 3600)
	msg, err := RenderGroupedEmailTemplate(GroupedWeekly, map[string]string{"DisplayName": "Ben"}, "en", "en", "", "",
		[]MessageTemplate{ShareCreated, UnsharedSpace},
		[]map[string]string{
			{"ShareSharer": "Anna", "ShareFolder": "Q3 Report", "ShareLink": "https://cloud.example.com/files/shares/with-me"},
			{"SpaceSharer": "Jonas", "SpaceName": "Nostromo", "ShareLink": "https://cloud.example.com/f/spaceid"},
		},
		[]time.Time{time.Date(2026, 10, 21, 10, 12, 0, 0, cet), {}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Subject != "Your weekly notifications" {
		t.Errorf("got subject %q", msg.Subject)
	}
	for _, want := range []string{
		"Hi Ben,",
		">2026-10-21 09:12 UTC</td>",
		`<p class="oc-muted" style="margin: 8px 0 0; color: #40484c;">You might still have access through your other groups or direct membership.</p>`,
	} {
		if !strings.Contains(msg.HTMLBody, want) {
			t.Errorf("html body is missing %q", want)
		}
	}
	for _, want := range []string{
		`Anna has shared "<a class="oc-link" href="https://cloud.example.com/files/shares/with-me" style="color: #00677f; text-decoration: none;">Q3 Report</a>" with you.`,
		`Jonas has removed you from "<a class="oc-link" href="https://cloud.example.com/f/spaceid" style="color: #00677f; text-decoration: none;">Nostromo</a>".`,
	} {
		if !strings.Contains(msg.HTMLBody, want) {
			t.Errorf("html body is missing the linked resource name %q", want)
		}
	}
	for _, want := range []string{
		"Click here to view it: https://cloud.example.com/files/shares/with-me",
		"Click here to check it: https://cloud.example.com/f/spaceid",
	} {
		if !strings.Contains(msg.TextBody, want) {
			t.Errorf("text body is missing %q", want)
		}
	}
	// only the line between the two entries is drawn
	if n := strings.Count(msg.HTMLBody, `class="oc-line"`); n != 1 {
		t.Errorf("got %d timeline line segments, want 1", n)
	}
}
