package preprocessor

import (
	"bytes"
	"io"
	"os"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPreprocessor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Preprocessor Suite")
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
			decoder := GgsDecoder{"_slide0/geogebra_thumbnail.png"}
			img, err := decoder.Convert(fileReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(img.ContentType).To(Equal("image/png"))
			Expect(img.Data).ToNot(BeEmpty())
		})

		It("should return an error if the ggs is invalid", func() {
			decoder := GgsDecoder{"_slide0/geogebra_thumbnail.png"}
			img, err := decoder.Convert(bytes.NewReader([]byte("not a ggs")))
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
		It("should decode an audio", func() {
			fileContent, err := os.ReadFile("test_assets/empty_no_image.mp3")
			if err != nil {
				panic(err)
			}
			fileReader = bytes.NewReader(fileContent)
			decoder := AudioDecoder{}
			img, err := decoder.Convert(fileReader)
			Expect(err).To(HaveOccurred())
			Expect(img).To(BeNil())
		})
		It("should return an error if the audio is invalid", func() {
			decoder := AudioDecoder{}
			img, err := decoder.Convert(bytes.NewReader([]byte("not an audio")))
			Expect(err).To(HaveOccurred())
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
		It("has no converter for an image, which never reaches the preprocessor", func() {
			Expect(ForType("image/png", nil)).To(BeNil())
		})

		It("returns a GgsDecoder for a ggs", func() {
			Expect(ForType("application/vnd.geogebra.slides", nil)).To(BeAssignableToTypeOf(GgsDecoder{}))
		})

		It("returns an AudioDecoder for audio", func() {
			Expect(ForType("audio/mpeg", nil)).To(BeAssignableToTypeOf(AudioDecoder{}))
		})

		It("returns a TxtToImageConverter for text", func() {
			Expect(ForType("text/plain", nil)).To(BeAssignableToTypeOf(TxtToImageConverter{}))
		})

		It("has no converter for an unknown type", func() {
			Expect(ForType("unknown", nil)).To(BeNil())
		})
	})
})
