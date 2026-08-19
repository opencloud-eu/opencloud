//go:build enable_vips

package thumbnail

import (
	"github.com/davidbyttow/govips/v2/vips"
)

var (
	// SupportedMimeTypes contains an all mimetypes which are supported by the thumbnailer.
	SupportedMimeTypes = map[string]struct{}{
		"image/png":                         {},
		"image/jpg":                         {},
		"image/jpeg":                        {},
		"image/gif":                         {},
		"image/bmp":                         {},
		"image/x-ms-bmp":                    {},
		"image/tiff":                        {},
		"text/plain":                        {},
		"audio/flac":                        {},
		"audio/mpeg":                        {},
		"audio/ogg":                         {},
		"application/vnd.geogebra.slides":   {},
		"application/vnd.geogebra.pinboard": {},
		"image/webp":                        {},
	}

	// heifMimeTypes are the mimetypes read by the libvips heif loader, which
	// decodes them as HEVC. They are registered only if that loader is there, see
	// registerOptionalMimeTypes.
	heifMimeTypes = []string{
		"image/heic",
		"image/heic-sequence",
		"image/heif",
		"image/heif-sequence",
	}

	// avifMimeTypes go through the same loader but are coded as AV1.
	avifMimeTypes = []string{
		"image/avif",
	}
)

func init() {
	// keep libvips as quiet as the preprocessor wants it. Looking for the loaders
	// below starts libvips and we cannot rely on the preprocessor init having run
	// first.
	vips.LoggingSettings(nil, vips.LogLevelError)
	registerOptionalMimeTypes()
}

// registerOptionalMimeTypes adds the mimetypes which libvips can only read
// through a loader that is not part of libvips itself.
// E.g. OpenCloud does not ship a HEVC decoder because of the patent situation
// (see Readme.md). Registering the mimetypes only when the loader is really
// there keeps the `oc:has-preview` flag and the graph thumbnail links in sync
// with what the thumbnails service can deliver.
func registerOptionalMimeTypes() {
	if vips.IsTypeSupported(vips.ImageTypeHEIF) {
		for _, mimeType := range heifMimeTypes {
			SupportedMimeTypes[mimeType] = struct{}{}
		}
	}
	if vips.IsTypeSupported(vips.ImageTypeAVIF) {
		for _, mimeType := range avifMimeTypes {
			SupportedMimeTypes[mimeType] = struct{}{}
		}
	}
}
