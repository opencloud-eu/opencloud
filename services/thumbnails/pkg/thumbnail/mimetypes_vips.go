//go:build enable_vips

package thumbnail

// UnconditionalPreviewMimeTypes always have a preview: the thumbnailer renders
// one from the content.
var UnconditionalPreviewMimeTypes = map[string]struct{}{
	"image/png":                         {},
	"image/jpg":                         {},
	"image/jpeg":                        {},
	"image/gif":                         {},
	"image/bmp":                         {},
	"image/x-ms-bmp":                    {},
	"image/tiff":                        {},
	"image/webp":                        {},
	"text/plain":                        {},
	"application/vnd.geogebra.slides":   {},
	"application/vnd.geogebra.pinboard": {},
}
