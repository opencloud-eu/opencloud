package thumbnail

import (
	"mime"
	"slices"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
)

// Arbitrary-metadata keys holding an embedded preview's dimensions (e.g. audio
// cover art), written at index time. Their presence signals a preview exists.
const (
	PreviewWidthKey  = "oc.preview.width"
	PreviewHeightKey = "oc.preview.height"
	// the type of the embedded image itself, which decides what a thumbnail of
	// it is written in
	PreviewContentTypeKey = "oc.preview.contentType"
)

// the image facet an image writes about itself at index time
const (
	imageWidthKey  = "libre.graph.image.width"
	imageHeightKey = "libre.graph.image.height"
)

// EmbeddedSourceTypes carry their preview inside the file, audio cover art for
// one, so whether a preview exists is a property of the file and not of its
// type. The index answers that, through the dimensions it stored.
var EmbeddedSourceTypes = []string{
	"audio/flac",
	"audio/mpeg",
	"audio/ogg",
}

// IsEmbeddedSourceType reports whether the type carries its preview inside the
// file, which is what the index has to answer for.
func IsEmbeddedSourceType(contentType string) bool {
	parsed, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	return slices.Contains(EmbeddedSourceTypes, parsed)
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
	if IsEmbeddedSourceType(mimeType) {
		return hasEmbeddedPreview
	}

	return true
}

// SourceDimensions are the dimensions of the image the preview is rendered from,
// or (0, 0) when they are unknown.
func SourceDimensions(md *provider.ResourceInfo) (int32, int32) {
	// a preview extracted at index time, an audio cover for one, has its own size
	if w, h := PreviewDimensions(md); w > 0 && h > 0 {
		return w, h
	}

	// an image renders itself, so the image facet holds the size
	meta := md.GetArbitraryMetadata().GetMetadata()

	return conversions.StringToInt32(meta[imageWidthKey], 0), conversions.StringToInt32(meta[imageHeightKey], 0)
}

// PreviewContentType is the type of the embedded image the index saw, empty
// when nothing is stored.
func PreviewContentType(md *provider.ResourceInfo) string {
	return md.GetArbitraryMetadata().GetMetadata()[PreviewContentTypeKey]
}

// PreviewDimensions returns the stored embedded-preview dimensions, or (0, 0).
func PreviewDimensions(md *provider.ResourceInfo) (int32, int32) {
	meta := md.GetArbitraryMetadata().GetMetadata()

	return conversions.StringToInt32(meta[PreviewWidthKey], 0), conversions.StringToInt32(meta[PreviewHeightKey], 0)
}
