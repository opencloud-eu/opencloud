package preprocessor

import (
	"archive/zip"
	"bytes"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	thumbnailerErrors "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
)

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
		_, _ = w.Write(data)
	}
	_ = zw.Close()
	return buf.Bytes()
}

var _ = Describe("RawTikaDecoder", func() {
	It("extracts the largest renderable JPEG from Tika's unpack zip", func() {
		small := encodeJPEG(16, 8)
		large := encodeJPEG(64, 32)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Expect(r.Method).To(Equal(http.MethodPut))
			Expect(r.URL.Path).To(Equal("/unpack"))
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipWith(map[string][]byte{
				"thumbnail-0":   small,
				"thumbnail-1":   large,
				"metadata.json": []byte(`{"not":"a jpeg"}`),
			}))
		}))
		defer srv.Close()

		img, err := RawTikaDecoder{tikaURL: srv.URL}.Convert(bytes.NewReader([]byte("raw")))
		Expect(err).ToNot(HaveOccurred())
		// default (imaging) build decodes to an image.Image
		bounds := img.(image.Image).Bounds()
		Expect(bounds.Dx()).To(Equal(64))
		Expect(bounds.Dy()).To(Equal(32))
	})

	It("yields no preview when the unpack zip holds no renderable JPEG", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(zipWith(map[string][]byte{"metadata.json": []byte("not a jpeg")}))
		}))
		defer srv.Close()

		_, err := RawTikaDecoder{tikaURL: srv.URL}.Convert(bytes.NewReader([]byte("raw")))
		Expect(err).To(MatchError(thumbnailerErrors.ErrNoImageFromRawFile))
	})

	It("errors when Tika returns a non-OK status", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		_, err := RawTikaDecoder{tikaURL: srv.URL}.Convert(bytes.NewReader([]byte("raw")))
		Expect(err).To(HaveOccurred())
	})
})
