package preprocessor

import (
	"bytes"
	"encoding/binary"
	"io"

	thumbnailerErrors "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
)

// RawImageDecoder is a converter for TIFF-based camera raw files (NEF & co).
// Decoding the raw sensor data would need a full raw development engine;
// instead it serves the camera-generated JPEG preview embedded in the file,
// the same image cameras and photo tools show for raw thumbnails.
type RawImageDecoder struct{}

// Convert extracts the largest embedded JPEG preview and decodes it
func (RawImageDecoder) Convert(r io.Reader) (any, error) {
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
	tiffTagOrientation = 0x0112
	tiffTagSubIFDs     = 0x014a
	tiffTagJPEGOffset  = 0x0201 // JPEGInterchangeFormat
	tiffTagJPEGLength  = 0x0202 // JPEGInterchangeFormatLength
	tiffTypeShort      = 3
	tiffTypeLong       = 4
	// cycle and decompression-bomb guard for untrusted IFD chains
	maxIFDs = 64
)

// extractEmbeddedJPEG walks the TIFF IFD chain (including SubIFDs, where
// NEF stores its full-size "JpgFromRaw") and returns the largest embedded
// JPEG plus the container's EXIF orientation.
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

		var jpegOffset, jpegLength uint32
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
		if len(jpg) < 3 || jpg[0] != 0xff || jpg[1] != 0xd8 {
			continue
		}
		best = jpg
	}
	if best == nil {
		return nil, 0, thumbnailerErrors.ErrNoImageFromRawFile
	}
	return best, orientation, nil
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
