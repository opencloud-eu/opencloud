package preprocessor

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"io"
	"os"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPreprocessor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Preprocessor Suite")
}

const ggsThumbnailPath = "_slide0/geogebra_thumbnail.png"

// ggsWithEntry deflates size bytes of zeros into a ggs container. lieAboutSize
// patches the declared sizes down to one byte, the way a crafted file does.
func ggsWithEntry(size int, lieAboutSize bool) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	entry, err := w.Create(ggsThumbnailPath)
	Expect(err).ToNot(HaveOccurred())
	_, err = entry.Write(make([]byte, size))
	Expect(err).ToNot(HaveOccurred())
	Expect(w.Close()).To(Succeed())

	out := buf.Bytes()
	if !lieAboutSize {
		return out
	}

	// uncompressed size: 4 bytes at offset 22 of the local header, at offset 24
	// of the central directory entry
	patch := func(signature []byte, offset int) {
		at := bytes.Index(out, signature)
		Expect(at).ToNot(Equal(-1))
		binary.LittleEndian.PutUint32(out[at+offset:], 1)
	}
	patch([]byte("PK\x03\x04"), 22)
	patch([]byte("PK\x01\x02"), 24)

	return out
}

var _ = Describe("preprocessor", func() {
	Describe("GgsDecoder", func() {
		var fileReader io.Reader
		BeforeEach(func() {
			fileContent, err := os.ReadFile("test_assets/ggs_test.ggs")
			if err != nil {
				panic(err)
			}
			fileReader = bytes.NewReader(fileContent)
		})

		It("hands on the embedded thumbnail as the png it is", func() {
			decoder := GgsDecoder{thumbnailpath: ggsThumbnailPath, maxSize: defaultMaxExtractedSize}
			img, err := decoder.Convert(fileReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(img.ContentType).To(Equal("image/png"))
			Expect(img.Data).ToNot(BeEmpty())
		})

		It("should return an error if the ggs is invalid", func() {
			decoder := GgsDecoder{thumbnailpath: ggsThumbnailPath, maxSize: defaultMaxExtractedSize}
			img, err := decoder.Convert(bytes.NewReader([]byte("not a ggs")))
			Expect(err).To(HaveOccurred())
			Expect(img).To(BeNil())
		})

		It("reports a ggs without a thumbnail as having none", func() {
			var buf bytes.Buffer
			w := zip.NewWriter(&buf)
			_, err := w.Create("geogebra.xml")
			Expect(err).ToNot(HaveOccurred())
			Expect(w.Close()).To(Succeed())

			decoder := GgsDecoder{thumbnailpath: ggsThumbnailPath, maxSize: defaultMaxExtractedSize}
			img, err := decoder.Convert(bytes.NewReader(buf.Bytes()))
			Expect(err).To(MatchError(ErrNoThumbnail))
			Expect(img).To(BeNil())
		})

		It("refuses an entry that expands past the limit", func() {
			decoder := GgsDecoder{thumbnailpath: ggsThumbnailPath, maxSize: 1 << 20}

			img, err := decoder.Convert(bytes.NewReader(ggsWithEntry(8<<20, false)))
			Expect(err).To(MatchError(ContainSubstring("allowed")))
			Expect(img).To(BeNil())
		})

		It("refuses an entry whose declared size lies about the expansion", func() {
			decoder := GgsDecoder{thumbnailpath: ggsThumbnailPath, maxSize: 1 << 20}

			// archive/zip refuses a patched header while reading, so the read stays bounded
			img, err := decoder.Convert(bytes.NewReader(ggsWithEntry(8<<20, true)))
			Expect(err).To(HaveOccurred())
			Expect(img).To(BeNil())
		})
	})

	Describe("should decode audio", func() {
		var fileReader io.Reader
		It("hands on the cover art in the type the tag declares", func() {
			fileContent, err := os.ReadFile("test_assets/empty.mp3")
			if err != nil {
				panic(err)
			}
			fileReader = bytes.NewReader(fileContent)
			decoder := AudioDecoder{}
			img, err := decoder.Convert(fileReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(img.ContentType).To(HavePrefix("image/"))
			Expect(img.Data).ToNot(BeEmpty())
		})
		It("reports an audio without cover art as having none", func() {
			fileContent, err := os.ReadFile("test_assets/empty_no_image.mp3")
			if err != nil {
				panic(err)
			}
			fileReader = bytes.NewReader(fileContent)
			decoder := AudioDecoder{}
			img, err := decoder.Convert(fileReader)
			Expect(err).To(MatchError(ErrNoThumbnail))
			Expect(img).To(BeNil())
		})
		It("should return an error if the audio is invalid", func() {
			decoder := AudioDecoder{}
			img, err := decoder.Convert(bytes.NewReader([]byte("not an audio")))
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrNoThumbnail))
			Expect(img).To(BeNil())
		})
	})

	Describe("GgpDecoder", func() {
		It("hands on the embedded thumbnail", func() {
			data, err := os.ReadFile("test_assets/noise.png")
			Expect(err).ToNot(HaveOccurred())
			ggp := `{"sections":[{"cards":[{"element":{"image":{"base64Image":"data:image/png;base64,` + base64.StdEncoding.EncodeToString(data) + `"}}}]}]}`

			img, err := GgpDecoder{}.Convert(strings.NewReader(ggp))
			Expect(err).ToNot(HaveOccurred())
			Expect(img.ContentType).To(Equal("image/png"))
			Expect(img.Data).To(Equal(data))
		})

		DescribeTable("reports a ggp without a thumbnail as having none",
			func(ggp string) {
				img, err := GgpDecoder{}.Convert(strings.NewReader(ggp))
				Expect(err).To(MatchError(ErrNoThumbnail))
				Expect(img).To(BeNil())
			},
			Entry("no sections", `{}`),
			Entry("no cards", `{"sections":[{"cards":[]}]}`),
			Entry("no base64 image", `{"sections":[{"cards":[{"element":{"image":{"base64Image":""}}}]}]}`),
			Entry("not an image", `{"sections":[{"cards":[{"element":{"image":{"base64Image":"data:text/plain;base64,aGVsbG8="}}}]}]}`),
		)

		It("should return an error if the ggp is invalid", func() {
			img, err := GgpDecoder{}.Convert(strings.NewReader("not a ggp"))
			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(ErrNoThumbnail))
			Expect(img).To(BeNil())
		})
	})

	Describe("should decode text", func() {
		var decoder TxtToImageConverter
		BeforeEach(func() {
			fontFaceOpts := &opentype.FaceOptions{
				Size:    12,
				DPI:     72,
				Hinting: font.HintingNone,
			}

			fontLoader, err := NewFontLoader("", fontFaceOpts)
			if err != nil {
				fontLoader, _ = NewFontLoader("", fontFaceOpts)
			}
			decoder = TxtToImageConverter{
				fontLoader: fontLoader,
			}
		})
		It("renders the text as a png", func() {
			img, err := decoder.Convert(bytes.NewReader([]byte("This is a test text")))
			Expect(err).ToNot(HaveOccurred())
			Expect(img.ContentType).To(Equal("image/png"))
			Expect(img.Data).ToNot(BeEmpty())
		})

		It("fails if the font is missing", func() {
			decoder.fontLoader.fontMapData = &FontMapData{
				FMap: &FontMap{
					DefaultFont: "/some/unknown/font.otf",
				},
			}
			img, err := decoder.Convert(bytes.NewReader([]byte("This is a test text")))
			Expect(err).To(HaveOccurred())
			Expect(img).To(BeNil())
		})
	})

	Describe("ForType", func() {
		It("hands an image to the generator as it is", func() {
			Expect(ForType("image/png", Options{})).To(BeNil())
		})

		It("returns a GgsDecoder for a ggs", func() {
			Expect(ForType("application/vnd.geogebra.slides", Options{})).To(BeAssignableToTypeOf(GgsDecoder{}))
		})

		It("returns an AudioDecoder for audio", func() {
			Expect(ForType("audio/mpeg", Options{})).To(BeAssignableToTypeOf(AudioDecoder{}))
		})

		It("returns a TxtToImageConverter for text", func() {
			Expect(ForType("text/plain", Options{})).To(BeAssignableToTypeOf(TxtToImageConverter{}))
		})

		It("hands an announced type without a converter to the generator, which says whether it reads it", func() {
			Expect(ForType("application/pdf", Options{})).To(BeNil())
		})
	})
})
