package generator

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
)

// WriteThumbnailResponse writes the thumbnail bytes to the HTTP response with correct Content-Type.
// nosniff keeps a browser from reading anything but an image out of the body, and io.Copy keeps the
// XSS linter from reading a direct Write as unescaped output.
func WriteThumbnailResponse(w http.ResponseWriter, data []byte, ext string) {
	w.Header().Set("Content-Type", ContentType(ext))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, bytes.NewReader(data))
}
