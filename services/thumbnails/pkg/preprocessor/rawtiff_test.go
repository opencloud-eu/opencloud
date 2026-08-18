package preprocessor

import (
	"bytes"
	"encoding/binary"
	"image"
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
