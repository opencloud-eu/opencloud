package thumbnail

import (
	"mime"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
)

// Arbitrary-metadata keys holding an embedded preview's dimensions (e.g. audio
// cover art), written at index time. Their presence signals a preview exists.
const (
	PreviewWidthKey  = "oc.preview.width"
	PreviewHeightKey = "oc.preview.height"
)

const (
	imageWidthKey  = "libre.graph.image.width"
	imageHeightKey = "libre.graph.image.height"
)

// HasPreview reports whether a preview can be produced for the resource:
// unconditional types by mimetype, embedded-preview types by stored dimensions.
func HasPreview(res *provider.ResourceInfo) bool {
	if res == nil {
		return false
	}
	w, h := previewDimensions(res)
	return HasPreviewForMimeType(res.GetMimeType(), w > 0 && h > 0)
}

// HasPreviewForMimeType is HasPreview for callers that only have the mimetype
// and a presence signal (e.g. search results) rather than a full ResourceInfo.
func HasPreviewForMimeType(mimeType string, hasEmbeddedPreview bool) bool {
	if _, ok := UnconditionalPreviewMimeTypes[mediaType(mimeType)]; ok {
		return true
	}
	if IsEmbeddedPreviewMimeType(mimeType) {
		return hasEmbeddedPreview
	}
	return false
}

// IsEmbeddedPreviewMimeType reports whether the preview of this type depends on
// an embedded preview, see EmbeddedPreviewMimeTypes.
func IsEmbeddedPreviewMimeType(mimeType string) bool {
	_, ok := EmbeddedPreviewMimeTypes[mediaType(mimeType)]
	return ok
}

func mediaType(mimeType string) string {
	mediaType, _, err := mime.ParseMediaType(mimeType)
	if err != nil {
		return ""
	}
	return mediaType
}

// SourceDimensions returns the dimensions of the image the preview is rendered
// from, or (0, 0) when they are unknown.
func SourceDimensions(res *provider.ResourceInfo) (int32, int32) {
	// A preview extracted or rendered at index time (e.g. audio cover) has its own size.
	if w, h := previewDimensions(res); w > 0 && h > 0 {
		return w, h
	}
	// Images render themselves, the image facet holds their size.
	return dimensions(res, imageWidthKey, imageHeightKey)
}

func previewDimensions(res *provider.ResourceInfo) (int32, int32) {
	return dimensions(res, PreviewWidthKey, PreviewHeightKey)
}

func dimensions(res *provider.ResourceInfo, widthKey, heightKey string) (int32, int32) {
	meta := res.GetArbitraryMetadata().GetMetadata()
	return conversions.StringToInt32(meta[widthKey], 0), conversions.StringToInt32(meta[heightKey], 0)
}
