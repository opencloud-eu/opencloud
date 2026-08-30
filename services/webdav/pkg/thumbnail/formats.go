package thumbnail

import (
	"mime"
	"strings"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/generator"
)

// No generator writes an icon back, and the format they fall back to is JPEG,
// which has no alpha channel.
var DefaultFormats = []string{
	"image/vnd.microsoft.icon:image/png",
	"image/x-icon:image/png",
}

// Formats says which type a thumbnail of an image is written in, for the types
// where the image's own is not the answer.
type Formats struct {
	byType map[string]string
}

func NewFormats(entries []string) Formats {
	f := Formats{byType: map[string]string{}}
	for _, entry := range entries {
		source, target, _ := strings.Cut(strings.ToLower(strings.TrimSpace(entry)), ":")
		if source = strings.TrimSpace(source); source != "" {
			f.byType[source] = strings.TrimSpace(target)
		}
	}
	return f
}

// ExtFor is the generator's name for the format, "" when nothing is mapped.
func (f Formats) ExtFor(contentType string) string {
	m, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	return generator.ExtForMime(f.byType[m])
}
