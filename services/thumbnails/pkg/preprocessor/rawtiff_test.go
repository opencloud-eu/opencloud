package preprocessor

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"time"

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

var _ = Describe("RawTiffDecoder", func() {
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

	Describe("candidate selection", func() {
		It("accepts a single-strip JPEG (the CR2 IFD0 layout)", func() {
			le := binary.LittleEndian
			jpg := encodeJPEG(32, 16)
			buf := []byte{'I', 'I', 42, 0, 8, 0, 0, 0}
			ifdStart := uint32(len(buf))
			dataStart := ifdStart + 2 + 2*12 + 4
			entry := func(tag, typ uint16, value uint32) []byte {
				e := le.AppendUint16(nil, tag)
				e = le.AppendUint16(e, typ)
				e = le.AppendUint32(e, 1)
				return le.AppendUint32(e, value)
			}
			ifd := le.AppendUint16(nil, 2)
			ifd = append(ifd, entry(tiffTagStripOffsets, tiffTypeLong, dataStart)...)
			ifd = append(ifd, entry(tiffTagStripByteCounts, tiffTypeLong, uint32(len(jpg)))...)
			ifd = le.AppendUint32(ifd, 0)
			buf = append(buf, ifd...)
			buf = append(buf, jpg...)

			got, _, err := extractEmbeddedJPEG(buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(jpg))
		})

		It("rejects lossless JPEG payloads (the DNG raw layout)", func() {
			// SOI + SOF3: starts like a JPEG but no common decoder renders it
			lossless := append([]byte{0xff, 0xd8, 0xff, 0xc3, 0x00, 0x0b, 8, 0, 16, 0, 16, 1, 0, 0x11, 0}, thumb...)
			_, _, err := extractEmbeddedJPEG(buildRawFile(1, nil, lossless))
			Expect(err).To(MatchError(thumbnailerErrors.ErrNoImageFromRawFile))
		})
	})

	Describe("Convert", func() {
		It("decodes the embedded preview", func() {
			img, err := RawTiffDecoder{}.Convert(bytes.NewReader(buildRawFile(1, thumb, preview)))
			Expect(err).ToNot(HaveOccurred())
			bounds := img.(image.Image).Bounds()
			Expect(bounds.Dx()).To(Equal(64))
			Expect(bounds.Dy()).To(Equal(32))
		})

		It("applies the container orientation to the preview", func() {
			// orientation 6 = 90 degrees clockwise: dimensions swap
			img, err := RawTiffDecoder{}.Convert(bytes.NewReader(buildRawFile(6, thumb, preview)))
			Expect(err).ToNot(HaveOccurred())
			bounds := img.(image.Image).Bounds()
			Expect(bounds.Dx()).To(Equal(32))
			Expect(bounds.Dy()).To(Equal(64))
		})
	})
	Describe("hardening against crafted files", func() {
		It("stays bounded when an IFD is packed with huge SubIFD counts", func() {
			le := binary.LittleEndian
			preview := encodeJPEG(64, 32)
			entry := func(tag, typ uint16, count, value uint32) []byte {
				e := le.AppendUint16(nil, tag)
				e = le.AppendUint16(e, typ)
				e = le.AppendUint32(e, count)
				return le.AppendUint32(e, value)
			}
			const subEntries = 200
			buf := []byte{'I', 'I', 42, 0, 8, 0, 0, 0}
			ifdStart := uint32(len(buf))
			entryCount := uint16(2 + subEntries)
			previewStart := ifdStart + 2 + uint32(entryCount)*12 + 4

			ifd := le.AppendUint16(nil, entryCount)
			ifd = append(ifd, entry(tiffTagJPEGOffset, tiffTypeLong, 1, previewStart)...)
			ifd = append(ifd, entry(tiffTagJPEGLength, tiffTypeLong, 1, uint32(len(preview)))...)
			for i := 0; i < subEntries; i++ {
				// each claims 4 billion SubIFD pointers at a bogus array offset
				ifd = append(ifd, entry(tiffTagSubIFDs, tiffTypeLong, 0xffffffff, 8)...)
			}
			ifd = le.AppendUint32(ifd, 0)
			buf = append(buf, ifd...)
			buf = append(buf, preview...)

			done := make(chan []byte, 1)
			go func() {
				jpg, _, _ := extractEmbeddedJPEG(buf)
				done <- jpg
			}()
			select {
			case jpg := <-done:
				Expect(jpg).To(Equal(preview))
			case <-time.After(5 * time.Second):
				Fail("extractEmbeddedJPEG did not return within 5s on a crafted file")
			}
		})
	})

	Describe("isRenderableJPEG", func() {
		It("reaches a SOF that lies past 64KB of leading segments", func() {
			buf := []byte{0xff, 0xd8}
			// two ~40KB APP1 segments push the SOF past the old 64KB window
			for n := 0; n < 2; n++ {
				const payload = 40000
				buf = append(buf, 0xff, 0xe1, byte((payload+2)>>8), byte((payload+2)&0xff))
				buf = append(buf, make([]byte, payload)...)
			}
			buf = append(buf, 0xff, 0xc0, 0x00, 0x0b, 0x08, 0, 16, 0, 16, 0x01, 0x01, 0x11, 0x00)
			Expect(isRenderableJPEG(buf)).To(BeTrue())
		})

		It("rejects a stream that hides the SOF behind too many segments", func() {
			buf := []byte{0xff, 0xd8}
			for n := 0; n < maxJPEGSegments+5; n++ {
				buf = append(buf, 0xff, 0xe1, 0x00, 0x04, 0x00, 0x00) // tiny APP1
			}
			buf = append(buf, 0xff, 0xc0, 0x00, 0x0b, 0x08, 0, 16, 0, 16, 0x01, 0x01, 0x11, 0x00)
			Expect(isRenderableJPEG(buf)).To(BeFalse())
		})
	})

})

