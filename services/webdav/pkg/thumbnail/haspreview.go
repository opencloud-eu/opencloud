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

// EmbeddedSourceTypes carry their preview inside the file, audio cover art for
// one, so whether a preview exists is a property of the file and not of its
// type. The index answers that, through the dimensions it stored.
var EmbeddedSourceTypes = []string{
	"audio/flac",
	"audio/mpeg",
	"audio/ogg",
}

// HasPreview reports whether a preview can be produced for the resource: a type
// that always has one says so by itself, a type that carries its preview inside
// the file only when the index saw one.
func (t SourceTypes) HasPreview(md *provider.ResourceInfo) bool {
	if md == nil {
		return false
	}
	w, h := PreviewDimensions(md)

	return t.HasPreviewForMimeType(md.GetMimeType(), w > 0 && h > 0)
}

// HasPreviewForMimeType is HasPreview for callers that only have the mimetype
// and a presence signal (e.g. search results) rather than a full ResourceInfo.
func (t SourceTypes) HasPreviewForMimeType(mimeType string, hasEmbeddedPreview bool) bool {
	if !t.Supports(mimeType) {
		return false
	}
	if t.carriesPreviewInside(mimeType) {
		return hasEmbeddedPreview
	}

	return true
}

// PreviewDimensions returns the stored embedded-preview dimensions, or (0, 0).
func PreviewDimensions(md *provider.ResourceInfo) (int32, int32) {
	meta := md.GetArbitraryMetadata().GetMetadata()

	return conversions.StringToInt32(meta[PreviewWidthKey], 0), conversions.StringToInt32(meta[PreviewHeightKey], 0)
}
