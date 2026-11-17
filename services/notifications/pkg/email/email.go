// Package email implements utility for rendering the Email.
//
// The email package supports transifex translation for email templates.
package email

import (
	"bytes"
	"embed"
	"errors"
	"html"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/opencloud-eu/opencloud/services/notifications/pkg/channels"
)

var (
	//go:embed templates
	templatesFS embed.FS

	imgDir = filepath.Join("templates", "html", "img")
)

// RenderEmailTemplate is responsible to prepare a message which than can be used to notify the user via email.
func RenderEmailTemplate(mt MessageTemplate, locale, defaultLocale string, emailTemplatePath string, translationPath string, vars map[string]string) (*channels.Message, error) {
	textMt, err := NewTextTemplate(mt, locale, defaultLocale, translationPath, vars)
	if err != nil {
		return nil, err
	}
	tpl, err := parseTemplate(emailTemplatePath, mt.textTemplate)
	if err != nil {
		return nil, err
	}
	textBody, err := emailTemplate(tpl, textMt)
	if err != nil {
		return nil, err
	}
	htmlContent, err := newMessageHTML(mt, locale, defaultLocale, translationPath, vars)
	if err != nil {
		return nil, err
	}
	htmlTpl, err := parseHTMLTemplate(emailTemplatePath, mt.htmlTemplate)
	if err != nil {
		return nil, err
	}
	htmlBody, err := executeTemplate(htmlTpl, htmlContent)
	if err != nil {
		return nil, err
	}
	data, err := readImages(emailTemplatePath)
	if err != nil {
		return nil, err
	}

	return &channels.Message{
		Subject:      textMt.Subject,
		TextBody:     textBody,
		HTMLBody:     htmlBody,
		AttachInline: data,
	}, nil
}

// RenderGroupedEmailTemplate is responsible to prepare a message which than can be used to notify the user via email.
// times holds the time of each grouped event, it is shown in the timeline of the html email.
func RenderGroupedEmailTemplate(gmt GroupedMessageTemplate, vars map[string]string, locale, defaultLocale string, emailTemplatePath string, translationPath string, mts []MessageTemplate, mtsVars []map[string]string, times []time.Time) (*channels.Message, error) {
	textMt, err := NewGroupedTextTemplate(gmt, vars, locale, defaultLocale, translationPath, mts, mtsVars)
	if err != nil {
		return nil, err
	}
	tpl, err := parseTemplate(emailTemplatePath, gmt.textTemplate)
	if err != nil {
		return nil, err
	}
	textBody, err := groupedEmailTemplate(tpl, textMt)
	if err != nil {
		return nil, err
	}

	htmlContent, err := newDigestHTML(gmt, vars, locale, defaultLocale, translationPath, mts, mtsVars, times)
	if err != nil {
		return nil, err
	}
	htmlTpl, err := parseHTMLTemplate(emailTemplatePath, gmt.htmlTemplate)
	if err != nil {
		return nil, err
	}
	htmlBody, err := executeTemplate(htmlTpl, htmlContent)
	if err != nil {
		return nil, err
	}
	data, err := readImages(emailTemplatePath)
	if err != nil {
		return nil, err
	}

	return &channels.Message{
		Subject:      textMt.Subject,
		TextBody:     textBody,
		HTMLBody:     htmlBody,
		AttachInline: data,
	}, nil
}

// emailTemplate builds the email template. It does not use any user provided input, so it is safe to use template.HTML.
func emailTemplate(tpl *template.Template, mt MessageTemplate) (string, error) {
	str, err := executeTemplate(tpl, map[string]any{
		"Greeting":     template.HTML(strings.TrimSpace(mt.Greeting)),     // #nosec G203
		"MessageBody":  template.HTML(strings.TrimSpace(mt.MessageBody)),  // #nosec G203
		"CallToAction": template.HTML(strings.TrimSpace(mt.CallToAction)), // #nosec G203
	})
	if err != nil {
		return "", err
	}
	return str, err
}

// groupedEmailTemplate builds the email template. It does not use any user provided input, so it is safe to use template.HTML.
func groupedEmailTemplate(tpl *template.Template, gmt GroupedMessageTemplate) (string, error) {
	str, err := executeTemplate(tpl, map[string]any{
		"Greeting":    template.HTML(strings.TrimSpace(gmt.Greeting)),    // #nosec G203
		"MessageBody": template.HTML(strings.TrimSpace(gmt.MessageBody)), // #nosec G203
	})
	if err != nil {
		return "", err
	}
	return str, err
}

func parseTemplate(emailTemplatePath string, file string) (*template.Template, error) {
	if emailTemplatePath != "" {
		return template.ParseFiles(filepath.Join(emailTemplatePath, file))
	}
	return template.ParseFS(templatesFS, filepath.Join(file))
}

// parseHTMLTemplate parses the html layout together with the given child template. A custom
// template folder without the layout is a folder made for the legacy single html template,
// which is used instead; it gets the same data with the Greeting, MessageBody and CallToAction
// placeholders.
func parseHTMLTemplate(emailTemplatePath string, child string) (*template.Template, error) {
	if emailTemplatePath == "" {
		return template.ParseFS(templatesFS, _htmlLayout, child)
	}
	layout := filepath.Join(emailTemplatePath, _htmlLayout)
	if _, err := os.Stat(layout); errors.Is(err, fs.ErrNotExist) {
		return template.ParseFiles(filepath.Join(emailTemplatePath, _htmlLegacyTemplate))
	}
	return template.ParseFiles(layout, filepath.Join(emailTemplatePath, child))
}

func executeTemplate(tpl *template.Template, vars any) (string, error) {
	var writer bytes.Buffer
	if err := tpl.Execute(&writer, vars); err != nil {
		return "", err
	}
	return writer.String(), nil
}

// readImages reads the images that are attached inline, from the custom template folder if one
// is set, else the embedded ones
func readImages(emailTemplatePath string) (map[string][]byte, error) {
	if emailTemplatePath == "" {
		entries, err := fs.ReadDir(templatesFS, imgDir)
		if err != nil {
			return nil, err
		}
		return read(entries, templatesFS)
	}
	dir := filepath.Join(emailTemplatePath, imgDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	return read(entries, os.DirFS(emailTemplatePath))
}

func read(entries []fs.DirEntry, fsys fs.FS) (map[string][]byte, error) {
	list := make(map[string][]byte)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		file, err := fs.ReadFile(fsys, filepath.Join(imgDir, e.Name()))
		if err != nil {
			// skip if the symbolic is a directory
			if errors.Is(err, syscall.EISDIR) {
				continue
			}
			return nil, err
		}
		if !validateMime(file) {
			continue
		}
		list[e.Name()] = file
	}
	return list, nil
}

// signature image formats signature https://go.dev/src/net/http/sniff.go #L:118
var signature = map[string]string{
	"\xff\xd8\xff":      "image/jpeg",
	"\x89PNG\r\n\x1a\n": "image/png",
	"GIF87a":            "image/gif",
	"GIF89a":            "image/gif",
}

// validateMime validate the mime type of image file from its first few bytes
func validateMime(incipit []byte) bool {
	for s := range signature {
		if strings.HasPrefix(string(incipit), s) {
			return true
		}
	}
	return false
}

func escapeStringMap(vars map[string]string) map[string]string {
	escaped := make(map[string]string, len(vars))
	for k, v := range vars {
		escaped[k] = html.EscapeString(v)
	}
	return escaped
}
