package thumbnail

import (
	"mime"
	"strings"
)

// Tika is the configured Tika server: its URL and the types it is asked for.
// It only decides which file goes there, the call lives in the preprocessor.
// The default list is a single "*", so every non-image file is sent and Tika
// decides; a list that names types restricts it, an empty one sends nothing.
type Tika struct {
	URL string
	// our mime type -> Tika's, "" when equal; "*" means every type
	mimeTypes map[string]string
}

// NewTika parses "mime[:tika-mime]" entries, "*" being every type.
func NewTika(url string, mimeTypes []string) Tika {
	t := Tika{URL: url, mimeTypes: map[string]string{}}
	for _, entry := range mimeTypes {
		ours, theirs, _ := strings.Cut(strings.ToLower(strings.TrimSpace(entry)), ":")
		if ours = strings.TrimSpace(ours); ours != "" {
			t.mimeTypes[ours] = strings.TrimSpace(theirs)
		}
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

// ConfiguredFor reports whether OC_TIKA_THUMBNAIL_MIME_TYPES lists this type,
// as opposed to the catch-all taking it. An image is only taken away from the
// generator when the configuration asks for it, because the generator may well
// read the type itself.
func (t Tika) ConfiguredFor(mimeType string) bool {
	if t.URL == "" {
		return false
	}
	m, _, err := mime.ParseMediaType(mimeType)
	if err != nil {
		return false
	}
	_, named := t.mimeTypes[m]

	return named
}

// ContentType is the type Tika is told, mapped where configured. Tika takes a
// declared type only when it refines what it detected itself, so a mapping pins
// a subtype its detector left at the parent type, a raw photo it only read as a
// tiff. It cannot overrule a type Tika detected.
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
