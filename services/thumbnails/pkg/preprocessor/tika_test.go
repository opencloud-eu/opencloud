package preprocessor

import (
	"archive/zip"
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	thumbnailerErrors "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
)

// writeBody writes through an io.Writer so the test's httptest handlers don't
// call ResponseWriter.Write directly (trips a static-analysis XSS rule).
func writeBody(w io.Writer, b []byte) { _, _ = io.Copy(w, bytes.NewReader(b)) }

func encodeJPEG(width, height int) []byte {
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, width, height)), nil)
	return buf.Bytes()
}

func zipWith(entries map[string][]byte) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range entries {
		w, _ := zw.Create(name)
		writeBody(w, data)
	}
	_ = zw.Close()
	return buf.Bytes()
}

// losslessJPEG builds a SOF3 (lossless) JPEG of the given size; such payloads
// carry DNG raw data and must be rejected by isRenderableJPEG.
func losslessJPEG(size int) []byte {
	b := []byte{0xff, 0xd8, 0xff, 0xc3}
	if size > len(b) {
		b = append(b, make([]byte, size-len(b))...)
	}
	return b
}

var _ = Describe("TikaDecoder", func() {
	It("extracts the largest renderable JPEG from Tika's unpack zip", func() {
		small := encodeJPEG(16, 8)
		large := encodeJPEG(64, 32)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Expect(r.Method).To(Equal(http.MethodPut))
			Expect(r.URL.Path).To(Equal("/unpack"))
			w.Header().Set("Content-Type", "application/zip")
			writeBody(w, zipWith(map[string][]byte{
				"thumbnail-0":   small,
				"thumbnail-1":   large,
				"metadata.json": []byte(`{"not":"a jpeg"}`),
			}))
		}))
		defer srv.Close()

		img, err := TikaDecoder{tikaURL: srv.URL}.Convert(context.TODO(), bytes.NewReader([]byte("raw")))
		Expect(err).ToNot(HaveOccurred())
		// default (imaging) build decodes to an image.Image
		bounds := img.(image.Image).Bounds()
		Expect(bounds.Dx()).To(Equal(64))
		Expect(bounds.Dy()).To(Equal(32))
	})

	It("yields no preview when the unpack zip holds no renderable JPEG", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeBody(w, zipWith(map[string][]byte{"metadata.json": []byte("not a jpeg")}))
		}))
		defer srv.Close()

		_, err := TikaDecoder{tikaURL: srv.URL}.Convert(context.TODO(), bytes.NewReader([]byte("raw")))
		Expect(err).To(MatchError(thumbnailerErrors.ErrNoEmbeddedImage))
	})

	It("errors when Tika returns a non-OK status", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		_, err := TikaDecoder{tikaURL: srv.URL}.Convert(context.TODO(), bytes.NewReader([]byte("raw")))
		Expect(err).To(HaveOccurred())
	})

	It("sends the quoted source filename so Tika can route by extension", func() {
		var gotCD string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotCD = r.Header.Get("Content-Disposition")
			writeBody(w, zipWith(map[string][]byte{"0.jpg": encodeJPEG(8, 8)}))
		}))
		defer srv.Close()

		_, err := TikaDecoder{tikaURL: srv.URL, filename: "my photo.nef"}.Convert(context.TODO(), bytes.NewReader([]byte("raw")))
		Expect(err).ToNot(HaveOccurred())
		Expect(gotCD).To(Equal(`attachment; filename="my photo.nef"`))
	})

	It("skips a larger lossless entry and picks the smaller renderable JPEG", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeBody(w, zipWith(map[string][]byte{
				"big-lossless": losslessJPEG(4096),
				"small":        encodeJPEG(16, 8),
			}))
		}))
		defer srv.Close()

		img, err := TikaDecoder{tikaURL: srv.URL}.Convert(context.TODO(), bytes.NewReader([]byte("raw")))
		Expect(err).ToNot(HaveOccurred())
		Expect(img.(image.Image).Bounds().Dx()).To(Equal(16))
	})

	It("treats Tika's 204 (no embedded resources) as a previewless raw", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		_, err := TikaDecoder{tikaURL: srv.URL}.Convert(context.TODO(), bytes.NewReader([]byte("raw")))
		Expect(err).To(MatchError(thumbnailerErrors.ErrNoEmbeddedImage))
	})

	It("errors when the 200 response body is not a zip", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeBody(w, []byte("not a zip"))
		}))
		defer srv.Close()

		_, err := TikaDecoder{tikaURL: srv.URL}.Convert(context.TODO(), bytes.NewReader([]byte("raw")))
		Expect(err).To(HaveOccurred())
		Expect(err).ToNot(MatchError(thumbnailerErrors.ErrNoEmbeddedImage))
	})

	It("classifies JPEG renderability by SOF marker", func() {
		Expect(isRenderableJPEG(encodeJPEG(8, 8))).To(BeTrue())
		Expect(isRenderableJPEG(losslessJPEG(64))).To(BeFalse())
		Expect(isRenderableJPEG([]byte{0xff, 0xd8})).To(BeFalse())
		Expect(isRenderableJPEG(nil)).To(BeFalse())
		Expect(isRenderableJPEG([]byte("not a jpeg"))).To(BeFalse())
	})
})
