package preprocessor

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
)

// unpackZip builds what /unpack/all returns: the entries and their metadata
// side by side, "1.png" next to "1.png.metadata.json".
func unpackZip(entries map[string][]byte, meta map[string]map[string]any) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range entries {
		w, err := zw.Create(name)
		Expect(err).ToNot(HaveOccurred())
		_, err = w.Write(data)
		Expect(err).ToNot(HaveOccurred())

		w, err = zw.Create(name + ".metadata.json")
		Expect(err).ToNot(HaveOccurred())
		Expect(json.NewEncoder(w).Encode(meta[name])).To(Succeed())
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
			if r.URL.Path != "/unpack/all" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			thumbnail(w, r)
		}))
	})

	AfterEach(func() {
		server.Close()
	})

	It("takes the entry Tika marked as the thumbnail, in one request", func() {
		thumbnail = func(w http.ResponseWriter, r *http.Request) {
			Expect(r.Method).To(Equal(http.MethodPut))
			Expect(r.Header.Get("Content-Disposition")).To(ContainSubstring(`filename="shot.nef"`))
			Expect(r.Header.Get("Content-Type")).To(Equal("image/x-nikon-nef"), "the file's type travels as the detection hint")
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(unpackZip(
				map[string][]byte{"0.nef": []byte("the raw itself"), "1.png": pngBytes()},
				map[string]map[string]any{
					"0.nef": {"Content-Type": "image/x-nikon-nef"},
					"1.png": {"Content-Type": "image/png", "tk:embedded-resource-type": "THUMBNAIL"},
				},
			))
		}

		img, err := TikaThumbnail{tikaURL: server.URL, filename: "shot.nef", contentType: "image/x-nikon-nef"}.Convert(bytes.NewReader([]byte("raw")))
		Expect(err).ToNot(HaveOccurred())
		Expect(img).ToNot(BeNil())
		Expect(requests).To(HaveLen(1))
	})

	It("ignores embedded images that are not the thumbnail", func() {
		thumbnail = func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(unpackZip(
				map[string][]byte{"1.png": pngBytes()},
				map[string]map[string]any{"1.png": {"Content-Type": "image/png", "tk:embedded-resource-type": "INLINE"}},
			))
		}

		_, err := TikaThumbnail{tikaURL: server.URL}.Convert(bytes.NewReader([]byte("raw")))
		Expect(err).To(MatchError(ErrNoThumbnail), "an inline image is not a thumbnail, tika 4.0 marks cover art that way")
	})

	It("takes the rendering when the thumbnail is a metafile", func() {
		thumbnail = func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(unpackZip(
				map[string][]byte{"1.wmf": []byte("vector"), "2.png": pngBytes()},
				map[string]map[string]any{
					"1.wmf": {"Content-Type": "image/wmf", "tk:embedded-resource-type": "THUMBNAIL", "tk:embedded-id-path": "/1"},
					"2.png": {"Content-Type": "image/png", "tk:embedded-resource-type": "RENDERING", "tk:embedded-id-path": "/1/2"},
				},
			))
		}

		img, err := TikaThumbnail{tikaURL: server.URL, filename: "report.doc"}.Convert(bytes.NewReader([]byte("ole2")))
		Expect(err).ToNot(HaveOccurred())
		Expect(img).ToNot(BeNil(), "office documents carry a metafile preview, the raster is its rendering")
	})

	It("reports no thumbnail when the metafile was not rendered", func() {
		thumbnail = func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(unpackZip(
				map[string][]byte{"1.wmf": []byte("vector")},
				map[string]map[string]any{
					"1.wmf": {"Content-Type": "image/wmf", "tk:embedded-resource-type": "THUMBNAIL", "tk:embedded-id-path": "/1"},
				},
			))
		}

		_, err := TikaThumbnail{tikaURL: server.URL}.Convert(bytes.NewReader([]byte("ole2")))
		Expect(err).To(MatchError(ErrNoThumbnail), "rendering is off in tika by default")
	})

	It("does not take a rendering that belongs to another image", func() {
		thumbnail = func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(unpackZip(
				map[string][]byte{"1.wmf": []byte("vector"), "3.png": pngBytes()},
				map[string]map[string]any{
					"1.wmf": {"Content-Type": "image/wmf", "tk:embedded-resource-type": "THUMBNAIL", "tk:embedded-id-path": "/1"},
					"3.png": {"Content-Type": "image/png", "tk:embedded-resource-type": "RENDERING", "tk:embedded-id-path": "/2/3"},
				},
			))
		}

		_, err := TikaThumbnail{tikaURL: server.URL}.Convert(bytes.NewReader([]byte("ole2")))
		Expect(err).To(MatchError(ErrNoThumbnail))
	})

	It("takes a rendering of the document itself", func() {
		thumbnail = func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(unpackZip(
				map[string][]byte{"1.png": pngBytes()},
				map[string]map[string]any{
					"1.png": {"Content-Type": "image/png", "tk:embedded-resource-type": "RENDERING", "tk:embedded-id-path": "/1"},
				},
			))
		}

		img, err := TikaThumbnail{tikaURL: server.URL, filename: "report.pdf"}.Convert(bytes.NewReader([]byte("pdf")))
		Expect(err).ToNot(HaveOccurred())
		Expect(img).ToNot(BeNil(), "a pdf has no preview of its own, a rendered page is one")
	})

	It("reports a document without a thumbnail", func() {
		thumbnail = func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }

		_, err := TikaThumbnail{tikaURL: server.URL}.Convert(bytes.NewReader([]byte("zip")))
		Expect(err).To(MatchError(ErrNoThumbnail))
	})

	It("reports a failing server", func() {
		thumbnail = func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }

		_, err := TikaThumbnail{tikaURL: server.URL}.Convert(bytes.NewReader([]byte("raw")))
		Expect(err).To(MatchError(ContainSubstring("500")))
	})

	It("reports a Tika that does not know the route", func() {
		thumbnail = func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }

		_, err := TikaThumbnail{tikaURL: server.URL}.Convert(bytes.NewReader([]byte("raw")))
		Expect(err).To(MatchError(ContainSubstring("404")))
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
