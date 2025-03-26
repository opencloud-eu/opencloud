package email

import (
	"bytes"
	"embed"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	"github.com/opencloud-eu/opencloud/pkg/l10n"
)

var (
	//go:embed l10n/locale
	_translationFS embed.FS
	_domain        = "notifications"
)

// NewTextTemplate replace the body message template placeholders with the translated template
func NewTextTemplate(mt MessageTemplate, locale, defaultLocale string, translationPath string, vars map[string]string) (MessageTemplate, error) {
	var err error
	t := l10n.NewTranslatorFromCommonConfig(defaultLocale, _domain, translationPath, _translationFS, "l10n/locale").Locale(locale)
	mt.Subject, err = composeMessage(t.Get(mt.Subject, []interface{}{}...), vars)
	if err != nil {
		return mt, err
	}
	mt.Greeting, err = composeMessage(t.Get(mt.Greeting, []interface{}{}...), vars)
	if err != nil {
		return mt, err
	}
	mt.MessageBody, err = composeMessage(t.Get(mt.MessageBody, []interface{}{}...), vars)
	if err != nil {
		return mt, err
	}
	mt.CallToAction, err = composeMessage(t.Get(mt.CallToAction, []interface{}{}...), vars)
	if err != nil {
		return mt, err
	}
	return mt, nil
}

// NewHTMLTemplate replace the body message template placeholders with the translated template
func NewHTMLTemplate(mt MessageTemplate, locale, defaultLocale string, translationPath string, vars map[string]string) (MessageTemplate, error) {
	var err error
	t := l10n.NewTranslatorFromCommonConfig(defaultLocale, _domain, translationPath, _translationFS, "l10n/locale").Locale(locale)
	mt.Subject, err = composeMessage(t.Get(mt.Subject, []interface{}{}...), vars)
	if err != nil {
		return mt, err
	}
	mt.Greeting, err = composeMessage(newlineToBr(t.Get(mt.Greeting, []interface{}{}...)), vars)
	if err != nil {
		return mt, err
	}
	mt.MessageBody, err = composeMessage(newlineToBr(t.Get(mt.MessageBody, []interface{}{}...)), vars)
	if err != nil {
		return mt, err
	}
	mt.CallToAction, err = composeMessage(callToActionToHTML(t.Get(mt.CallToAction, []interface{}{}...)), vars)
	if err != nil {
		return mt, err
	}
	return mt, nil
}

// NewGroupedTextTemplate replace the body message template placeholders with the translated template
func NewGroupedTextTemplate(gmt GroupedMessageTemplate, vars map[string]string, locale, defaultLocale string, translationPath string, mts []MessageTemplate, mtsVars []map[string]string) (GroupedMessageTemplate, error) {
	if len(mts) != len(mtsVars) {
		return gmt, errors.New("number of templates does not match number of variables")
	}

	var err error
	t := l10n.NewTranslatorFromCommonConfig(defaultLocale, _domain, translationPath, _translationFS, "l10n/locale").Locale(locale)
	gmt.Subject, err = composeMessage(t.Get(gmt.Subject, []interface{}{}...), vars)
	if err != nil {
		return gmt, err
	}
	gmt.Greeting, err = composeMessage(t.Get(gmt.Greeting, []interface{}{}...), vars)
	if err != nil {
		return gmt, err
	}

	bodyParts := make([]string, 0, len(mtsVars))
	for i, mt := range mts {
		bodyPart, err := composeMessage(t.Get(mt.MessageBody, []interface{}{}...), mtsVars[i])
		if err != nil {
			return gmt, err
		}
		bodyParts = append(bodyParts, bodyPart)
	}
	gmt.MessageBody = strings.Join(bodyParts, "\n\n\n")
	return gmt, nil
}

// NewGroupedHTMLTemplate replace the body message template placeholders with the translated template
func NewGroupedHTMLTemplate(gmt GroupedMessageTemplate, vars map[string]string, locale, defaultLocale string, translationPath string, mts []MessageTemplate, mtsVars []map[string]string) (GroupedMessageTemplate, error) {
	if len(mts) != len(mtsVars) {
		return gmt, errors.New("number of templates does not match number of variables")
	}

	var err error
	t := l10n.NewTranslatorFromCommonConfig(defaultLocale, _domain, translationPath, _translationFS, "l10n/locale").Locale(locale)
	gmt.Subject, err = composeMessage(t.Get(gmt.Subject, []interface{}{}...), vars)
	if err != nil {
		return gmt, err
	}
	gmt.Greeting, err = composeMessage(newlineToBr(t.Get(gmt.Greeting, []interface{}{}...)), vars)
	if err != nil {
		return gmt, err
	}

	bodyParts := make([]string, 0, len(mtsVars))
	for i, mt := range mts {
		bodyPart, err := composeMessage(t.Get(mt.MessageBody, []interface{}{}...), mtsVars[i])
		if err != nil {
			return gmt, err
		}
		bodyParts = append(bodyParts, bodyPart)
	}
	gmt.MessageBody = strings.Join(bodyParts, "<br><br><br>")

	return gmt, nil
}

// composeMessage renders the message based on template
func composeMessage(tmpl string, vars map[string]string) (string, error) {
	tpl, err := template.New("").Parse(replacePlaceholders(tmpl))
	if err != nil {
		return "", err
	}
	var writer bytes.Buffer
	if err := tpl.Execute(&writer, vars); err != nil {
		return "", err
	}
	return writer.String(), nil
}

func replacePlaceholders(raw string) string {
	for o, n := range _placeholders {
		raw = strings.ReplaceAll(raw, o, n)
	}
	return raw
}

func newlineToBr(s string) string {
	return strings.Replace(s, "\n", "<br>", -1)
}

func callToActionToHTML(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	s = strings.TrimSuffix(s, "{ShareLink}")
	return s + `<a href="{ShareLink}">{ShareLink}</a>`
}
