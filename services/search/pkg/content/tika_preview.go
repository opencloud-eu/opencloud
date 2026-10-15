package content

import (
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail"
)

// frontCoverDescription is the picture type Tika reports (as dc:description) for
// the front cover, matching the thumbnailer's cover selection.
const frontCoverDescription = "Cover (front)"

// getPreview returns the dimensions of an audio file's embedded cover art from
// Tika's recursive metadata, preferring the front cover (dc:description) and
// falling back to the first image, matching the thumbnailer's selection. It only
// runs for EmbeddedPreviewMimeTypes.
func getPreview(mimeType string, metas []map[string][]string) *Preview {
	if _, ok := thumbnail.EmbeddedPreviewMimeTypes[mimeType]; !ok {
		return nil
	}
	var first *Preview
	for _, meta := range metas {
		ct, err := getFirstValue(meta, "Content-Type")
		if err != nil || !strings.HasPrefix(ct, "image/") {
			continue
		}
		w, _ := getFirstValue(meta, "tiff:ImageWidth")
		h, _ := getFirstValue(meta, "tiff:ImageLength")
		preview := &Preview{Width: conversions.StringToInt32(w, 0), Height: conversions.StringToInt32(h, 0)}
		if preview.Width <= 0 || preview.Height <= 0 {
			continue
		}
		if desc, _ := getFirstValue(meta, "dc:description"); desc == frontCoverDescription {
			return preview
		}
		if first == nil {
			first = preview
		}
	}
	return first
}
