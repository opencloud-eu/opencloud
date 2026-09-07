package content

import (
	"strconv"
	"strings"

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
		w, wErr := getFirstValue(meta, "tiff:ImageWidth")
		h, hErr := getFirstValue(meta, "tiff:ImageLength")
		if wErr != nil || hErr != nil {
			continue
		}
		width, wErr := strconv.ParseInt(w, 10, 32)
		height, hErr := strconv.ParseInt(h, 10, 32)
		if wErr != nil || hErr != nil || width <= 0 || height <= 0 {
			continue
		}
		preview := &Preview{Width: int32(width), Height: int32(height)}
		if desc, _ := getFirstValue(meta, "dc:description"); desc == frontCoverDescription {
			return preview
		}
		if first == nil {
			first = preview
		}
	}
	return first
}
