package preprocessor

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
)

// presetZip builds what /unpack/preset/thumbnail returns: the one image the
// preset selected, and nothing else.
func presetZip(entries map[string][]byte) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range entries {
		w, err := zw.Create(name)
		Expect(err).ToNot(HaveOccurred())
		_, err = w.Write(data)
		Expect(err).ToNot(HaveOccurred())
	}
	Expect(zw.Close()).To(Succeed())
	return buf.Bytes()
}

func pngBytes() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	img.Set(1, 1, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	Expect(png.Encode(&buf, img)).To(Succeed())
	return buf.Bytes()
}

var _ = Describe("TikaThumbnail", func() {
	var (
		server    *httptest.Server
		requests  []*http.Request
		thumbnail func(w http.ResponseWriter, r *http.Request)
	)

	BeforeEach(func() {
		requests = nil
		thumbnail = func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests = append(requests, r)
			if r.URL.Path != "/unpack/preset/thumbnail" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			thumbnail(w, r)
		}))
	})

	AfterEach(func() {
		server.Close()
	})

	It("takes the image the preset unpacked, in one request", func() {
		thumbnail = func(w http.ResponseWriter, r *http.Request) {
			Expect(r.Method).To(Equal(http.MethodPut))
			Expect(r.Header.Get("Content-Disposition")).To(ContainSubstring(`filename="shot.nef"`))
			Expect(r.Header.Get("Content-Type")).To(Equal("image/x-nikon-nef"), "the file's type travels as the detection hint")
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(presetZip(map[string][]byte{"1.png": pngBytes()}))
		}

		img, err := TikaThumbnail{tikaURL: server.URL, filename: "shot.nef", contentType: "image/x-nikon-nef"}.Convert(bytes.NewReader([]byte("raw")))
		Expect(err).ToNot(HaveOccurred())
		Expect(img).ToNot(BeNil())
		Expect(requests).To(HaveLen(1))
	})

	It("decodes an image whose entry name has no telling extension", func() {
		thumbnail = func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(presetZip(map[string][]byte{"thumbnail": pngBytes()}))
		}

		img, err := TikaThumbnail{tikaURL: server.URL}.Convert(bytes.NewReader([]byte("raw")))
		Expect(err).ToNot(HaveOccurred())
		Expect(img).ToNot(BeNil(), "the bytes name the type when the entry does not")
	})

	It("reports a document without a thumbnail", func() {
		thumbnail = func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }

		_, err := TikaThumbnail{tikaURL: server.URL}.Convert(bytes.NewReader([]byte("zip")))
		Expect(err).To(MatchError(ErrNoThumbnail))
	})

	It("reports an empty zip as no thumbnail", func() {
		thumbnail = func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(presetZip(nil))
		}

		_, err := TikaThumbnail{tikaURL: server.URL}.Convert(bytes.NewReader([]byte("zip")))
		Expect(err).To(MatchError(ErrNoThumbnail))
	})

	It("reports a failing server", func() {
		thumbnail = func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }

		_, err := TikaThumbnail{tikaURL: server.URL}.Convert(bytes.NewReader([]byte("raw")))
		Expect(err).To(MatchError(ContainSubstring("500")))
	})

	It("points at the preset config when the route is missing", func() {
		_, err := TikaThumbnail{tikaURL: server.URL + "/nowhere"}.Convert(bytes.NewReader([]byte("raw")))
		Expect(err).To(MatchError(ContainSubstring("404")))
		Expect(err).To(MatchError(ContainSubstring("preset")), "a 404 means the catalog preset is not active")
	})
})

var _ = Describe("ForType with a Tika server", func() {
	It("routes everything Tika is asked for to Tika, text and gif stay native", func() {
		opts := map[string]any{"tika": thumbnail.NewTika("http://tika:9998", nil), "filename": "song.mp3"}
		Expect(ForType("audio/mpeg", opts)).To(BeAssignableToTypeOf(TikaThumbnail{}))
		Expect(ForType("image/x-nikon-nef", opts)).To(BeAssignableToTypeOf(TikaThumbnail{}))
		Expect(ForType("application/vnd.geogebra.slides", opts)).To(BeAssignableToTypeOf(TikaThumbnail{}))
		Expect(ForType("application/pdf", opts)).To(BeAssignableToTypeOf(TikaThumbnail{}))
		Expect(ForType("text/plain", opts)).To(BeAssignableToTypeOf(TxtToImageConverter{}))
		Expect(ForType("image/gif", opts)).To(BeAssignableToTypeOf(GifDecoder{}))
	})

	It("keeps the in-process converters without a Tika server", func() {
		Expect(ForType("audio/mpeg", nil)).To(BeAssignableToTypeOf(AudioDecoder{}))
		Expect(ForType("application/vnd.geogebra.slides", nil)).To(BeAssignableToTypeOf(GgsDecoder{}))
		Expect(ForType("image/x-nikon-nef", nil)).To(BeAssignableToTypeOf(ImageDecoder{}))
	})

	It("follows a configured list and hands the mapped type to Tika", func() {
		opts := map[string]any{"tika": thumbnail.NewTika("http://tika:9998", []string{"application/pdf", "image/x-raw-samsung:image/x-samsung-srw"})}
		Expect(ForType("application/pdf", opts)).To(BeAssignableToTypeOf(TikaThumbnail{}))
		Expect(ForType("audio/mpeg", opts)).To(BeAssignableToTypeOf(AudioDecoder{}), "not in the configured list")
		Expect(ForType("image/x-raw-samsung", opts).(TikaThumbnail).contentType).To(Equal("image/x-samsung-srw"))
	})
})
