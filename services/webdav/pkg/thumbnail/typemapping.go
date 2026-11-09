package thumbnail

import (
	"mime"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/generator"
)

// DefaultTypeMapping maps the type of the image a thumbnail is made from, the
// file itself or what an extractor found in it. Types that carry alpha or
// animation keep their own format, everything else becomes a JPEG. An icon is
// the one type no generator writes back: it has to become a PNG, or JPEG would
// flatten its transparent pixels onto black.
var DefaultTypeMapping = []string{
	"image/png:image/png",
	"image/gif:image/gif",
	"image/jpeg:image/jpeg",
	"image/jpg:image/jpeg",
	"image/webp:image/jpeg",
	"image/bmp:image/jpeg",
	"image/x-ms-bmp:image/jpeg",
	"image/tiff:image/jpeg",
	"image/vnd.microsoft.icon:image/png",
	"image/x-icon:image/png",
}

// TypeMapping says which type a thumbnail of an image is returned in, for the
// types where the image's own is not the answer.
type TypeMapping struct {
	byType map[string]string
}

// NewTypeMapping parses "mime:mime" entries on top of DefaultTypeMapping, so a
// configuration that names one type does not take the answer away from all the
// others.
func NewTypeMapping(entries []string) TypeMapping {
	return TypeMapping{byType: generator.ParsePairs(DefaultTypeMapping, entries)}
}

// TargetFor is the type a thumbnail of this type is returned in, "" when
// nothing is mapped and the caller keeps what it had.
func (m TypeMapping) TargetFor(contentType string) string {
	parsed, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}

	return m.byType[parsed]
}

// Targets lists the type a thumbnail is returned in, by source type.
func (m TypeMapping) Targets() map[string]string { return m.byType }
