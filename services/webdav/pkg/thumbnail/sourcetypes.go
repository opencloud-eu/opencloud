package thumbnail

import (
	"mime"
	"slices"
	"strings"
)

// GeneratorImageTypes are the image types both resize backends read, so webdav
// hands the file to the generator as it is. Every other type needs something to
// make an image of it first, a preprocessor here or a Tika server.
var GeneratorImageTypes = []string{
	"image/png",
	"image/jpg",
	"image/jpeg",
	"image/gif",
	"image/bmp",
	"image/x-ms-bmp",
	"image/tiff",
	"image/webp",
}

// DefaultSourceTypes are the types a preview is announced and attempted for
// without further configuration.
var DefaultSourceTypes = slices.Concat(GeneratorImageTypes, []string{
	"text/plain",
	"audio/flac",
	"audio/mpeg",
	"audio/ogg",
	"application/vnd.geogebra.slides",
	"application/vnd.geogebra.pinboard",
})

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
	for _, entry := range slices.Concat(DefaultSourceTypes, entries) {
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
