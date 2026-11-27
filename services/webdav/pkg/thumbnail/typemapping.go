package thumbnail

import (
	"mime"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/generator"
)

// DefaultTypeMapping maps the type of the image a thumbnail is made from, not
// the file's. Icons become PNG: no generator writes icons, and JPEG would
// flatten their transparency.
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
