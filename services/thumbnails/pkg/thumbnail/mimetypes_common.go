package thumbnail

// EmbeddedPreviewMimeTypes have an embedded preview (e.g. audio cover art) that
// may be absent; availability is decided per file (see HasPreview).
var EmbeddedPreviewMimeTypes = map[string]struct{}{
	"audio/flac": {},
	"audio/mpeg": {},
	"audio/ogg":  {},
}

// SupportedMimeTypes is the union of both preview sets; the generator gates on it.
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
