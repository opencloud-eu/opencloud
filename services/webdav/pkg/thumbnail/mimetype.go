package thumbnail

import "mime"

// IsMimeTypeSupported validates if the mime type is supported.
func IsMimeTypeSupported(m string) bool {
	mimeType, _, err := mime.ParseMediaType(m)
	if err != nil {
		return false
	}
	_, supported := SupportedMimeTypes[mimeType]
	return supported
}
