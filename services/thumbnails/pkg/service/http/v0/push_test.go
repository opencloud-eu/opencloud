package svc

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"golang.org/x/image/bmp"
)

func createTestImage(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func createMultipartBody(imgBytes []byte) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "test.png")
	if err != nil {
		panic(err)
	}
	if _, err := part.Write(imgBytes); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	return body, writer.FormDataContentType()
}

func newTestMux() *chi.Mux {
	return newTestMuxWithLimits(0, 0)
}

// newTestMuxWithLimits builds the route table with explicit max width/height so
// tests can exercise the ErrMaxResolutionExceeded (422) path. A limit of 0
// disables the corresponding bound.
func newTestMuxWithLimits(maxWidth, maxHeight int) *chi.Mux {
	mux := chi.NewRouter()
	h := Thumbnails{maxWidth: maxWidth, maxHeight: maxHeight}.pushHandler
	mux.Post("/unsafe/{operation}/{width}x{height}/filters:{filters}/", h)
	mux.Post("/unsafe/{operation}/{width}x{height}/filters:{filters}", h)
	mux.Post("/unsafe/{width}x{height}/filters:{filters}/", h)
	mux.Post("/unsafe/{width}x{height}/filters:{filters}", h)
	return mux
}

// createTestGIF returns an animated gif with the given number of frames, each
// frame a solid color so that animation preservation can be verified.
func createTestGIF(width, height, frames int) []byte {
	pal := color.Palette{color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}, color.RGBA{0, 0, 255, 255}}
	g := &gif.GIF{}
	for i := 0; i < frames; i++ {
		frame := image.NewPaletted(image.Rect(0, 0, width, height), pal)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				frame.SetColorIndex(x, y, uint8(i%len(pal)))
			}
		}
		g.Image = append(g.Image, frame)
		g.Delay = append(g.Delay, 10)
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestPushEndpoint_FitIn_JPEG(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestImage(200, 200)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/100x100/filters:format(jpeg)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("expected Content-Type image/jpeg, got %s", ct)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestPushEndpoint_FitIn_PNG(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestImage(200, 200)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/100x100/filters:format(png)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("expected Content-Type image/png, got %s", ct)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestPushEndpoint_FitIn_GIF(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestGIF(200, 200, 3)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/100x100/filters:format(gif)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/gif" {
		t.Errorf("expected Content-Type image/gif, got %s", ct)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected non-empty response body")
	}

	out, err := gif.DecodeAll(rec.Body)
	if err != nil {
		t.Fatalf("failed to decode response gif: %v", err)
	}
	if len(out.Image) != 3 {
		t.Errorf("expected 3 frames preserved, got %d", len(out.Image))
	}
}

func TestPushEndpoint_FitIn_GIFPreservesAnimation(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestGIF(200, 100, 4)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/50x50/filters:format(gif)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	out, err := gif.DecodeAll(rec.Body)
	if err != nil {
		t.Fatalf("failed to decode response gif: %v", err)
	}
	if len(out.Image) != 4 {
		t.Errorf("expected 4 frames preserved, got %d", len(out.Image))
	}
	// fit-in must not enlarge the 200x100 source to 50x50 (fit-in never upscales).
	for i, frame := range out.Image {
		b := frame.Bounds()
		if b.Dx() > 200 || b.Dy() > 100 {
			t.Errorf("frame %d exceeds source dimensions: %dx%d", i, b.Dx(), b.Dy())
		}
	}
}

func TestPushEndpoint_Fill_GIF(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestGIF(200, 100, 3)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/64x64/filters:format(gif)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	out, err := gif.DecodeAll(rec.Body)
	if err != nil {
		t.Fatalf("failed to decode response gif: %v", err)
	}
	if len(out.Image) != 3 {
		t.Errorf("expected 3 frames preserved, got %d", len(out.Image))
	}
	for i, frame := range out.Image {
		b := frame.Bounds()
		if b.Dx() != 64 || b.Dy() != 64 {
			t.Errorf("fill should produce exact dimensions, frame %d is %dx%d", i, b.Dx(), b.Dy())
		}
	}
}

func TestPushEndpoint_FitIn_AspectRatio(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestImage(400, 200)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/100x100/filters:format(png)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	img, err := png.Decode(rec.Body)
	if err != nil {
		t.Fatalf("failed to decode response image: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() > 100 || bounds.Dy() > 100 {
		t.Errorf("fit-in should not exceed requested dimensions, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestPushEndpoint_FitIn_NoUpscale(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestImage(50, 50)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/200x200/filters:format(png)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	img, err := png.Decode(rec.Body)
	if err != nil {
		t.Fatalf("failed to decode response image: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() > 50 || bounds.Dy() > 50 {
		t.Errorf("fit-in should not enlarge the image, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

// TestPushEndpoint_Default_SquareFill pins the default processor's outcome:
// center-crop to fill the requested box exactly (the legacy "thumbnail"
// processor behavior). A 200x100 source requested at 100x100 must come back as
// exactly 100x100 — center-cropped, NOT letterboxed to 100x50. This is the route
// the webdav service uses by default (no fit-in segment).
func TestPushEndpoint_Default_SquareFill(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestImage(200, 100)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/100x100/filters:format(png)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	img, err := png.Decode(rec.Body)
	if err != nil {
		t.Fatalf("failed to decode response image: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("default should center-crop 200x100 into a 100x100 box as 100x100, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

// TestPushEndpoint_Default_Upscales documents that the default (fill) processor
// upscales small sources to fill the box exactly, matching the legacy
// "thumbnail" processor. A 50x50 source requested at 200x200 must come back as
// exactly 200x200.
func TestPushEndpoint_Default_Upscales(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestImage(50, 50)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/200x200/filters:format(png)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	img, err := png.Decode(rec.Body)
	if err != nil {
		t.Fatalf("failed to decode response image: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 200 || bounds.Dy() != 200 {
		t.Errorf("default should upscale 50x50 to fill a 200x200 box as 200x200, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

// TestPushEndpoint_FitIn_LetterboxOutcome pins the opt-in "fit" processor's
// outcome: aspect-preserving fit with no upscaling. A 200x100 source requested
// at 100x100 must come back as exactly 100x50 — centered/letterboxed in the box,
// NOT center-cropped to 100x100.
func TestPushEndpoint_FitIn_LetterboxOutcome(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestImage(200, 100)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/100x100/filters:format(png)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	img, err := png.Decode(rec.Body)
	if err != nil {
		t.Fatalf("failed to decode response image: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 50 {
		t.Errorf("fit-in should letterbox 200x100 into a 100x100 box as 100x50, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestPushEndpoint_Fill_JPEG(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestImage(200, 200)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/100x100/filters:format(jpeg)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("expected Content-Type image/jpeg, got %s", ct)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestPushEndpoint_Fill_ExactDimensions(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestImage(400, 200)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/100x100/filters:format(png)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	img, err := png.Decode(rec.Body)
	if err != nil {
		t.Fatalf("failed to decode response image: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("fill should produce exact dimensions, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestPushEndpoint_BadDimensions(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestImage(100, 100)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/abcxdef/filters:format(jpeg)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestPushEndpoint_ZeroDimensions(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestImage(100, 100)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/0x0/filters:format(jpeg)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for zero dimensions, got %d", rec.Code)
	}
}

func TestPushEndpoint_UnsupportedFormat(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestImage(100, 100)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/50x50/filters:format(webp)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for unsupported format, got %d", rec.Code)
	}
}

func TestPushEndpoint_InvalidImageData(t *testing.T) {
	mux := newTestMux()

	body, contentType := createMultipartBody([]byte("not a valid image"))

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/50x50/filters:format(jpeg)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid image data, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPushEndpoint_MissingImageField(t *testing.T) {
	mux := newTestMux()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("other_field", "some value")
	if err := writer.Close(); err != nil {
		panic(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/50x50/filters:format(jpeg)/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing image field, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPushEndpoint_NotMultipart(t *testing.T) {
	mux := newTestMux()

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/50x50/filters:format(jpeg)/", bytes.NewReader([]byte("plain text")))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for non-multipart request, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPushEndpoint_NoTrailingSlash(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestImage(200, 200)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/100x100/filters:format(png)", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 without trailing slash, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("expected Content-Type image/png, got %s", ct)
	}
}

// TestPushEndpoint_Stretch pins the stretch operation: resize to the exact box
// without preserving aspect ratio. A 200x100 source requested at 64x32 must come
// back as exactly 64x32 (distorted), not letterboxed or cropped.
func TestPushEndpoint_Stretch(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestImage(200, 100)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/stretch/64x32/filters:format(png)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	img, err := png.Decode(rec.Body)
	if err != nil {
		t.Fatalf("failed to decode response image: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 64 || bounds.Dy() != 32 {
		t.Errorf("stretch should produce exactly 64x32, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

// TestPushEndpoint_MaxResolutionExceeded pins the ErrMaxResolutionExceeded path:
// a decoded input exceeding the configured max width/height returns 422.
func TestPushEndpoint_MaxResolutionExceeded(t *testing.T) {
	mux := newTestMuxWithLimits(100, 100)

	imgBytes := createTestImage(500, 500)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/64x64/filters:format(png)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422 for oversized input, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestPushEndpoint_WithinMaxResolution confirms an input within the configured
// limits still succeeds (the 422 check must not over-trigger).
func TestPushEndpoint_WithinMaxResolution(t *testing.T) {
	mux := newTestMuxWithLimits(100, 100)

	imgBytes := createTestImage(50, 50)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/32x32/filters:format(png)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 for in-bounds input, got %d: %s", rec.Code, rec.Body.String())
	}
}

func createTestJPEG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func createTestBMP(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := bmp.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// TestPushEndpoint_ToleratesExtraFilters proves the generic filters parser ignores
// filters other than format (a real imagor client may send no_upscale or others).
func TestPushEndpoint_ToleratesExtraFilters(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestImage(200, 200)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/100x100/filters:no_upscale():format(jpeg)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("expected Content-Type image/jpeg, got %s", ct)
	}
}

// TestPushEndpoint_NoFormat_FallbackGIF pins the 1c pass-through fallback: a filters
// segment with no format filter (here just no_upscale) keeps a gif input as gif.
func TestPushEndpoint_NoFormat_FallbackGIF(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestGIF(200, 200, 2)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/100x100/filters:no_upscale()/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/gif" {
		t.Errorf("expected Content-Type image/gif, got %s", ct)
	}
}

// TestPushEndpoint_NoFormat_FallbackPNG pins the 1c pass-through fallback: a filters
// segment with no format filter keeps a png input as png.
func TestPushEndpoint_NoFormat_FallbackPNG(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestImage(200, 200)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/100x100/filters:no_upscale()/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("expected Content-Type image/png, got %s", ct)
	}
}

// TestPushEndpoint_NoFormat_FallbackJPEG pins the 1c pass-through fallback: a filters
// segment with no format filter keeps a jpeg input as jpeg.
func TestPushEndpoint_NoFormat_FallbackJPEG(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestJPEG(200, 200)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/100x100/filters:no_upscale()/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("expected Content-Type image/jpeg, got %s", ct)
	}
}

// TestPushEndpoint_NoFormat_UnsupportedInputFallsBackToJPEG pins imagor's unsavable
// source behavior: an input we cannot re-encode (here a BMP) with no format filter is
// returned as JPEG, matching imagor's default for formats outside its savable set.
func TestPushEndpoint_NoFormat_UnsupportedInputFallsBackToJPEG(t *testing.T) {
	mux := newTestMux()

	imgBytes := createTestBMP(200, 200)
	body, contentType := createMultipartBody(imgBytes)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/fit-in/100x100/filters:no_upscale()/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("expected Content-Type image/jpeg, got %s", ct)
	}
}

func TestInputFormatViaImaging(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"jpeg", createTestJPEG(16, 16), "jpg"},
		{"png", createTestImage(16, 16), "png"},
		{"gif", createTestGIF(16, 16, 2), "gif"},
		{"bmp falls back to jpeg", createTestBMP(16, 16), "jpg"},
		{"undecodable falls back to jpeg", []byte("not an image at all"), "jpg"},
		{"empty falls back to jpeg", []byte{}, "jpg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inputFormatViaImaging(tt.data); got != tt.want {
				t.Errorf("inputFormatViaImaging() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitFilter(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		wantName string
		wantArgs string
		wantOK   bool
	}{
		{"format", "format(jpeg)", "format", "jpeg", true},
		{"no args name", "no_upscale()", "no_upscale", "", true},
		{"no parens", "plain", "", "", false},
		{"open only", "format(jpeg", "", "", false},
		{"close only", "formatjpeg)", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, args, ok := splitFilter(tt.token)
			if ok != tt.wantOK || name != tt.wantName || args != tt.wantArgs {
				t.Errorf("splitFilter(%q) = (%q,%q,%v), want (%q,%q,%v)",
					tt.token, name, args, ok, tt.wantName, tt.wantArgs, tt.wantOK)
			}
		})
	}
}

func TestOutputFormatFromFilters(t *testing.T) {
	tests := []struct {
		name    string
		segment string
		wantExt string
		wantHas bool
		wantErr bool
	}{
		{"format jpeg", "format(jpeg)", "jpg", true, false},
		{"extra filters ignored", "no_upscale():format(png)", "png", true, false},
		{"unsupported format", "format(webp)", "", false, true},
		{"no format filter", "no_upscale()", "", false, false},
		{"empty segment", "", "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, has, err := outputFormatFromFilters(tt.segment)
			if (err != nil) != tt.wantErr {
				t.Fatalf("outputFormatFromFilters() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.wantExt || has != tt.wantHas {
				t.Errorf("outputFormatFromFilters() = (%q,%v), want (%q,%v)", got, has, tt.wantExt, tt.wantHas)
			}
		})
	}
}
