package content

import (
	"mime"
	"slices"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
)

// thumbnailResourceTypes are how Tika marks the image it considers the file's
// thumbnail: one stored inside the file, an audio cover or a document's own
// thumbnail, or one Tika rendered for the purpose, a pdf's first page. The
// unpacker's thumbnail preset extracts that same image, so the index reports
// what the thumbnailer serves. A file that needs no such image, an image
// itself for one, has no marked part.
var thumbnailResourceTypes = []string{"THUMBNAIL", "RENDERING"}

// getPreview returns the size and type of the preview image Tika found inside
// the file, or nil when Tika marked none.
func getPreview(metas []map[string][]string) *Preview {
	for _, meta := range metas {
		resourceType, _ := getFirstValue(meta, "tk:embedded-resource-type")
		if !slices.Contains(thumbnailResourceTypes, resourceType) {
			continue
		}
		ct, err := getFirstValue(meta, "Content-Type")
		if err != nil || !strings.HasPrefix(ct, "image/") {
			continue
		}
		// Tika answers with a full header value, parameters included; what is
		// stored is the media type alone, the way a file's own type is stored
		mediaType, _, err := mime.ParseMediaType(ct)
		if err != nil {
			continue
		}
		w, _ := getFirstValue(meta, "tiff:ImageWidth")
		h, _ := getFirstValue(meta, "tiff:ImageLength")
		preview := &Preview{
			Width:    conversions.StringToInt32(w, 0),
			Height:   conversions.StringToInt32(h, 0),
			MimeType: mediaType,
		}
		if preview.Width <= 0 || preview.Height <= 0 {
			continue
		}

		return preview
	}

	return nil
}
