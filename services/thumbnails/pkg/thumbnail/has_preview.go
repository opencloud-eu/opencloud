package thumbnail

import (
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
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
	w, h := previewDimensions(md)
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

// SourceDimensions returns the dimensions of the image the preview is rendered
// from, or (0, 0) when they are unknown.
func SourceDimensions(md *provider.ResourceInfo) (int32, int32) {
	// A preview extracted or rendered at index time (e.g. audio cover) has its own size.
	if w, h := previewDimensions(md); w > 0 && h > 0 {
		return w, h
	}
	// Images render themselves, the image facet holds their size.
	meta := md.GetArbitraryMetadata().GetMetadata()
	w := conversions.StringToInt32(meta["libre.graph.image.width"], 0)
	h := conversions.StringToInt32(meta["libre.graph.image.height"], 0)
	return w, h
}

func previewDimensions(md *provider.ResourceInfo) (int32, int32) {
	meta := md.GetArbitraryMetadata().GetMetadata()
	w := conversions.StringToInt32(meta[PreviewWidthKey], 0)
	h := conversions.StringToInt32(meta[PreviewHeightKey], 0)
	return w, h
}
