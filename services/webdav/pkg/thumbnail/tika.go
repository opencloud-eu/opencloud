package thumbnail

import (
	"mime"
	"strings"
)

// Tika is the server that thumbnails what the built-in converters cannot.
// By default every non-image file is sent and Tika decides.
type Tika struct {
	URL string
	// our mime type -> Tika's, "" when equal; "*" means every type
	mimeTypes map[string]string
}

// NewTika parses "mime[:tika-mime]" entries; empty means every type.
func NewTika(url string, mimeTypes []string) Tika {
	t := Tika{URL: url, mimeTypes: map[string]string{}}
	for _, entry := range mimeTypes {
		ours, theirs, _ := strings.Cut(strings.ToLower(strings.TrimSpace(entry)), ":")
		if ours = strings.TrimSpace(ours); ours != "" {
			t.mimeTypes[ours] = strings.TrimSpace(theirs)
		}
	}
	if len(t.mimeTypes) == 0 {
		t.mimeTypes["*"] = ""
	}
	return t
}

// Supports reports whether the type is sent to Tika for its thumbnail.
func (t Tika) Supports(mimeType string) bool {
	if t.URL == "" {
		return false
	}
	m, _, err := mime.ParseMediaType(mimeType)
	if err != nil || m == "httpd/unix-directory" {
		return false
	}
	if _, ok := t.mimeTypes[m]; ok {
		return true
	}
	_, all := t.mimeTypes["*"]
	return all
}

// ContentType is the type Tika is told, mapped where configured.
func (t Tika) ContentType(mimeType string) string {
	m, _, err := mime.ParseMediaType(mimeType)
	if err != nil {
		return ""
	}
	if theirs := t.mimeTypes[m]; theirs != "" {
		return theirs
	}
	return m
}
