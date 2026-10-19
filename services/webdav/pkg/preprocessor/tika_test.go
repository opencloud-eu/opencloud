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

// presetZip builds what /unpack/preset/thumbnail returns: the image the preset
// selected, and whatever bookkeeping the package carries beside it.
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

	It("takes the image out of the package the preset wraps it in", func() {
		thumbnail = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/zip")
			//the package names its own bookkeeping, and has renamed it before
			_, _ = w.Write(presetZip(map[string][]byte{
				"datapackage.json": []byte(`{"resources":[]}`),
				"rmeta.json":       []byte(`[{"Content-Type":"application/pdf"}]`),
				"unpacked/1.png":   pngBytes(),
			}))
		}

		img, err := TikaThumbnail{tikaURL: server.URL, filename: "document.pdf"}.Convert(bytes.NewReader([]byte("pdf")))
		Expect(err).ToNot(HaveOccurred())
		Expect(img).ToNot(BeNil())
		Expect(img.ContentType).To(Equal("image/png"))
		Expect(img.Data).To(Equal(pngBytes()))
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

	It("points at the preset activation when the route is missing", func() {
		_, err := TikaThumbnail{tikaURL: server.URL + "/nowhere"}.Convert(bytes.NewReader([]byte("raw")))
		Expect(err).To(MatchError(ContainSubstring("404")))
		Expect(err).To(MatchError(ContainSubstring("preset")), "a 404 means the catalog preset is not active")
	})
})

var _ = Describe("ForType with a Tika server", func() {
	It("routes everything Tika is asked for to Tika, text and gif stay native", func() {
		opts := Options{Tika: thumbnail.NewTika("http://tika:9998", nil), Filename: "song.mp3"}
		Expect(ForType("audio/mpeg", opts)).To(BeAssignableToTypeOf(TikaThumbnail{}))
		Expect(ForType("application/vnd.geogebra.slides", opts)).To(BeAssignableToTypeOf(TikaThumbnail{}))
		Expect(ForType("application/pdf", opts)).To(BeAssignableToTypeOf(TikaThumbnail{}))
		Expect(ForType("text/plain", opts)).To(BeAssignableToTypeOf(TxtToImageConverter{}))
		Expect(ForType("image/png", opts)).To(BeNil(), "the generator reads a png itself")
		Expect(ForType("image/x-nikon-nef", opts)).To(BeNil(), "an image type the catch-all does not take, the generator may read it")
	})

	It("keeps the in-process converters without a Tika server", func() {
		Expect(ForType("audio/mpeg", Options{})).To(BeAssignableToTypeOf(AudioDecoder{}))
		Expect(ForType("application/vnd.geogebra.slides", Options{})).To(BeAssignableToTypeOf(GgsDecoder{}))
		Expect(ForType("image/x-nikon-nef", Options{})).To(BeNil(), "an image reaches the generator undecoded")
	})

	It("follows a configured list and hands the mapped type to Tika", func() {
		opts := Options{Tika: thumbnail.NewTika("http://tika:9998", []string{"application/pdf", "image/x-nikon-nef:image/x-raw-nikon"})}
		Expect(ForType("application/pdf", opts)).To(BeAssignableToTypeOf(TikaThumbnail{}))
		Expect(ForType("audio/mpeg", opts)).To(BeAssignableToTypeOf(AudioDecoder{}), "not in the configured list")
		Expect(ForType("image/x-nikon-nef", opts).(TikaThumbnail).contentType).To(Equal("image/x-raw-nikon"), "a named image type does go to Tika")
	})
})
