//go:build !enable_vips

package thumbnail

// UnconditionalPreviewMimeTypes are mimetypes whose preview availability follows
// from the mimetype alone: the thumbnailer can always render a preview from the
// content, so a preview is guaranteed to exist.
var UnconditionalPreviewMimeTypes = map[string]struct{}{
	"image/png":                         {},
	"image/jpg":                         {},
	"image/jpeg":                        {},
	"image/gif":                         {},
	"image/bmp":                         {},
	"image/x-ms-bmp":                    {},
	"image/tiff":                        {},
	"text/plain":                        {},
	"application/vnd.geogebra.slides":   {},
	"application/vnd.geogebra.pinboard": {},
}
