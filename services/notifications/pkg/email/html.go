package email

import (
	"html/template"
	"maps"
	"strings"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/l10n"
)

// htmlContent is the data of the html templates. All text is plain text with unescaped
// variables, html/template escapes it.
type htmlContent struct {
	Subject  string
	Greeting string
	// Paragraphs are shown before the button, Notes after it
	Paragraphs        []htmlParagraph
	Notes             []htmlParagraph
	ActionLabel       string
	ActionLink        string
	FallbackLinkLabel string
	// Entries are the events of a digest email
	Entries []digestEntry

	// MessageBody and CallToAction are the placeholders of the legacy single html template
	MessageBody  template.HTML
	CallToAction template.HTML
}

// htmlParagraph is a paragraph of the message body, Kind is one of "text", "hint", "code" and "note".
// A code paragraph has a Label and a Value instead of Lines.
type htmlParagraph struct {
	Kind  string
	Lines []htmlLine
	Label string
	Value string
}

// htmlLine is a line of a paragraph, a segment with a Href is shown as a link
type htmlLine []htmlSegment

type htmlSegment struct {
	Text string
	Href string
}

// digestEntry is one event in the timeline of a digest email
type digestEntry struct {
	Time       string
	Paragraphs []htmlParagraph
	Last       bool
}

// _linkMarker stands in for the linked resource name while the message is composed
const _linkMarker = "\x00oc-link\x00"

// newMessageHTML translates the template and prepares the data of the html templates
func newMessageHTML(mt MessageTemplate, locale, defaultLocale, translationPath string, vars map[string]string) (htmlContent, error) {
	legacy, err := NewHTMLTemplate(mt, locale, defaultLocale, translationPath, escapeStringMap(vars))
	if err != nil {
		return htmlContent{}, err
	}
	t := l10n.NewTranslatorFromCommonConfig(defaultLocale, _domain, translationPath, _translationFS, "l10n/locale").Locale(locale)
	content := htmlContent{
		MessageBody:  legacyHTML(legacy.MessageBody),
		CallToAction: legacyHTML(legacy.CallToAction),
	}
	if content.Subject, err = composeMessage(translate(t, mt.Subject), vars); err != nil {
		return htmlContent{}, err
	}
	if content.Greeting, err = composeMessage(translate(t, mt.Greeting), vars); err != nil {
		return htmlContent{}, err
	}
	body, err := composeMessage(translate(t, mt.MessageBody), vars)
	if err != nil {
		return htmlContent{}, err
	}
	for _, p := range layoutParagraphs(body, mt.bodyLayout, nil) {
		if p.Kind == "note" {
			content.Notes = append(content.Notes, p)
			continue
		}
		content.Paragraphs = append(content.Paragraphs, p)
	}
	if mt.actionLink != "" {
		if content.ActionLink, err = composeMessage(mt.actionLink, vars); err != nil {
			return htmlContent{}, err
		}
		content.ActionLabel = translate(t, mt.ActionLabel)
		content.FallbackLinkLabel = translate(t, _fallbackLinkLabel)
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
	content := htmlContent{MessageBody: legacyHTML(legacy.MessageBody)}
	if content.Subject, err = composeMessage(translate(t, gmt.Subject), vars); err != nil {
		return htmlContent{}, err
	}
	if content.Greeting, err = composeMessage(translate(t, gmt.Greeting), vars); err != nil {
		return htmlContent{}, err
	}
	for i, mt := range mts {
		entryVars, link, err := markResourceName(mt, mtsVars[i])
		if err != nil {
			return htmlContent{}, err
		}
		body, err := composeMessage(translate(t, mt.MessageBody), entryVars)
		if err != nil {
			return htmlContent{}, err
		}
		entry := digestEntry{Paragraphs: layoutParagraphs(body, mt.bodyLayout, link)}
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

// markResourceName replaces the resource name of a digest entry with a marker, so that it can
// be shown as a link to the resource. It returns the variables to compose the message with and
// the segment the marker stands for, nil if the template has no link.
func markResourceName(mt MessageTemplate, vars map[string]string) (map[string]string, *htmlSegment, error) {
	if mt.digestLinkVar == "" || mt.actionLink == "" {
		return vars, nil, nil
	}
	link, err := composeMessage(mt.actionLink, vars)
	if err != nil {
		return nil, nil, err
	}
	marked := maps.Clone(vars)
	marked[mt.digestLinkVar] = _linkMarker
	return marked, &htmlSegment{Text: vars[mt.digestLinkVar], Href: link}, nil
}

// layoutParagraphs splits a composed message body into paragraphs and gives each the role of
// the layout. The roles only apply if the number of paragraphs matches the layout, so a
// translation that merges or splits paragraphs is shown as plain text. Markers in the body
// are replaced with the link segment.
func layoutParagraphs(body string, layout []paragraphRole, link *htmlSegment) []htmlParagraph {
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
		p := htmlParagraph{Kind: "text", Lines: splitLines(part, link)}
		if layout != nil {
			switch layout[i] {
			case paragraphHint:
				p.Kind = "hint"
			case paragraphNote:
				p.Kind = "note"
			case paragraphCode:
				if label, value, ok := cutLast(part, ":"); ok {
					p = htmlParagraph{Kind: "code", Label: strings.TrimSpace(label), Value: strings.TrimSpace(value)}
				}
			}
		}
		paragraphs = append(paragraphs, p)
	}
	return paragraphs
}

// splitLines splits a paragraph into lines of text segments, with the link in place of each marker
func splitLines(paragraph string, link *htmlSegment) []htmlLine {
	var lines []htmlLine
	for _, text := range strings.Split(paragraph, "\n") {
		var line htmlLine
		for i, part := range strings.Split(text, _linkMarker) {
			if i > 0 && link != nil {
				line = append(line, *link)
			}
			if part != "" {
				line = append(line, htmlSegment{Text: part})
			}
		}
		lines = append(lines, line)
	}
	return lines
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

// legacyHTML marks the html for the legacy single html template as safe. It is composed from
// translations and from variables that escapeStringMap escaped before.
func legacyHTML(s string) template.HTML {
	// nosemgrep
	return template.HTML(strings.TrimSpace(s)) // #nosec G203
}
