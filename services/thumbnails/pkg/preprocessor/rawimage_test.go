package preprocessor

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	thumbnailerErrors "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
)

// encodeJPEG returns a real JPEG of the given size
func encodeJPEG(width, height int) []byte {
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	Expect(jpeg.Encode(&buf, img, nil)).To(Succeed())
	return buf.Bytes()
}

// buildRawFile assembles a minimal little-endian TIFF: IFD0 carries the
// orientation, a thumbnail JPEG and a SubIFD pointer, the SubIFD carries the
// full-size preview JPEG (the NEF "JpgFromRaw" layout)
func buildRawFile(orientation uint16, thumb, preview []byte) []byte {
	le := binary.LittleEndian
	buf := []byte{'I', 'I', 42, 0, 8, 0, 0, 0}

	entry := func(tag, typ uint16, value uint32) []byte {
		e := le.AppendUint16(nil, tag)
		e = le.AppendUint16(e, typ)
		e = le.AppendUint32(e, 1)
		return le.AppendUint32(e, value)
	}

	// layout: header(8) ifd0(2+4*12+4) subifd(2+2*12+4) thumb preview
	ifd0Start := uint32(len(buf))
	subifdStart := ifd0Start + 2 + 4*12 + 4
	thumbStart := subifdStart + 2 + 2*12 + 4
	previewStart := thumbStart + uint32(len(thumb))

	ifd0 := le.AppendUint16(nil, 4)
	ifd0 = append(ifd0, entry(tiffTagOrientation, tiffTypeShort, uint32(orientation))...)
	ifd0 = append(ifd0, entry(tiffTagSubIFDs, tiffTypeLong, subifdStart)...)
	ifd0 = append(ifd0, entry(tiffTagJPEGOffset, tiffTypeLong, thumbStart)...)
	ifd0 = append(ifd0, entry(tiffTagJPEGLength, tiffTypeLong, uint32(len(thumb)))...)
	ifd0 = le.AppendUint32(ifd0, 0)

	subifd := le.AppendUint16(nil, 2)
	subifd = append(subifd, entry(tiffTagJPEGOffset, tiffTypeLong, previewStart)...)
	subifd = append(subifd, entry(tiffTagJPEGLength, tiffTypeLong, uint32(len(preview)))...)
	subifd = le.AppendUint32(subifd, 0)

	buf = append(buf, ifd0...)
	buf = append(buf, subifd...)
	buf = append(buf, thumb...)
	return append(buf, preview...)
}

var _ = Describe("RawImageDecoder", func() {
	var (
		thumb   = encodeJPEG(16, 8)
		preview = encodeJPEG(64, 32)
	)

	Describe("extractEmbeddedJPEG", func() {
		It("extracts the largest embedded JPEG", func() {
			jpg, _, err := extractEmbeddedJPEG(buildRawFile(1, thumb, preview))
			Expect(err).ToNot(HaveOccurred())
			Expect(jpg).To(Equal(preview))
		})

		It("returns the orientation of the primary image", func() {
			_, orientation, err := extractEmbeddedJPEG(buildRawFile(6, thumb, preview))
			Expect(err).ToNot(HaveOccurred())
			Expect(orientation).To(Equal(uint16(6)))
		})

		It("rejects files without a TIFF header", func() {
			_, _, err := extractEmbeddedJPEG([]byte("not a tiff at all"))
			Expect(err).To(MatchError(thumbnailerErrors.ErrNoImageFromRawFile))
		})

		It("rejects files without an embedded JPEG", func() {
			_, _, err := extractEmbeddedJPEG(buildRawFile(1, nil, nil))
			Expect(err).To(MatchError(thumbnailerErrors.ErrNoImageFromRawFile))
		})

		It("survives truncated files", func() {
			full := buildRawFile(1, thumb, preview)
			for cut := 0; cut < len(full); cut += 7 {
				_, _, _ = extractEmbeddedJPEG(full[:cut])
			}
		})

		It("survives cyclic IFD chains", func() {
			data := buildRawFile(1, thumb, preview)
			// point the first IFD offset at itself
			binary.LittleEndian.PutUint32(data[4:8], 8)
			ifdEnd := 8 + 2 + 4*12
			binary.LittleEndian.PutUint32(data[ifdEnd:ifdEnd+4], 8)
			_, _, err := extractEmbeddedJPEG(data)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Convert", func() {
		It("decodes the embedded preview", func() {
			img, err := RawImageDecoder{}.Convert(bytes.NewReader(buildRawFile(1, thumb, preview)))
			Expect(err).ToNot(HaveOccurred())
			bounds := img.(image.Image).Bounds()
			Expect(bounds.Dx()).To(Equal(64))
			Expect(bounds.Dy()).To(Equal(32))
		})

		It("applies the container orientation to the preview", func() {
			// orientation 6 = 90 degrees clockwise: dimensions swap
			img, err := RawImageDecoder{}.Convert(bytes.NewReader(buildRawFile(6, thumb, preview)))
			Expect(err).ToNot(HaveOccurred())
			bounds := img.(image.Image).Bounds()
			Expect(bounds.Dx()).To(Equal(32))
			Expect(bounds.Dy()).To(Equal(64))
		})
	})
})
