package thumbnail

import (
	"mime"
	"strings"

	thumbnails "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail"
)

// SourceTypes are the types a preview is announced and attempted for. The
// defaults are the ones the thumbnails service compiles in, which is also where
// reva reads them for oc:has-preview, so a build with libvips brings its extra
// types along. Configured entries are merged on top here, per service: nobody
// writes into that package's map.
//
// The list is a hint, not a verdict. The generator decides what it can really
// read, and a source it refuses answers 404.
type SourceTypes struct {
	byType map[string]struct{}
}

// NewSourceTypes merges the configured types onto the compiled-in defaults.
func NewSourceTypes(entries []string) SourceTypes {
	t := SourceTypes{byType: make(map[string]struct{}, len(thumbnails.SupportedMimeTypes)+len(entries))}
	for mimeType := range thumbnails.SupportedMimeTypes {
		t.byType[mimeType] = struct{}{}
	}
	for _, entry := range entries {
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

// All lists the types, so a caller can iterate them.
func (t SourceTypes) All() map[string]struct{} { return t.byType }

// List is the types as a slice, for a configuration that takes one.
func (t SourceTypes) List() []string {
	out := make([]string, 0, len(t.byType))
	for mimeType := range t.byType {
		out = append(out, mimeType)
	}

	return out
}
