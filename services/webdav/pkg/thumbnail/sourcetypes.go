package thumbnail

import (
	"mime"
	"strings"
)

// DefaultSourceTypes are the types a preview is announced and attempted for
// without further configuration. The image types are what both resize backends
// read; the others are converted to an image here, by a preprocessor, before a
// generator ever sees them.
var DefaultSourceTypes = []string{
	"image/png",
	"image/jpg",
	"image/jpeg",
	"image/gif",
	"image/bmp",
	"image/x-ms-bmp",
	"image/tiff",
	"image/webp",
	"text/plain",
	"audio/flac",
	"audio/mpeg",
	"audio/ogg",
	"application/vnd.geogebra.slides",
	"application/vnd.geogebra.pinboard",
}

// SourceTypes are the types a preview is announced and attempted for: the
// defaults plus what a deployment configured, which is also the list the
// frontend hands to ocdav for oc:has-preview and graph answers its thumbnails
// relationship from.
//
// The list is a hint, not a verdict. The generator decides what it can really
// read, and a source it refuses answers 404.
type SourceTypes struct {
	byType map[string]struct{}
}

// NewSourceTypes merges the configured types onto the defaults.
func NewSourceTypes(entries []string) SourceTypes {
	t := SourceTypes{byType: make(map[string]struct{}, len(DefaultSourceTypes)+len(entries))}
	for _, entry := range append(append([]string{}, DefaultSourceTypes...), entries...) {
		if entry = strings.ToLower(strings.TrimSpace(entry)); entry != "" {
			t.byType[entry] = struct{}{}
		}
	}

	return t
}

// Supports reports whether a preview is made of this type.
func (t SourceTypes) Supports(contentType string) bool {
	parsed, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	_, supported := t.byType[parsed]

	return supported
}

// List is the types as a slice, for a configuration that takes one.
func (t SourceTypes) List() []string {
	out := make([]string, 0, len(t.byType))
	for mimeType := range t.byType {
		out = append(out, mimeType)
	}

	return out
}