// buildRawFileBE mirrors buildRawFile in big-endian ('MM') byte order, which is
// how real Nikon NEFs are laid out. The inline orientation SHORT must be
// left-justified in the 4-byte value field, so it is shifted into the high
// bytes; LONG offset/length entries fill the field and need no adjustment.
func buildRawFileBE(orientation uint16, thumb, preview []byte) []byte {
	be := binary.BigEndian
	buf := []byte{'M', 'M', 0, 42, 0, 0, 0, 8}

	entryLong := func(tag uint16, value uint32) []byte {
		e := be.AppendUint16(nil, tag)
		e = be.AppendUint16(e, tiffTypeLong)
		e = be.AppendUint32(e, 1)
		return be.AppendUint32(e, value)
	}
	entryShort := func(tag uint16, value uint16) []byte {
		e := be.AppendUint16(nil, tag)
		e = be.AppendUint16(e, tiffTypeShort)
		e = be.AppendUint32(e, 1)
		return be.AppendUint32(e, uint32(value)<<16) // left-justified
	}

	ifd0Start := uint32(len(buf))
	subifdStart := ifd0Start + 2 + 4*12 + 4
	thumbStart := subifdStart + 2 + 2*12 + 4
	previewStart := thumbStart + uint32(len(thumb))

	ifd0 := be.AppendUint16(nil, 4)
	ifd0 = append(ifd0, entryShort(tiffTagOrientation, orientation)...)
	ifd0 = append(ifd0, entryLong(tiffTagSubIFDs, subifdStart)...)
	ifd0 = append(ifd0, entryLong(tiffTagJPEGOffset, thumbStart)...)
	ifd0 = append(ifd0, entryLong(tiffTagJPEGLength, uint32(len(thumb)))...)
	ifd0 = be.AppendUint32(ifd0, 0)

	subifd := be.AppendUint16(nil, 2)
	subifd = append(subifd, entryLong(tiffTagJPEGOffset, previewStart)...)
	subifd = append(subifd, entryLong(tiffTagJPEGLength, uint32(len(preview)))...)
	subifd = be.AppendUint32(subifd, 0)

	buf = append(buf, ifd0...)
	buf = append(buf, subifd...)
	buf = append(buf, thumb...)
	return append(buf, preview...)
}

