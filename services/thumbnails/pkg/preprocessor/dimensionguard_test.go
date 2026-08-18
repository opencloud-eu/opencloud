//go:build !enable_vips

package preprocessor

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	thumbnailerErrors "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
)

// craftDimensionBomb encodes a tiny valid grayscale JPEG, then overwrites the
// SOF0 width/height so the header declares huge dimensions while the payload
// stays tiny.
func craftDimensionBomb(width, height uint16) []byte {
	var buf bytes.Buffer
	Expect(jpeg.Encode(&buf, image.NewGray(image.Rect(0, 0, 8, 8)), &jpeg.Options{Quality: 10})).To(Succeed())
	b := buf.Bytes()
	for i := 0; i+9 < len(b); i++ {
		if b[i] == 0xff && b[i+1] == 0xc0 {
			b[i+5], b[i+6] = byte(height>>8), byte(height)
			b[i+7], b[i+8] = byte(width>>8), byte(width)
			return b
		}
	}
	Fail("no SOF0 marker in encoded jpeg")
	return nil
}

var _ = Describe("ImageDecoder dimension guard", func() {
	It("rejects a source whose declared dimensions exceed the limit before decoding", func() {
		dec := ImageDecoder{limit: decodeLimit{maxWidth: 7680, maxHeight: 7680}}
		_, err := dec.Convert(bytes.NewReader(craftDimensionBomb(20000, 20000)))
		Expect(err).To(MatchError(thumbnailerErrors.ErrImageTooLarge))
	})

	It("decodes an image within the limit", func() {
		var buf bytes.Buffer
		Expect(jpeg.Encode(&buf, image.NewGray(image.Rect(0, 0, 800, 600)), nil)).To(Succeed())
		dec := ImageDecoder{limit: decodeLimit{maxWidth: 7680, maxHeight: 7680}}
		img, err := dec.Convert(bytes.NewReader(buf.Bytes()))
		Expect(err).ToNot(HaveOccurred())
		Expect(img).ToNot(BeNil())
	})

	It("applies no limit when the bounds are zero", func() {
		dec := ImageDecoder{}
		var buf bytes.Buffer
		Expect(jpeg.Encode(&buf, image.NewGray(image.Rect(0, 0, 16, 16)), nil)).To(Succeed())
		_, err := dec.Convert(bytes.NewReader(buf.Bytes()))
		Expect(err).ToNot(HaveOccurred())
	})

	It("rejects an oversized gif before decoding", func() {
		dec := GifDecoder{limit: decodeLimit{maxWidth: 7680, maxHeight: 7680}}
		// LSD declares a 20000x20000 logical screen
		g := []byte("GIF89a")
		g = append(g, 0x20, 0x4e, 0x20, 0x4e, 0xf0, 0x00, 0x00) // 20000x20000, gct flag
		g = append(g, bytes.Repeat([]byte{0}, 6)...)            // minimal gct + terminator-ish
		_, err := dec.Convert(bytes.NewReader(g))
		Expect(err).To(MatchError(thumbnailerErrors.ErrImageTooLarge))
	})
})

var _ = Describe("dimension limit propagation", func() {
	limit := decodeLimit{maxWidth: 7680, maxHeight: 7680}
	opts := map[string]any{"maxInputWidth": 7680, "maxInputHeight": 7680}

	It("threads the limit into decoders that recurse into ForType", func() {
		Expect(ForType("audio/mpeg", opts)).To(Equal(AudioDecoder{limit: limit}))
		Expect(ForType("application/vnd.geogebra.pinboard", opts)).To(Equal(GgpDecoder{limit: limit}))
		g, ok := ForType("application/vnd.geogebra.slides", opts).(GgsDecoder)
		Expect(ok).To(BeTrue())
		Expect(g.limit).To(Equal(limit))
	})

	It("rejects an oversized cover image embedded in a ggp file", func() {
		bomb := craftDimensionBomb(20000, 20000)
		payload := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(bomb)
		ggp := GGPStruct{}
		ggp.Sections = append(ggp.Sections, struct {
			Cards []struct {
				Element struct {
					Image struct{ Base64Image string }
				}
			}
		}{Cards: []struct {
			Element struct {
				Image struct{ Base64Image string }
			}
		}{{Element: struct {
			Image struct{ Base64Image string }
		}{Image: struct{ Base64Image string }{Base64Image: payload}}}}})
		raw, err := json.Marshal(ggp)
		Expect(err).ToNot(HaveOccurred())
		_, err = GgpDecoder{limit: limit}.Convert(bytes.NewReader(raw))
		Expect(err).To(MatchError(thumbnailerErrors.ErrImageTooLarge))
	})
})
