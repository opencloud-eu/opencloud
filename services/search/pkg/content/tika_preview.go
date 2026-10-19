package content

import (
	"mime"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
)

// thumbnailResourceType is how Tika marks the embedded image it considers the
// file's thumbnail. The unpacker's thumbnail preset extracts that same image,
// so the index reports what the thumbnailer serves.
const thumbnailResourceType = "THUMBNAIL"

// getPreview returns the size and type of the preview image Tika found inside
// the file. It only runs for the types that carry their preview inside.
func getPreview(mimeType string, metas []map[string][]string) *Preview {
	if !thumbnail.IsEmbeddedSourceType(mimeType) {
		return nil
	}
	for _, meta := range metas {
		if resourceType, _ := getFirstValue(meta, "tk:embedded-resource-type"); resourceType != thumbnailResourceType {
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