var _ = Describe("RawTiffDecoder byte order and selection", func() {
	It("parses big-endian (MM) files like real NEFs", func() {
		thumb := encodeJPEG(16, 8)
		preview := encodeJPEG(64, 32)
		jpg, orientation, err := extractEmbeddedJPEG(buildRawFileBE(6, thumb, preview))
		Expect(err).ToNot(HaveOccurred())
		Expect(jpg).To(Equal(preview))
		Expect(orientation).To(Equal(uint16(6)))
	})

	It("applies the container orientation on a big-endian file", func() {
		img, err := RawTiffDecoder{}.Convert(bytes.NewReader(buildRawFileBE(6, encodeJPEG(16, 8), encodeJPEG(64, 32))))
		Expect(err).ToNot(HaveOccurred())
		bounds := img.(image.Image).Bounds()
		Expect(bounds.Dx()).To(Equal(32))
		Expect(bounds.Dy()).To(Equal(64))
	})

	It("selects the largest candidate regardless of discovery order", func() {
		// the larger JPEG sits in IFD0 (appended first), the smaller in the
		// SubIFD (appended later): a last-wins bug would pick the small one
		large := encodeJPEG(64, 32)
		small := encodeJPEG(16, 8)
		jpg, _, err := extractEmbeddedJPEG(buildRawFile(1, large, small))
		Expect(err).ToNot(HaveOccurred())
		Expect(jpg).To(Equal(large))
	})

	It("rejects a preview whose length exceeds maxPreviewLength", func() {
		preview := encodeJPEG(64, 32)
		original := maxPreviewLength
		defer func() { maxPreviewLength = original }()
		maxPreviewLength = int64(len(preview)) - 1
		_, _, err := extractEmbeddedJPEG(buildRawFile(1, encodeJPEG(8, 8), preview))
		// only the tiny thumbnail remains under the cap
		Expect(err).ToNot(HaveOccurred())
		maxPreviewLength = 1
		_, _, err = extractEmbeddedJPEG(buildRawFile(1, preview, preview))
		Expect(err).To(MatchError(thumbnailerErrors.ErrNoImageFromRawFile))
	})
})

var _ = Describe("isRenderableJPEG classification", func() {
	sof := func(marker byte) []byte {
		return []byte{0xff, 0xd8, 0xff, marker, 0x00, 0x0b, 0x08, 0, 16, 0, 16, 0x01, 0x01, 0x11, 0x00}
	}
	cases := []struct {
		name string
		buf  []byte
		want bool
	}{
		{"baseline SOF0", sof(0xc0), true},
		{"extended sequential SOF1", sof(0xc1), true},
		{"progressive SOF2", sof(0xc2), true},
		{"lossless SOF3", sof(0xc3), false},
		{"arithmetic SOF9", sof(0xc9), false},
		{"not a JPEG", []byte{0x00, 0x01, 0x02, 0x03}, false},
		{"SOI only, truncated", []byte{0xff, 0xd8}, false},
		{"SOS before any SOF", []byte{0xff, 0xd8, 0xff, 0xda, 0x00, 0x02}, false},
		{"skips a fill byte before the SOF", append([]byte{0xff, 0xd8, 0xff}, sof(0xc0)[2:]...), true},
		{"skips an APP1 segment before the SOF", append([]byte{0xff, 0xd8, 0xff, 0xe1, 0x00, 0x04, 0x00, 0x00}, sof(0xc0)[2:]...), true},
	}
	for _, tc := range cases {
		It(tc.name, func() {
			Expect(isRenderableJPEG(tc.buf)).To(Equal(tc.want))
		})
	}
})

var _ = Describe("RawTiffDecoder orientation direction", func() {
	// encodeCornerMarked returns a JPEG with the top-left quadrant bright red on
	// black, so the rotation direction can be read from where the red lands.
	encodeCornerMarked := func(w, h int) []byte {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h/2; y++ {
			for x := 0; x < w/2; x++ {
				img.Set(x, y, color.RGBA{R: 255, A: 255})
			}
		}
		var buf bytes.Buffer
		Expect(jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95})).To(Succeed())
		return buf.Bytes()
	}
	reddestCorner := func(img image.Image) string {
		b := img.Bounds()
		w, h := b.Dx(), b.Dy()
		avg := func(x0, y0 int) int {
			sum := 0
			for y := y0; y < y0+h/4; y++ {
				for x := x0; x < x0+w/4; x++ {
					r, _, _, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
					sum += int(r >> 8)
				}
			}
			return sum
		}
		corners := map[string]int{"TL": avg(0, 0), "TR": avg(3*w/4, 0), "BL": avg(0, 3*h/4), "BR": avg(3*w/4, 3*h/4)}
		best, bestV := "", -1
		for k, v := range corners {
			if v > bestV {
				best, bestV = k, v
			}
		}
		return best
	}
	cases := []struct {
		name        string
		orientation uint16
		wantCorner  string
	}{
		{"orientation 1 keeps it top-left", 1, "TL"},
		{"orientation 6 rotates it to top-right", 6, "TR"},
		{"orientation 8 rotates it to bottom-left", 8, "BL"},
	}
	for _, tc := range cases {
		It(tc.name, func() {
			marked := encodeCornerMarked(48, 24)
			img, err := RawTiffDecoder{}.Convert(bytes.NewReader(buildRawFile(tc.orientation, encodeJPEG(8, 8), marked)))
			Expect(err).ToNot(HaveOccurred())
			Expect(reddestCorner(img.(image.Image))).To(Equal(tc.wantCorner))
		})
	}
})
