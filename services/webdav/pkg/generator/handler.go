package generator

import (
	"net/http"
)

// WriteThumbnailResponse writes the thumbnail bytes to the HTTP response with correct Content-Type.
func WriteThumbnailResponse(w http.ResponseWriter, data []byte, ext string) {
	ct := ContentType(ext)
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
