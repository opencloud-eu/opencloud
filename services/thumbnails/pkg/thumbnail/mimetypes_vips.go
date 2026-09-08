//go:build enable_vips

package thumbnail

// SupportedMimeTypes contains all mimetypes which are supported by the thumbnailer.
var SupportedMimeTypes = map[string]struct{}{
	"image/png":                         {},
	"image/jpg":                         {},
	"image/jpeg":                        {},
	"image/gif":                         {},
	"image/bmp":                         {},
	"image/x-ms-bmp":                    {},
	"image/tiff":                        {},
	"image/webp":                        {},
	"text/plain":                        {},
	"audio/flac":                        {},
	"audio/mpeg":                        {},
	"audio/ogg":                         {},
	"application/vnd.geogebra.slides":   {},
	"application/vnd.geogebra.pinboard": {},
}
