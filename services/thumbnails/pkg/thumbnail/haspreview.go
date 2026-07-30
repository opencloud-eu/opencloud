package thumbnail

import (
	"strconv"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
)

// Arbitrary-metadata keys under which the content extraction pipeline stores the
// dimensions of an embedded preview (for example audio cover art). These are an
// internal signal, not a Microsoft Graph facet. Their presence means the file
// carries an embedded preview.
const (
	PreviewWidthKey  = "oc.preview.width"
	PreviewHeightKey = "oc.preview.height"
)

// HasPreview reports whether a thumbnail/preview can be produced for the given
// resource. For unconditional types it follows from the mimetype alone. For
// embedded-preview types (audio cover art) it depends on whether an embedded
// preview was detected at index time, signalled by the stored preview
// dimensions. Files that have not been indexed yet report no preview rather than
// promising one that would fail to render.
func HasPreview(md *provider.ResourceInfo) bool {
	if md == nil {
		return false
	}

	mimeType := md.GetMimeType()
	if _, ok := UnconditionalPreviewMimeTypes[mimeType]; ok {
		return true
	}
	if _, ok := EmbeddedPreviewMimeTypes[mimeType]; ok {
		w, h := PreviewDimensions(md)
		return w > 0 && h > 0
	}
	return false
}

// PreviewDimensions returns the stored dimensions of a resource's embedded
// preview, or (0, 0) if none were recorded. Only meaningful for
// EmbeddedPreviewMimeTypes.
func PreviewDimensions(md *provider.ResourceInfo) (width, height int32) {
	meta := md.GetArbitraryMetadata().GetMetadata()
	if meta == nil {
		return 0, 0
	}
	return parseInt32(meta[PreviewWidthKey]), parseInt32(meta[PreviewHeightKey])
}

func parseInt32(s string) int32 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0
	}
	return int32(v)
}
