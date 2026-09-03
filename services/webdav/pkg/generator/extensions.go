package generator

import (
	"mime"
	"strings"
)

// ExtensionInfo holds the output format and content type for a given file extension.
type ExtensionInfo struct {
	OutputFormat string // generator output format (gif, png, jpeg)
	ContentType  string // HTTP Content-Type header value
}

var extMap = map[string]ExtensionInfo{
	"gif":  {OutputFormat: "gif", ContentType: "image/gif"},
	"png":  {OutputFormat: "png", ContentType: "image/png"},
	"jpeg": {OutputFormat: "jpeg", ContentType: "image/jpeg"},
	"jpg":  {OutputFormat: "jpeg", ContentType: "image/jpeg"},
}

var defaultExtInfo = ExtensionInfo{OutputFormat: "jpeg", ContentType: "image/jpeg"}

// GetExtensionInfo returns the extension info for a given file extension.
func GetExtensionInfo(ext string) ExtensionInfo {
	if info, ok := extMap[strings.ToLower(strings.TrimLeft(ext, "."))]; ok {
		return info
	}
	return defaultExtInfo
}

// OutputFormat returns the generator output format for a given file extension.
func OutputFormat(ext string) string {
	return GetExtensionInfo(ext).OutputFormat
}

// ContentType returns the HTTP Content-Type for a given file extension.
func ContentType(ext string) string {
	return GetExtensionInfo(ext).ContentType
}

// GuessExtension guesses a file extension from a MIME type.
func GuessExtension(mimeType string) string {
	exts := []string{".jpg", ".png", ".gif"}
	for _, ext := range exts {
		if strings.HasSuffix(mimeType, "/"+strings.TrimLeft(ext, ".")) ||
			strings.Contains(mimeType, "jpeg") && ext == ".jpg" {
			return ext[1:]
		}
	}

	switch {
	case strings.Contains(mimeType, "png"):
		return "png"
	case strings.Contains(mimeType, "gif"):
		return "gif"
	default:
		return "jpg"
	}
}

// ExtForMime mirrors main's GetExtForMime: returns the output extension for a
// source mime, or "" when the source has no dedicated image type (txt, audio,
// bmp, tiff) so the caller falls back to the requested extension.
func ExtForMime(mimeType string) string {
	m, _, _ := mime.ParseMediaType(mimeType)
	switch m {
	case "image/jpeg", "image/webp":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	default:
		return ""
	}
}
