package svc

import (
	"bytes"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"testing"
)

// craftDimensionBomb encodes a tiny valid grayscale JPEG, then overwrites the
// SOF0 width/height so the header declares huge dimensions while the payload
// stays tiny. Feeding this to the endpoint must be rejected by the shared
// header check in push.go before any pixel buffer is allocated, regardless of
// which imaging backend is compiled in.
func craftDimensionBomb(t *testing.T, width, height uint16) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewGray(image.Rect(0, 0, 8, 8)), &jpeg.Options{Quality: 10}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	b := buf.Bytes()
	for i := 0; i+9 < len(b); i++ {
		if b[i] == 0xff && b[i+1] == 0xc0 {
			b[i+5], b[i+6] = byte(height>>8), byte(height)
			b[i+7], b[i+8] = byte(width>>8), byte(width)
			return b
		}
	}
	t.Fatal("no SOF0 marker in encoded jpeg")
	return nil
}

// TestPushEndpoint_DimensionBombJPEG pins the OOM-prevention property at its new
// home: a tiny file whose JPEG header declares dimensions beyond the configured
// max must be rejected with 422 by the shared guard, before processImage runs.
func TestPushEndpoint_DimensionBombJPEG(t *testing.T) {
	mux := newTestMuxWithLimits(7680, 7680)

	bomb := craftDimensionBomb(t, 20000, 20000)
	body, contentType := createMultipartBody(bomb)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/64x64/filters:format(png)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422 for dimension bomb, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestPushEndpoint_DimensionBombGIF is the gif variant of the same property: a
// gif whose logical screen descriptor declares an oversized canvas must be
// rejected with 422 before decoding.
func TestPushEndpoint_DimensionBombGIF(t *testing.T) {
	mux := newTestMuxWithLimits(7680, 7680)

	// LSD declares a 20000x20000 logical screen; the rest is minimal padding.
	g := []byte("GIF89a")
	g = append(g, 0x20, 0x4e, 0x20, 0x4e, 0xf0, 0x00, 0x00) // 20000x20000, gct flag
	g = append(g, bytes.Repeat([]byte{0}, 6)...)            // minimal gct + terminator-ish

	body, contentType := createMultipartBody(g)

	req := httptest.NewRequest(http.MethodPost, "/unsafe/64x64/filters:format(gif)/", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422 for oversized gif dimension bomb, got %d: %s", rec.Code, rec.Body.String())
	}
}
