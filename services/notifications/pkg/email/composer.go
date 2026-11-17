package email

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/leonelquinteros/gotext"
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
	mt.Subject, err = composeMessage(translate(t, mt.Subject), vars)
	if err != nil {
		return mt, err
	}
	mt.Greeting, err = composeMessage(translate(t, mt.Greeting), vars)
	if err != nil {
		return mt, err
	}
	mt.MessageBody, err = composeMessage(translate(t, mt.MessageBody), vars)
	if err != nil {
		return mt, err
	}
	mt.CallToAction, err = composeMessage(translate(t, mt.CallToAction), vars)
	if err != nil {
		return mt, err
	}
	return mt, nil
}

// NewHTMLTemplate replace the body message template placeholders with the translated template
func NewHTMLTemplate(mt MessageTemplate, locale, defaultLocale string, translationPath string, vars map[string]string) (MessageTemplate, error) {
	var err error
	t := l10n.NewTranslatorFromCommonConfig(defaultLocale, _domain, translationPath, _translationFS, "l10n/locale").Locale(locale)
	mt.Subject, err = composeMessage(translate(t, mt.Subject), vars)
	if err != nil {
		return mt, err
	}
	mt.Greeting, err = composeMessage(newlineToBr(translate(t, mt.Greeting)), vars)
	if err != nil {
		return mt, err
	}
	mt.MessageBody, err = composeMessage(newlineToBr(translate(t, mt.MessageBody)), vars)
	if err != nil {
		return mt, err
	}
	mt.CallToAction, err = composeMessage(callToActionToHTML(translate(t, mt.CallToAction)), vars)
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
	gmt.Subject, err = composeMessage(translate(t, gmt.Subject), vars)
	if err != nil {
		return gmt, err
	}
	gmt.Greeting, err = composeMessage(translate(t, gmt.Greeting), vars)
	if err != nil {
		return gmt, err
	}

	bodyParts := make([]string, 0, len(mtsVars))
	for i, mt := range mts {
		bodyPart, err := composeMessage(translate(t, mt.MessageBody), mtsVars[i])
		if err != nil {
			return gmt, err
		}
		callToAction, err := composeMessage(translate(t, mt.CallToAction), mtsVars[i])
		if err != nil {
			return gmt, err
		}
		if callToAction = strings.TrimSpace(callToAction); callToAction != "" {
			bodyPart += "\n\n" + callToAction
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
	gmt.Subject, err = composeMessage(translate(t, gmt.Subject), vars)
	if err != nil {
		return gmt, err
	}
	gmt.Greeting, err = composeMessage(newlineToBr(translate(t, gmt.Greeting)), vars)
	if err != nil {
		return gmt, err
	}

	bodyParts := make([]string, 0, len(mtsVars))
	for i, mt := range mts {
		bodyPart, err := composeMessage(newlineToBr(translate(t, mt.MessageBody)), mtsVars[i])
		if err != nil {
			return gmt, err
		}
		callToAction, err := composeMessage(callToActionToHTML(translate(t, mt.CallToAction)), mtsVars[i])
		if err != nil {
			return gmt, err
		}
		if callToAction = strings.TrimSpace(callToAction); callToAction != "" {
			bodyPart += "<br><br>" + callToAction
		}
		bodyParts = append(bodyParts, bodyPart)
	}
	gmt.MessageBody = strings.Join(bodyParts, "<br><br><br>")

	return gmt, nil
}

// translate returns the translation of s. An empty string stays empty: gotext returns the
// header of the .po file for it.
func translate(t *gotext.Locale, s string) string {
	if s == "" {
		return ""
	}
	return t.Get(s)
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

	// substitute links
	for _, token := range []string{"ShareLink", "ResourceLink"} {
		s = strings.ReplaceAll(s, "{"+token+"}", fmt.Sprintf(`<a href="{%s}">{%s}</a>`, token, token))
	}

	return s
}
