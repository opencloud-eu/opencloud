package email

import (
	"html"
	"html/template"
	"maps"
	"strings"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/l10n"
)

// htmlContent is the data of the html templates. Text fields are already translated and
// hold escaped variables, so they are safe to use as template.HTML.
type htmlContent struct {
	Subject  template.HTML
	Greeting template.HTML
	// Paragraphs are shown before the button, Notes after it
	Paragraphs []htmlParagraph
	Notes      []htmlParagraph
	// ActionLink is built from the unescaped variables, the template escapes it
	ActionLabel       string
	ActionLink        string
	FallbackLinkLabel string
	// Entries are the events of a digest email
	Entries []digestEntry

	// MessageBody and CallToAction are the placeholders of the legacy single html template
	MessageBody  template.HTML
	CallToAction template.HTML
}

// htmlParagraph is a paragraph of the message body, Kind is one of "text", "hint", "code" and "note"
type htmlParagraph struct {
	Kind  string
	Text  template.HTML
	Label template.HTML
	Value template.HTML
}

// digestEntry is one event in the timeline of a digest email
type digestEntry struct {
	Time       string
	Paragraphs []htmlParagraph
	Last       bool
}

// newMessageHTML translates the template and prepares the data of the html templates
func newMessageHTML(mt MessageTemplate, locale, defaultLocale, translationPath string, vars map[string]string) (htmlContent, error) {
	escaped := escapeStringMap(vars)
	legacy, err := NewHTMLTemplate(mt, locale, defaultLocale, translationPath, escaped)
	if err != nil {
		return htmlContent{}, err
	}
	t := l10n.NewTranslatorFromCommonConfig(defaultLocale, _domain, translationPath, _translationFS, "l10n/locale").Locale(locale)
	body, err := composeMessage(t.Get(mt.MessageBody, []any{}...), escaped)
	if err != nil {
		return htmlContent{}, err
	}
	content := htmlContent{
		Subject:      template.HTML(legacy.Subject),                         // #nosec G203
		Greeting:     template.HTML(strings.TrimSpace(legacy.Greeting)),     // #nosec G203
		MessageBody:  template.HTML(strings.TrimSpace(legacy.MessageBody)),  // #nosec G203
		CallToAction: template.HTML(strings.TrimSpace(legacy.CallToAction)), // #nosec G203
	}
	for _, p := range layoutParagraphs(body, mt.bodyLayout) {
		if p.Kind == "note" {
			content.Notes = append(content.Notes, p)
			continue
		}
		content.Paragraphs = append(content.Paragraphs, p)
	}
	if mt.actionLink != "" {
		content.ActionLink, err = composeMessage(mt.actionLink, vars)
		if err != nil {
			return htmlContent{}, err
		}
		content.ActionLabel = t.Get(mt.ActionLabel, []any{}...)
		content.FallbackLinkLabel = t.Get(_fallbackLinkLabel, []any{}...)
	}
	return content, nil
}

// newDigestHTML translates the grouped template and prepares the data of the digest html template
func newDigestHTML(gmt GroupedMessageTemplate, vars map[string]string, locale, defaultLocale, translationPath string, mts []MessageTemplate, mtsVars []map[string]string, times []time.Time) (htmlContent, error) {
	escapedMtsVars := make([]map[string]string, 0, len(mtsVars))
	for _, m := range mtsVars {
		escapedMtsVars = append(escapedMtsVars, escapeStringMap(m))
	}
	legacy, err := NewGroupedHTMLTemplate(gmt, escapeStringMap(vars), locale, defaultLocale, translationPath, mts, escapedMtsVars)
	if err != nil {
		return htmlContent{}, err
	}
	t := l10n.NewTranslatorFromCommonConfig(defaultLocale, _domain, translationPath, _translationFS, "l10n/locale").Locale(locale)
	content := htmlContent{
		Subject:     template.HTML(legacy.Subject),                        // #nosec G203
		Greeting:    template.HTML(strings.TrimSpace(legacy.Greeting)),    // #nosec G203
		MessageBody: template.HTML(strings.TrimSpace(legacy.MessageBody)), // #nosec G203
	}
	for i, mt := range mts {
		entryVars, err := linkResourceName(mt, mtsVars[i], escapedMtsVars[i])
		if err != nil {
			return htmlContent{}, err
		}
		body, err := composeMessage(t.Get(mt.MessageBody, []any{}...), entryVars)
		if err != nil {
			return htmlContent{}, err
		}
		entry := digestEntry{Paragraphs: layoutParagraphs(body, mt.bodyLayout)}
		if i < len(times) {
			entry.Time = formatEntryTime(times[i])
		}
		content.Entries = append(content.Entries, entry)
	}
	if n := len(content.Entries); n > 0 {
		content.Entries[n-1].Last = true
	}
	return content, nil
}

// linkResourceName returns the escaped variables of a digest entry with the resource name
// turned into a link to the resource, if the template has one
func linkResourceName(mt MessageTemplate, vars, escaped map[string]string) (map[string]string, error) {
	if mt.digestLinkVar == "" || mt.actionLink == "" {
		return escaped, nil
	}
	link, err := composeMessage(mt.actionLink, vars)
	if err != nil {
		return nil, err
	}
	// the link is set into the message as html, so only plain web links are accepted
	if !strings.HasPrefix(link, "https://") && !strings.HasPrefix(link, "http://") {
		return escaped, nil
	}
	linked := maps.Clone(escaped)
	linked[mt.digestLinkVar] = `<a class="oc-link" href="` + html.EscapeString(link) + `" style="color: #00677f; text-decoration: none;">` + escaped[mt.digestLinkVar] + `</a>`
	return linked, nil
}

// layoutParagraphs splits a composed message body into paragraphs and gives each the role of
// the layout. The roles only apply if the number of paragraphs matches the layout, so a
// translation that merges or splits paragraphs is shown as plain text.
func layoutParagraphs(body string, layout []paragraphRole) []htmlParagraph {
	var parts []string
	for _, part := range strings.Split(strings.TrimSpace(body), "\n\n") {
		if part = strings.TrimSpace(part); part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) != len(layout) {
		layout = nil
	}

	paragraphs := make([]htmlParagraph, 0, len(parts))
	for i, part := range parts {
		p := htmlParagraph{Kind: "text", Text: template.HTML(newlineToBr(part))} // #nosec G203
		if layout != nil {
			switch layout[i] {
			case paragraphHint:
				p.Kind = "hint"
			case paragraphNote:
				p.Kind = "note"
			case paragraphCode:
				if label, value, ok := cutLast(part, ":"); ok {
					p.Kind = "code"
					p.Label = template.HTML(strings.TrimSpace(label)) // #nosec G203
					p.Value = template.HTML(strings.TrimSpace(value)) // #nosec G203
				}
			}
		}
		paragraphs = append(paragraphs, p)
	}
	return paragraphs
}

// cutLast slices s around the last instance of sep
func cutLast(s, sep string) (before, after string, found bool) {
	if i := strings.LastIndex(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

// formatEntryTime formats the time of a digest entry, the notifications service does not know
// the time zone of the recipient
func formatEntryTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02 15:04") + " UTC"
}
