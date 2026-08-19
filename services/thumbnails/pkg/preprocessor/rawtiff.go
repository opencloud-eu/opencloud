package preprocessor

import (
	"bytes"
	"encoding/binary"
	"io"

	thumbnailerErrors "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
)

// RawTiffDecoder serves the embedded JPEG preview of TIFF-based camera raw
// files (the formats routed to it in ForType) instead of decoding raw sensor
// data.
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
	// the rotation lives in the container, the preview carries no EXIF
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
	tiffTypeIFD            = 13 // 32-bit IFD pointer
	tiffTypeLong8          = 16 // BigTIFF 64-bit offset
	tiffTypeIFD8           = 18 // BigTIFF 64-bit IFD pointer
	tiffMagic              = 42
	bigTiffMagic           = 43
	// caps that bound work on untrusted input
	maxIFDs         = 64
	maxIFDEntries   = 4096
	maxJPEGSegments = 32
)

// isLongType reports whether a tag type carries a 32-bit (or, in BigTIFF, a
// 64-bit) integer offset we can follow.
func isLongType(typ uint16, bigTiff bool) bool {
	return typ == tiffTypeLong || (bigTiff && typ == tiffTypeLong8)
}

// isSubIFDType reports whether a SubIFDs tag carries a followable pointer.
// Besides LONG/LONG8 it accepts the dedicated IFD (13) and, in BigTIFF, IFD8
// (18) pointer types, which cameras use for SubIFDs; length fields keep the
// stricter isLongType so they never accept a pointer type.
func isSubIFDType(typ uint16, bigTiff bool) bool {
	return isLongType(typ, bigTiff) || typ == tiffTypeIFD || (bigTiff && typ == tiffTypeIFD8)
}

// is8ByteType reports whether a value of this type occupies 8 bytes.
func is8ByteType(typ uint16) bool {
	return typ == tiffTypeLong8 || typ == tiffTypeIFD8
}

// maxPreviewLength caps the served preview; a var so tests can lower it.
var maxPreviewLength int64 = 100 * 1024 * 1024

