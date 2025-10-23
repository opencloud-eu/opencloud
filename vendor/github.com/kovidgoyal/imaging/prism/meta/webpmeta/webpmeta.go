package webpmeta

import (
	"errors"
	"fmt"
	"io"

	"github.com/kovidgoyal/imaging/prism/meta"
	"github.com/kovidgoyal/imaging/streams"
)

// Format specifies the image format handled by this package
var Format = meta.ImageFormat("WebP")

// Signature is FourCC bytes in the RIFF chunk, "RIFF????WEBP"
var webpSignature = [4]byte{'W', 'E', 'B', 'P'}

type webpFormat int

const (
	webpFormatSimple = webpFormat(iota)
	webpFormatLossless
	webpFormatExtended
)

// Bits per component is fixed in WebP
const bitsPerComponent = 8

// Load loads the metadata for a WebP image stream.
//
// Only as much of the stream is consumed as necessary to extract the metadata;
// the returned stream contains a buffered copy of the consumed data such that
// reading from it will produce the same results as fully reading the input
// stream. This provides a convenient way to load the full image after loading
// the metadata.
//
// An error is returned if basic metadata could not be extracted. The returned
// stream still provides the full image data.
func Load(r io.Reader) (md *meta.Data, imgStream io.Reader, err error) {
	imgStream, err = streams.CallbackWithSeekable(r, func(r io.Reader) (err error) {
		md, err = ExtractMetadata(r)
		return
	})
	return
}

// Same as Load() except that no new stream is provided
func ExtractMetadata(r io.Reader) (md *meta.Data, err error) {
	md = &meta.Data{Format: Format}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic while extracting image metadata: %v", r)
		}
	}()

	if err := verifySignature(r); err != nil {
		return nil, err
	}
	format, chunkLen, err := readWebPFormat(r)
	if err != nil {
		return nil, err
	}
	err = parseFormat(r, md, format, chunkLen)
	if err != nil {
		return nil, err
	}
	return md, nil
}

func parseFormat(r io.Reader, md *meta.Data, format webpFormat, chunkLen uint32) error {
	switch format {
	case webpFormatExtended:
		return parseWebpExtended(r, md, chunkLen)
	case webpFormatSimple:
		return parseWebpSimple(r, md, chunkLen)
	case webpFormatLossless:
		return parseWebpLossless(r, md, chunkLen)
	default:
		return errors.New("unknown WebP format")
	}
}

func parseWebpSimple(r io.Reader, md *meta.Data, chunkLen uint32) error {
	var buf [10]byte
	b := buf[:]
	if _, err := io.ReadFull(r, b); err != nil {
		return err
	}
	b = b[3:]
	if b[0] != 0x9d || b[1] != 0x01 || b[2] != 0x2a {
		return errors.New("corrupted WebP VP8 frame")
	}
	md.PixelWidth = uint32(b[4]&((1<<6)-1))<<8 | uint32(b[3])
	md.PixelWidth = uint32(b[6]&((1<<6)-1))<<8 | uint32(b[5])
	md.BitsPerComponent = bitsPerComponent
	return nil
}

func parseWebpLossless(r io.Reader, md *meta.Data, chunkLen uint32) error {
	var b [5]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return err
	}
	if b[0] != 0x2f {
		return errors.New("corrupted lossless WebP")
	}
	// Next 28 bits are width-1 and height-1.
	w := uint32(b[1])
	w |= uint32(b[2]&((1<<6)-1)) << 8
	w &= 0x3FFF

	h := uint32((b[2] >> 6) & ((1 << 2) - 1))
	h |= uint32(b[3]) << 2
	h |= uint32(b[4]&((1<<4)-1)) << 10
	h &= 0x3FFF

	md.PixelWidth = w + 1
	md.PixelHeight = h + 1
	md.BitsPerComponent = bitsPerComponent
	return nil
}

func parseWebpExtended(r io.Reader, md *meta.Data, chunkLen uint32) error {
	if chunkLen != 10 {
		return fmt.Errorf("unexpected VP8X chunk length: %d", chunkLen)
	}
	var hb [10]byte
	h := hb[:]
	if _, err := io.ReadFull(r, h); err != nil {
		return err
	}
	hasProfile := h[0]&(1<<5) != 0
	h = h[4:]
	w := uint32(h[0]) | uint32(h[1])<<8 | uint32(h[2])<<16
	ht := uint32(h[3]) | uint32(h[4])<<8 | uint32(h[5])<<16
	md.PixelWidth = w + 1
	md.PixelHeight = ht + 1
	md.BitsPerComponent = bitsPerComponent

	if hasProfile {
		data, err := readICCP(r, chunkLen)
		if err != nil {
			md.SetICCProfileError(err)
		} else {
			md.SetICCProfileData(data)
		}
	}

	return nil
}

func readICCP(r io.Reader, chunkLen uint32) ([]byte, error) {
	// Skip to the end of the chunk.
	if err := skip(r, chunkLen-10); err != nil {
		return nil, err
	}

	// ICCP _must_ be the next chunk.
	ch, err := readChunkHeader(r)
	if err != nil {
		return nil, err
	}
	if ch.ChunkType != chunkTypeICCP {
		return nil, errors.New("no expected ICCP chunk")
	}

	// Extract ICCP.
	data := make([]byte, ch.Length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data, nil
}

func verifySignature(r io.Reader) error {
	ch, err := readChunkHeader(r)
	if err != nil {
		return err
	}
	if ch.ChunkType != chunkTypeRIFF {
		return errors.New("missing RIFF header")
	}
	var fourcc [4]byte
	if _, err := io.ReadFull(r, fourcc[:]); err != nil {
		return err
	}
	if fourcc != webpSignature {
		return errors.New("not a WEBP file")
	}
	return nil
}

func readWebPFormat(r io.Reader) (format webpFormat, length uint32, err error) {
	ch, err := readChunkHeader(r)
	if err != nil {
		return 0, 0, err
	}
	switch ch.ChunkType {
	case chunkTypeVP8:
		return webpFormatSimple, ch.Length, nil
	case chunkTypeVP8L:
		return webpFormatLossless, ch.Length, nil
	case chunkTypeVP8X:
		return webpFormatExtended, ch.Length, nil
	default:
		return 0, 0, fmt.Errorf("unexpected WEBP format: %s", string(ch.ChunkType[:]))
	}
}

func skip(r io.Reader, length uint32) error {
	return streams.Skip(r, int64(length))
}
