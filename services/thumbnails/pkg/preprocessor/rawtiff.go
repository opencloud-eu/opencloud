package preprocessor

import (
	"bytes"
	"encoding/binary"
	"io"

	thumbnailerErrors "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
)

// RawTiffDecoder is a converter for TIFF-based camera raw files (NEF, CR2,
// PEF, ARW, SR2, DNG). Decoding the raw sensor data would need a full raw
// development engine; instead it serves the camera-generated JPEG preview
// embedded in the file, the same image cameras and photo tools show for raw
// thumbnails.
type RawTiffDecoder struct{}

// Convert extracts the largest renderable embedded JPEG preview and decodes it
func (RawTiffDecoder) Convert(r io.Reader) (any, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	preview, orientation, err := extractEmbeddedJPEG(data)
	if err != nil {
		return nil, err
	}
	// the embedded preview usually carries no EXIF of its own, the rotation
	// lives in the raw container: splice it in so the regular image decoding
	// orients the thumbnail like any plain JPEG
	if orientation > 1 {
		preview = spliceJPEGOrientation(preview, orientation)
	}
	converter := ForType("image/jpeg", nil)
	if converter == nil {
		return nil, thumbnailerErrors.ErrNoImageFromRawFile
	}
	return converter.Convert(bytes.NewReader(preview))
}

const (
	tiffTagStripOffsets    = 0x0111
	tiffTagOrientation     = 0x0112
	tiffTagStripByteCounts = 0x0117
	tiffTagSubIFDs         = 0x014a
	tiffTagJPEGOffset      = 0x0201 // JPEGInterchangeFormat
	tiffTagJPEGLength      = 0x0202 // JPEGInterchangeFormatLength
	tiffTypeShort          = 3
	tiffTypeLong           = 4
	// cycle and decompression-bomb guard for untrusted IFD chains
	maxIFDs = 64
	// metadata segments (APPn, DQT, DHT) rarely exceed a few KB before the
	// SOF marker appears
	sofScanLimit = 64 * 1024
)

// extractEmbeddedJPEG walks the TIFF IFD chain (including SubIFDs, where NEF
// stores its full-size "JpgFromRaw") and returns the largest renderable
// embedded JPEG plus the container's EXIF orientation. Candidates come from
// the JPEGInterchangeFormat pair and from single-strip images (CR2 keeps its
// full-size JPEG as an IFD0 strip); they qualify by their actual stream
// content, which keeps raw sensor payloads out (DNG stores those as lossless
// JPEG, which no common decoder renders).
func extractEmbeddedJPEG(data []byte) ([]byte, uint16, error) {
	if len(data) < 8 {
		return nil, 0, thumbnailerErrors.ErrNoImageFromRawFile
	}
	var order binary.ByteOrder
	switch {
	case data[0] == 'I' && data[1] == 'I':
		order = binary.LittleEndian
	case data[0] == 'M' && data[1] == 'M':
		order = binary.BigEndian
	default:
		return nil, 0, thumbnailerErrors.ErrNoImageFromRawFile
	}
	if order.Uint16(data[2:4]) != 42 {
		return nil, 0, thumbnailerErrors.ErrNoImageFromRawFile
	}

	type candidate struct{ offset, length uint32 }
	var candidates []candidate
	var orientation uint16

	queue := []uint32{order.Uint32(data[4:8])}
	seen := map[uint32]struct{}{}
	first := true
	for len(queue) > 0 && len(seen) < maxIFDs {
		ifdOffset := queue[0]
		queue = queue[1:]
		if _, ok := seen[ifdOffset]; ok {
			continue
		}
		seen[ifdOffset] = struct{}{}
		if ifdOffset == 0 || int(ifdOffset)+2 > len(data) {
			continue
		}
		entryCount := int(order.Uint16(data[ifdOffset : ifdOffset+2]))
		entriesEnd := int(ifdOffset) + 2 + entryCount*12
		if entriesEnd+4 > len(data) {
			continue
		}

		var jpegOffset, jpegLength, stripOffset, stripLength uint32
		for i := 0; i < entryCount; i++ {
			entry := data[int(ifdOffset)+2+i*12:]
			tag := order.Uint16(entry[0:2])
			typ := order.Uint16(entry[2:4])
			count := order.Uint32(entry[4:8])
			switch tag {
			case tiffTagOrientation:
				// only the primary image's orientation applies to the previews
				if first && typ == tiffTypeShort && count == 1 {
					orientation = order.Uint16(entry[8:10])
				}
			case tiffTagJPEGOffset:
				if typ == tiffTypeLong && count == 1 {
					jpegOffset = order.Uint32(entry[8:12])
				}
			case tiffTagJPEGLength:
				if typ == tiffTypeLong && count == 1 {
					jpegLength = order.Uint32(entry[8:12])
				}
			case tiffTagStripOffsets:
				// only single-strip images can be a contiguous JPEG stream
				if typ == tiffTypeLong && count == 1 {
					stripOffset = order.Uint32(entry[8:12])
				}
			case tiffTagStripByteCounts:
				if typ == tiffTypeLong && count == 1 {
					stripLength = order.Uint32(entry[8:12])
				}
			case tiffTagSubIFDs:
				if typ != tiffTypeLong {
					continue
				}
				if count == 1 {
					queue = append(queue, order.Uint32(entry[8:12]))
					continue
				}
				arrayOffset := order.Uint32(entry[8:12])
				for j := uint32(0); j < count && j < maxIFDs; j++ {
					pos := int(arrayOffset) + int(j)*4
					if pos+4 > len(data) {
						break
					}
					queue = append(queue, order.Uint32(data[pos:pos+4]))
				}
			}
		}
		if jpegOffset > 0 && jpegLength > 0 {
			candidates = append(candidates, candidate{jpegOffset, jpegLength})
		}
		if stripOffset > 0 && stripLength > 0 {
			candidates = append(candidates, candidate{stripOffset, stripLength})
		}
		queue = append(queue, order.Uint32(data[entriesEnd:entriesEnd+4]))
		first = false
	}

	var best []byte
	for _, c := range candidates {
		end := int64(c.offset) + int64(c.length)
		if end > int64(len(data)) || int(c.length) <= len(best) {
			continue
		}
		jpg := data[c.offset:end]
		if !isRenderableJPEG(jpg) {
			continue
		}
		best = jpg
	}
	if best == nil {
		return nil, 0, thumbnailerErrors.ErrNoImageFromRawFile
	}
	return best, orientation, nil
}

