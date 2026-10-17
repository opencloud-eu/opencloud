package thumbnail

// SupportedMimeTypes are the types a preview is announced and attempted for
// without further configuration. Both builds read all of them, webp included:
// the pure Go build decodes it through the imaging library and answers the JPEG
// the type mapping asks for. A generator that reads more than this is told so
// with OC_THUMBNAIL_SOURCE_TYPES.
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
