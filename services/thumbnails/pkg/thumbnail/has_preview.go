package thumbnail

import (
	"strconv"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
)

// Arbitrary-metadata keys holding an embedded preview's dimensions (e.g. audio
// cover art), written at index time. Their presence signals a preview exists.
const (
	PreviewWidthKey  = "oc.preview.width"
	PreviewHeightKey = "oc.preview.height"
)

// HasPreview reports whether a preview can be produced for the resource:
// unconditional types by mimetype, embedded-preview types by stored dimensions.
func HasPreview(md *provider.ResourceInfo) bool {
	if md == nil {
		return false
	}
	w, h := PreviewDimensions(md)
	return HasPreviewForMimeType(md.GetMimeType(), w > 0 && h > 0)
}

// HasPreviewForMimeType is HasPreview for callers that only have the mimetype
// and a presence signal (e.g. search results) rather than a full ResourceInfo.
func HasPreviewForMimeType(mimeType string, hasEmbeddedPreview bool) bool {
	if _, ok := UnconditionalPreviewMimeTypes[mimeType]; ok {
		return true
	}
	if _, ok := EmbeddedPreviewMimeTypes[mimeType]; ok {
		return hasEmbeddedPreview
	}
	return false
}

// PreviewDimensions returns the stored embedded-preview dimensions, or (0, 0).
// Only meaningful for EmbeddedPreviewMimeTypes.
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