// isRenderableJPEG walks the JPEG segments until the SOF marker and accepts
// only the DCT processes common decoders implement. Raw sensor payloads in
// DNGs are lossless JPEG (SOF3) and start with the same SOI marker, so the
// SOI alone does not qualify a stream.
func isRenderableJPEG(buf []byte) bool {
	if len(buf) < 4 || buf[0] != 0xff || buf[1] != 0xd8 {
		return false
	}
	if len(buf) > sofScanLimit {
		buf = buf[:sofScanLimit]
	}
	i := 2
	for i+4 <= len(buf) {
		if buf[i] != 0xff {
			return false
		}
		marker := buf[i+1]
		switch {
		case marker == 0xff: // fill byte
			i++
			continue
		case marker >= 0xd0 && marker <= 0xd7: // RST, no length field
			i += 2
			continue
		}
		switch marker {
		case 0xc0, 0xc1, 0xc2: // baseline, extended sequential, progressive
			return true
		case 0xc3, 0xc5, 0xc6, 0xc7, 0xc9, 0xca, 0xcb, 0xcd, 0xce, 0xcf:
			// lossless, differential and arithmetic processes
			return false
		case 0xd9, 0xda: // EOI or scan start without a SOF
			return false
		}
		segLen := int(buf[i+2])<<8 | int(buf[i+3])
		if segLen < 2 {
			return false
		}
		i += 2 + segLen
	}
	return false
}

// spliceJPEGOrientation inserts a minimal EXIF APP1 segment carrying only the
// orientation tag right after the SOI marker
func spliceJPEGOrientation(jpg []byte, orientation uint16) []byte {
	if len(jpg) < 2 || jpg[0] != 0xff || jpg[1] != 0xd8 {
		return jpg
	}
	tiff := make([]byte, 0, 26)
	le := binary.LittleEndian
	tiff = append(tiff, 'I', 'I')
	tiff = le.AppendUint16(tiff, 42)
	tiff = le.AppendUint32(tiff, 8) // IFD0 offset
	tiff = le.AppendUint16(tiff, 1) // one entry
	tiff = le.AppendUint16(tiff, tiffTagOrientation)
	tiff = le.AppendUint16(tiff, tiffTypeShort)
	tiff = le.AppendUint32(tiff, 1)
	tiff = le.AppendUint16(tiff, orientation)
	tiff = le.AppendUint16(tiff, 0) // value padding
	tiff = le.AppendUint32(tiff, 0) // no next IFD

	payload := append([]byte("Exif\x00\x00"), tiff...)
	out := make([]byte, 0, len(jpg)+4+len(payload))
	out = append(out, jpg[:2]...)
	out = append(out, 0xff, 0xe1, byte((len(payload)+2)>>8), byte(len(payload)+2))
	out = append(out, payload...)
	return append(out, jpg[2:]...)
}
