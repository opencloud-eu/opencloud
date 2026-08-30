package preprocessor

import (
	"bytes"
	"encoding/base64"
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
			if r.URL.Path != "/unpack/thumbnail" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			thumbnail(w, r)
		}))
	})

	AfterEach(func() {
		server.Close()
	})

	It("takes the thumbnail Tika picks, with its metadata, from one request", func() {
		thumbnail = func(w http.ResponseWriter, r *http.Request) {
			Expect(r.Method).To(Equal(http.MethodPut))
			Expect(r.URL.Query().Get("renderThumbnails")).To(Equal("true"))
			Expect(r.Header.Get("Content-Disposition")).To(ContainSubstring(`filename="shot.nef"`))
			Expect(r.Header.Get("Content-Type")).To(Equal("image/x-nikon-nef"), "the file's type travels as the detection hint")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"metadata": map[string]any{"Content-Type": "image/png", "tk:embedded-resource-type": "THUMBNAIL"},
				"image":    base64.StdEncoding.EncodeToString(pngBytes()),
			})
		}

		img, err := TikaThumbnail{tikaURL: server.URL, filename: "shot.nef", contentType: "image/x-nikon-nef"}.Convert(bytes.NewReader([]byte("raw")))
		Expect(err).ToNot(HaveOccurred())
		Expect(img).ToNot(BeNil())
		Expect(requests).To(HaveLen(1))
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

	It("reports a Tika without the endpoint", func() {
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
