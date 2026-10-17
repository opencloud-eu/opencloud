package svc

import (
	"bytes"
	"errors"
	"fmt"
	"image/gif"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// gifMagic is the leading signature of a GIF file (GIF87a or GIF89a). It decides
// whether a request takes the animation path.
var gifMagic = []byte("GIF8")

// Operations understood by the push endpoint. They map to imagor's resize/crop
// modes so a real imagor deployment can replace this service.
const (
	OpFill    = "fill"
	OpFitIn   = "fit-in"
	OpStretch = "stretch"
)

// errFormatUnsupported is what an encoder returns for a format this build has no
// encoder for, as opposed to an encode that went wrong.
var errFormatUnsupported = fmt.Errorf("format not supported by this build")

// pushHandler handles POST requests for the imagor-like push-based thumbnail endpoint.
// Path patterns (operation and format are both optional; absent operation is the default
// center-crop fill, absent format keeps the input's own format):
//   - POST /unsafe(/fit-in|/stretch)/{W}x{H}(/filters:{filters})
//
// The filters segment is captured whole and parsed by outputFormatFromFilters; only the
// format filter is meaningful to this executor, other filters are ignored. When no format
// filter is present the input's own format is preserved, as imagor does it, and the
// backend that will decode the image says which format that is. The answers are imagor's:
// 400 for a malformed request, 406 for a source it cannot read or a format it cannot
// write, 422 for an image above the configured maximum dimensions, 429 when the
// concurrency limit is reached.
func (s Thumbnails) pushHandler(w http.ResponseWriter, r *http.Request) {
	ok, retryAfter := s.limiter.acquire()
	if !ok {
		writeTooManyRequests(w, retryAfter)
		return
	}
	defer s.limiter.release()

	width, err := parseDim(r, "width")
	if err != nil {
		writeInvalid(w, "invalid width")
		return
	}
	height, err := parseDim(r, "height")
	if err != nil {
		writeInvalid(w, "invalid height")
		return
	}

	operation := chi.URLParam(r, "operation")
	if operation == "" {
		operation = OpFill
	}
	switch operation {
	case OpFill, OpFitIn, OpStretch:
	default:
		writeInvalid(w, fmt.Sprintf("unsupported operation: %s", operation))
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeInvalid(w, "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		writeInvalid(w, "missing image field in form data")
		return
	}
	defer file.Close()

	// imagor's upload loader accepts image/* only (UPLOAD_LOADER_ACCEPT), so an
	// untyped part is refused here too.
	if ct := header.Header.Get("Content-Type"); !strings.HasPrefix(ct, "image/") {
		writeNotAcceptable(w, fmt.Sprintf("unsupported part type: %q", ct))
		return
	}

	imgData, err := io.ReadAll(file)
	if err != nil {
		writeInvalid(w, "failed to read image")
		return
	}

	filtersSegment := filtersFromTail(chi.URLParam(r, "*"))
	ext, hasFormat, err := outputFormatFromFilters(filtersSegment)
	if err != nil {
		writeInvalid(w, err.Error())
		return
	}
	noUpscale := filterPresent(filtersSegment, "no_upscale")
	// The box and operation are chosen by webdav (the sizing brain); this service
	// is a dumb, imagor-like executor that just fits within the given box.
	req := resizeRequest{Width: width, Height: height, Operation: operation, NoUpscale: noUpscale}

	// An animated gif is the one source both builds treat the same way: neither
	// libvips nor the pure Go path has anything better for a paletted animation,
	// so it never reaches a backend. Asking for a still format flattens it, and
	// that is a backend's job again.
	if bytes.HasPrefix(imgData, gifMagic) && (ext == "gif" || !hasFormat) {
		s.serveAnimation(w, imgData, req)

		return
	}

	// The backend that decodes the image is the one that answers for it: it reads
	// the header here, and only the resize pulls pixels.
	src, err := loadSource(imgData)
	if err != nil {
		s.logger.Debug().Err(err).Msg("could not read the image header")
		writeNotAcceptable(w, "unreadable image")

		return
	}
	defer src.Close()

	// imagor ErrMaxResolutionExceeded: the declared input exceeds the configured
	// maximum width or height. Checked on the header BEFORE any pixels exist, so
	// a dimension bomb (a tiny file declaring huge dimensions) is rejected
	// without allocating a frame buffer. webdav translates the resulting 422 into
	// its legacy 403 response.
	srcWidth, srcHeight := src.Width(), src.Height()
	if (s.maxWidth > 0 && srcWidth > s.maxWidth) || (s.maxHeight > 0 && srcHeight > s.maxHeight) {
		writeMaxResolution(w)

		return
	}

	if !hasFormat {
		// No format filter: preserve the input's own format, the way imagor does.
		ext = extForSourceFormat(src)
	}

	processed, err := resizeSource(src, req)
	if err != nil {
		writeNotAcceptable(w, "failed to process image")
		return
	}

	var (
		buf         bytes.Buffer
		contentType string
		encErr      error
	)
	switch ext {
	case "jpg":
		contentType = "image/jpeg"
		encErr = encodeJPEG(&buf, processed)
	case "png":
		contentType = "image/png"
		encErr = encodePNG(&buf, processed)
	case "webp":
		contentType = "image/webp"
		encErr = encodeWebP(&buf, processed)
	case "gif":
		contentType = "image/gif"
		encErr = encodeGIF(&buf, processed)
	default:
		// unreachable today: the filters whitelist and the backends both answer
		// in jpg, png, gif or webp. A format added to one of them without an
		// encoder lands here, and that is this build's limit, not a bad request.
		writeNotAcceptable(w, fmt.Sprintf("this build writes no %s", ext))

		return
	}
	if errors.Is(encErr, errFormatUnsupported) {
		// 406, not 400: the request is fine, this build just cannot answer it,
		// and webdav reads a 400 as its own bug
		writeNotAcceptable(w, fmt.Sprintf("this build writes no %s", ext))
		return
	}
	if encErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "failed to encode image")
		return
	}

	writeImage(w, contentType, buf.Bytes())
}