// extractEmbeddedJPEG walks the IFD chain incl. SubIFDs and returns the
// largest renderable embedded JPEG plus the container orientation.
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

	// classic TIFF (magic 42, 32-bit offsets) and BigTIFF (magic 43, 64-bit
	// offsets, allowed for DNG since spec 1.7) share the IFD layout, only with
	// wider entry-count, per-entry count and offset/value fields.
	var (
		bigTiff   bool
		firstIFD  uint64
		countSize int // width of the IFD entry-count field
		entrySize int // width of one IFD entry
		offW      int // width of an offset / inline value field
	)
	switch order.Uint16(data[2:4]) {
	case tiffMagic:
		countSize, entrySize, offW = 2, 12, 4
		firstIFD = uint64(order.Uint32(data[4:8]))
	case bigTiffMagic:
		// header carries the offset size (always 8) and a constant 0 before
		// the 8-byte IFD0 offset
		if len(data) < 16 || order.Uint16(data[4:6]) != 8 || order.Uint16(data[6:8]) != 0 {
			return nil, 0, thumbnailerErrors.ErrNoImageFromRawFile
		}
		bigTiff = true
		countSize, entrySize, offW = 8, 20, 8
		firstIFD = order.Uint64(data[8:16])
	default:
		return nil, 0, thumbnailerErrors.ErrNoImageFromRawFile
	}

	// readOff reads a structural offset (IFD0, next-IFD, SubIFD array pointer):
	// container width, 4 bytes classic / 8 bytes BigTIFF.
	readOff := func(b []byte) uint64 {
		if bigTiff {
			return order.Uint64(b[:8])
		}
		return uint64(order.Uint32(b[:4]))
	}
	// readVal reads a tag value at its declared type width, left-justified in the
	// value field. A LONG offset in a BigTIFF entry is 4 bytes, not 8, so reading
	// it as Uint64 would shift it on big-endian input.
	readVal := func(b []byte, typ uint16) uint64 {
		if is8ByteType(typ) {
			return order.Uint64(b[:8])
		}
		return uint64(order.Uint32(b[:4]))
	}
	valAt := 4 + offW // inline value field: after tag(2) + type(2) + count(offW)

	type candidate struct{ offset, length uint64 }
	var candidates []candidate
	var orientation uint16

	queue := []uint64{firstIFD}
	seen := map[uint64]struct{}{}
	// push caps the queue so a crafted SubIFD flood can't grow it unbounded
	push := func(offset uint64) {
		if len(queue) < maxIFDs {
			queue = append(queue, offset)
		}
	}
	dlen := uint64(len(data))
	first := true
	for len(queue) > 0 && len(seen) < maxIFDs {
		ifdOffset := queue[0]
		queue = queue[1:]
		if _, ok := seen[ifdOffset]; ok {
			continue
		}
		seen[ifdOffset] = struct{}{}
		// overflow-safe bound: a crafted BigTIFF offset near 2^64 must not wrap
		if ifdOffset == 0 || ifdOffset > dlen || dlen-ifdOffset < uint64(countSize) {
			continue
		}
		var entryCount int
		if bigTiff {
			entryCount = int(order.Uint64(data[ifdOffset : ifdOffset+8]))
		} else {
			entryCount = int(order.Uint16(data[ifdOffset : ifdOffset+2]))
		}
		if entryCount < 0 || entryCount > maxIFDEntries {
			continue
		}
		entriesEnd := ifdOffset + uint64(countSize) + uint64(entryCount)*uint64(entrySize)
		if entriesEnd+uint64(offW) > dlen {
			continue
		}

		var jpegOffset, jpegLength, stripOffset, stripLength uint64
		for i := 0; i < entryCount; i++ {
			entry := data[ifdOffset+uint64(countSize)+uint64(i*entrySize):]
			tag := order.Uint16(entry[0:2])
			typ := order.Uint16(entry[2:4])
			var count uint64
			if bigTiff {
				count = order.Uint64(entry[4:12])
			} else {
				count = uint64(order.Uint32(entry[4:8]))
			}
			switch tag {
			case tiffTagOrientation:
				// only IFD0's orientation applies
				if first && typ == tiffTypeShort && count == 1 {
					orientation = order.Uint16(entry[valAt : valAt+2])
				}
			case tiffTagJPEGOffset:
				if isLongType(typ, bigTiff) && count == 1 {
					jpegOffset = readVal(entry[valAt:], typ)
				}
			case tiffTagJPEGLength:
				if isLongType(typ, bigTiff) && count == 1 {
					jpegLength = readVal(entry[valAt:], typ)
				}
			case tiffTagStripOffsets:
				// only single-strip images can be a contiguous JPEG stream
				if isLongType(typ, bigTiff) && count == 1 {
					stripOffset = readVal(entry[valAt:], typ)
				}
			case tiffTagStripByteCounts:
				if isLongType(typ, bigTiff) && count == 1 {
					stripLength = readVal(entry[valAt:], typ)
				}
			case tiffTagSubIFDs:
				if !isSubIFDType(typ, bigTiff) {
					continue
				}
				if count == 1 {
					push(readVal(entry[valAt:], typ))
					continue
				}
				// count > 1: the value field points at an array of count offsets,
				// each of the tag's own width (LONG/IFD 4B, LONG8/IFD8 8B)
				arrayOffset := readOff(entry[valAt:])
				elemW := uint64(4)
				if is8ByteType(typ) {
					elemW = 8
				}
				for j := uint64(0); j < count && j < uint64(maxIFDs); j++ {
					pos := arrayOffset + j*elemW
					// overflow-safe bound: a crafted array offset must not wrap
					if pos < arrayOffset || pos > dlen || dlen-pos < elemW || len(queue) >= maxIFDs {
						break
					}
					push(readVal(data[pos:], typ))
				}
			}
		}
		if jpegOffset > 0 && jpegLength > 0 {
			candidates = append(candidates, candidate{jpegOffset, jpegLength})
		}
		if stripOffset > 0 && stripLength > 0 {
			candidates = append(candidates, candidate{stripOffset, stripLength})
		}
		push(readOff(data[entriesEnd:]))
		first = false
	}

	var best []byte
	for _, c := range candidates {
		end := c.offset + c.length
		if c.length == 0 || end < c.offset || end > dlen || int64(c.length) > maxPreviewLength || int(c.length) <= len(best) {
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

// isRenderableJPEG accepts only DCT processes common decoders render; DNG
// raw payloads are lossless JPEG (SOF3) behind a regular SOI marker.
func isRenderableJPEG(buf []byte) bool {
	if len(buf) < 4 || buf[0] != 0xff || buf[1] != 0xd8 {
		return false
	}
	i := 2
	for segments := 0; i+4 <= len(buf) && segments < maxJPEGSegments; segments++ {
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
	tiff = le.AppendUint16(tiff, tiffMagic)
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
