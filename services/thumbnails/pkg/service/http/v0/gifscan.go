package svc

import "errors"

// gifFrames counts the frames a gif declares and measures the largest of them,
// walking the block structure because image/gif can only answer this by
// decoding every frame, which is the allocation the limits exist to prevent.
func gifFrames(data []byte) (frames, widest, tallest int, err error) {
	errMalformed := errors.New("malformed gif")

	// header (6) + logical screen descriptor (7), the flags byte is the last one
	if len(data) < 13 {
		return 0, 0, 0, errMalformed
	}
	at := 13
	if data[10]&0x80 != 0 {
		at += colorTableSize(data[10])
	}

	for {
		if at >= len(data) {
			return 0, 0, 0, errMalformed
		}
		switch data[at] {
		case 0x3b: // trailer
			return frames, widest, tallest, nil
		case 0x21: // extension: label, then sub-blocks
			at += 2
			if at, err = skipSubBlocks(data, at); err != nil {
				return 0, 0, 0, err
			}
		case 0x2c: // image descriptor: 9 bytes, an optional table, then the frame
			if at+10 > len(data) {
				return 0, 0, 0, errMalformed
			}
			frames++
			widest = max(widest, int(data[at+5])|int(data[at+6])<<8)
			tallest = max(tallest, int(data[at+7])|int(data[at+8])<<8)

			flags := data[at+9]
			at += 10
			if flags&0x80 != 0 {
				at += colorTableSize(flags)
			}
			at++ // LZW minimum code size
			if at, err = skipSubBlocks(data, at); err != nil {
				return 0, 0, 0, err
			}
		default:
			return 0, 0, 0, errMalformed
		}
	}
}

func colorTableSize(flags byte) int { return 3 << ((flags & 0x07) + 1) }

// skipSubBlocks walks the length-prefixed blocks up to the terminating empty one.
func skipSubBlocks(data []byte, at int) (int, error) {
	for {
		if at >= len(data) {
			return 0, errors.New("malformed gif")
		}
		size := int(data[at])
		at++
		if size == 0 {
			return at, nil
		}
		at += size
	}
}