// serveAnimation resizes every frame of an animated gif and writes it back. The
// guard runs on the gif header first, so a gif that only declares huge frames
// never reaches the frame decode.
func (s Thumbnails) serveAnimation(w http.ResponseWriter, imgData []byte, req resizeRequest) {
	config, err := gif.DecodeConfig(bytes.NewReader(imgData))
	if err != nil {
		s.logger.Debug().Err(err).Msg("could not read the gif header")
		writeNotAcceptable(w, "unreadable image")

		return
	}

	if (s.maxWidth > 0 && config.Width > s.maxWidth) || (s.maxHeight > 0 && config.Height > s.maxHeight) {
		writeMaxResolution(w)

		return
	}

	g, err := gif.DecodeAll(bytes.NewReader(imgData))
	if err != nil || len(g.Image) == 0 {
		s.logger.Debug().Err(err).Msg("could not decode the gif")
		writeNotAcceptable(w, "unreadable image")

		return
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, resizeGIF(g, req)); err != nil {
		s.logger.Error().Err(err).Msg("could not encode the gif")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "failed to encode image")

		return
	}

	writeImage(w, "image/gif", buf.Bytes())
}

// writeImage answers with the encoded image.
func writeImage(w http.ResponseWriter, contentType string, data []byte) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func parseDim(r *http.Request, name string) (int, error) {
	v := chi.URLParam(r, name)
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid %s: %s", name, v)
	}

	return n, nil
}

// filtersFromTail reads the filters out of imagor's path tail. The tail is the
// optional "filters:..." segment plus an optional image key behind it; only the
// filters segment is ours, and it may be empty.
func filtersFromTail(tail string) string {
	segment, _, _ := strings.Cut(tail, "/")
	filters, ok := strings.CutPrefix(segment, "filters:")
	if !ok {
		return ""
	}

	return filters
}

