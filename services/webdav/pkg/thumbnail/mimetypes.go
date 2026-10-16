package thumbnail

var (
	// SupportedMimeTypes contains all mimetypes a preview is made of. Whether
	// the generator can read one is its own business: webdav does not decode an
	// image, so a build without libvips answers 400 for a webp instead of
	// refusing it here.
	SupportedMimeTypes = map[string]struct{}{
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
)
