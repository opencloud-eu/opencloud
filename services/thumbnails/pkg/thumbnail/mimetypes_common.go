package thumbnail

// EmbeddedPreviewMimeTypes are mimetypes whose preview is an embedded resource
// (for example audio cover art) that may or may not be present. Preview
// availability cannot be derived from the mimetype alone and must be determined
// per file (see HasPreview).
var EmbeddedPreviewMimeTypes = map[string]struct{}{
	"audio/flac": {},
	"audio/mpeg": {},
	"audio/ogg":  {},
}

// SupportedMimeTypes contains all mimetypes the thumbnailer can produce a
// thumbnail for: the union of the unconditional and embedded preview types.
// The generator gates on this union; preview availability per file is decided
// by HasPreview.
var SupportedMimeTypes = func() map[string]struct{} {
	m := make(map[string]struct{}, len(UnconditionalPreviewMimeTypes)+len(EmbeddedPreviewMimeTypes))
	for k := range UnconditionalPreviewMimeTypes {
		m[k] = struct{}{}
	}
	for k := range EmbeddedPreviewMimeTypes {
		m[k] = struct{}{}
	}
	return m
}()