// writeNotAcceptable responds with imagor's status for a source it cannot read
// (406 Not Acceptable), as opposed to a malformed request.
func writeNotAcceptable(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusNotAcceptable)
	fmt.Fprintln(w, msg)
}

// writeInvalid responds with imagor's ErrInvalid status (400 Bad Request).
func writeInvalid(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintln(w, msg)
}

// writeMaxResolution responds with imagor's ErrMaxResolutionExceeded status
// (422 Unprocessable Entity).
func writeMaxResolution(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnprocessableEntity)
	fmt.Fprintln(w, "maximum resolution exceeded")
}

// concurrencyLimiter bounds the number of requests processed in parallel. It is
// nil when no limit is configured (THUMBNAILS_MAX_CONCURRENT_REQUESTS = 0).
type concurrencyLimiter struct {
	tokens chan struct{}
}

func newConcurrencyLimiter(limit int) *concurrencyLimiter {
	if limit <= 0 {
		return nil
	}
	tokens := make(chan struct{}, limit)
	for i := 0; i < limit; i++ {
		tokens <- struct{}{}
	}
	return &concurrencyLimiter{tokens: tokens}
}

// acquire tries to take a slot without blocking. It reports whether the request
// may proceed and, when not, how many seconds to wait before retrying.
func (l *concurrencyLimiter) acquire() (bool, int) {
	if l == nil {
		return true, 0
	}
	select {
	case <-l.tokens:
		return true, 0
	default:
		return false, 1
	}
}

func (l *concurrencyLimiter) release() {
	if l == nil {
		return
	}
	l.tokens <- struct{}{}
}

// writeTooManyRequests responds with 429 and a short Retry-After header.
func writeTooManyRequests(w http.ResponseWriter, retryAfter int) {
	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	w.WriteHeader(http.StatusTooManyRequests)
	fmt.Fprintln(w, "too many concurrent requests")
}

// mapFormatToExt maps imagor format names to the internal extension used by encoders.
func mapFormatToExt(format string) string {
	switch strings.ToLower(format) {
	case "jpeg":
		return "jpg"
	default:
		return strings.ToLower(format)
	}
}

// outputFormatFromFilters parses an imagor filters segment (e.g. "format(jpeg)") and
// reports the requested output extension. Only the format filter is meaningful to this
// executor; other filters are ignored because the operation already governs
// sizing/upscaling. It returns hasFormat=false when no format filter is present, in
// which case the caller keeps the input's own format. An explicit but
// unsupported format is an error.
func outputFormatFromFilters(segment string) (ext string, hasFormat bool, err error) {
	for _, token := range strings.Split(segment, ":") {
		name, args, ok := splitFilter(token)
		if !ok || name != "format" {
			continue
		}
		mapped := mapFormatToExt(args)
		switch mapped {
		case "jpg", "png", "gif", "webp":
			return mapped, true, nil
		default:
			return "", false, fmt.Errorf("unsupported output format: %s", args)
		}
	}
	return "", false, nil
}

// splitFilter splits a "name(args)" token into name and args. It returns ok=false for
// tokens that are not in parenthesized form.
func splitFilter(token string) (name, args string, ok bool) {
	i := strings.IndexByte(token, '(')
	if i < 0 || !strings.HasSuffix(token, ")") {
		return "", "", false
	}
	return token[:i], token[i+1 : len(token)-1], true
}

// filterPresent reports whether a named filter occurs in an imagor filters
// segment (e.g. "no_upscale():format(jpeg)").
func filterPresent(segment, name string) bool {
	for _, token := range strings.Split(segment, ":") {
		if n, _, ok := splitFilter(token); ok && n == name {
			return true
		}
	}
	return false
}

// resizeRequest is what the executor is asked to do. webdav chose the box and
// the operation; noUpscale mirrors imagor's no_upscale() filter.
type resizeRequest struct {
	Width     int
	Height    int
	Operation string
	NoUpscale bool
}
