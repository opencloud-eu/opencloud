package webp

import (
	"image"
	"image/color"
	"io"
)

// AnimatedWEBP is the struct of a AnimatedWEBP container and the image data contained within.
type AnimatedWEBP struct {
	Frames []Frame
	Header ANIMHeader
	Config image.Config
}

type Frame struct {
	Header ANMFHeader
	Frame  image.Image
}

type VP8XHeader struct {
	ICCProfile   bool
	Alpha        bool
	ExifMetadata bool
	XmpMetadata  bool
	Animation    bool
	CanvasWidth  uint32
	CanvasHeight uint32
}

type ALPHHeader struct {
	Preprocessing   uint8
	FilteringMethod uint8
	Compression     uint8
}

type ANIMHeader struct {
	BackgroundColor color.Color
	LoopCount       uint16
}

type ANMFHeader struct {
	FrameX         uint32
	FrameY         uint32
	FrameWidth     uint32
	FrameHeight    uint32
	FrameDuration  uint32
	AlphaBlend     bool
	DisposalBitSet bool
}

func parseALPHHeader(r io.Reader) ALPHHeader {
	h := make([]byte, 1)
	_, _ = io.ReadFull(r, h)

	const (
		twoBits = byte(3)
	)

	return ALPHHeader{
		Preprocessing:   h[0] >> 4 & twoBits,
		FilteringMethod: h[0] >> 2 & twoBits,
		Compression:     h[0] & twoBits,
	}
}

func parseANIMHeader(r io.Reader) ANIMHeader {
	h := make([]byte, 6)
	_, _ = io.ReadFull(r, h)

	loopCount := uint16(h[4]) | uint16(h[5])<<8
	bg := color.RGBA{
		R: h[2],
		G: h[1],
		B: h[0],
		A: h[3],
	}

	return ANIMHeader{
		BackgroundColor: bg,
		LoopCount:       loopCount,
	}
}

func parseANMFHeader(r io.Reader) ANMFHeader {
	h := make([]byte, 16)
	_, _ = io.ReadFull(r, h)

	const (
		disposeBit = 1
		blendBit   = 1 << 1
	)

	return ANMFHeader{
		FrameX:         u24(h[0:3]),
		FrameY:         u24(h[3:6]),
		FrameWidth:     u24(h[6:9]) + 1,
		FrameHeight:    u24(h[9:12]) + 1,
		FrameDuration:  u24(h[12:15]),
		AlphaBlend:     (h[15] & blendBit) == 0,
		DisposalBitSet: (h[15] & disposeBit) != 0,
	}
}

func parseVP8XHeader(r io.Reader) VP8XHeader {
	const (
		anim  = 1 << 1
		xmp   = 1 << 2
		exif  = 1 << 3
		alpha = 1 << 4
		icc   = 1 << 5
	)

	h := make([]byte, 10)
	_, _ = io.ReadFull(r, h)

	widthMinusOne := u24(h[4:])
	heightMinusOne := u24(h[7:])

	header := VP8XHeader{
		ICCProfile:   h[0]&icc != 0,
		Alpha:        h[0]&alpha != 0,
		ExifMetadata: h[0]&exif != 0,
		XmpMetadata:  h[0]&xmp != 0,
		Animation:    h[0]&anim != 0,
		CanvasWidth:  widthMinusOne + 1,
		CanvasHeight: heightMinusOne + 1,
	}
	return header
}

func u24(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16
}

func u32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}
