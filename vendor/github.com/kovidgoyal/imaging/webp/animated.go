package webp

import (
	"bytes"
	"errors"
	"image"
	"io"

	"golang.org/x/image/riff"
	"golang.org/x/image/vp8"
	"golang.org/x/image/vp8l"
)

var (
	errNotExtended = errors.New("there was no vp8x header in this webp file, it cannot be animated")
	errNotAnimated = errors.New("the vp8x header did not have the animation bit set")
)

func decodeAnimated(r io.Reader) (*AnimatedWEBP, error) {
	riffReader, err := webpRiffReader(r)
	if err != nil {
		return nil, err
	}

	vp8xHeader, err := validateVP8XHeader(riffReader)
	if err != nil {
		return nil, err
	}

	animHeader, err := validateANIMHeader(riffReader)
	if err != nil {
		return nil, err
	}
	awp := AnimatedWEBP{
		Frames: make([]Frame, 0, 128),
		Header: animHeader,
		Config: image.Config{
			ColorModel: nil, // TODO(patricsss) set the color model correctly
			Width:      int(vp8xHeader.CanvasWidth),
			Height:     int(vp8xHeader.CanvasHeight),
		},
	}

	for {
		frame, err := parseFrame(riffReader)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		awp.Frames = append(awp.Frames, *frame)
	}

	return &awp, nil
}

func validateVP8XHeader(r *riff.Reader) (VP8XHeader, error) {
	fourCC, chunkLen, chunkData, err := r.Next()
	if err != nil {
		return VP8XHeader{}, err
	}
	if fourCC != fccVP8X {
		return VP8XHeader{}, errNotExtended
	}
	if chunkLen != 10 {
		return VP8XHeader{}, errInvalidFormat
	}

	h := parseVP8XHeader(chunkData)
	if !h.Animation {
		return VP8XHeader{}, errNotAnimated
	}

	return h, nil
}

func validateANIMHeader(r *riff.Reader) (ANIMHeader, error) {
	fourCC, chunkLen, chunkData, err := r.Next()
	if err != nil {
		return ANIMHeader{}, err
	}
	if fourCC != fccANIM {
		return ANIMHeader{}, errInvalidFormat
	}
	if chunkLen != 6 {
		return ANIMHeader{}, errInvalidFormat
	}

	h := parseANIMHeader(chunkData)

	return h, nil
}

func parseFrame(r *riff.Reader) (*Frame, error) {
	fourCC, chunkLen, chunkData, err := r.Next()
	if err != nil {
		return nil, err
	}
	if fourCC != fccANMF {
		return nil, errInvalidFormat
	}

	anmfHeader := parseANMFHeader(chunkData)

	// buffer chunk data based on chunkLen for safety
	// TODO(patricsss): establish if this is necessary, perhaps chunkData has a bounds
	// ANMF headers are 16 bytes
	wrappedChunkData, err := rewrap(chunkData, int(chunkLen-16))
	if err != nil {
		return nil, err
	}
	subReader := NewSubChunkReader(wrappedChunkData)

	var (
		alpha  []byte
		stride int
		i      *image.YCbCr
	)

	subFourCC, subChunkData, subChunkLen, err := subReader.Next()
	if subFourCC == fccALPH {
		alpha, stride, err = decodeAlpha(subChunkData, int(subChunkLen), anmfHeader)
		if err != nil {
			return nil, err
		}
		// read next chunk
		subFourCC, subChunkData, subChunkLen, err = subReader.Next()
		if err != nil {
			return nil, err
		}
	}
	var out image.Image
	switch subFourCC {
	case fccVP8:
		i, err = decodeVp8Bitstream(subChunkData, int(subChunkLen))
		if err != nil {
			return nil, err
		}
		if len(alpha) > 0 {
			out = &image.NYCbCrA{
				YCbCr:   *i,
				A:       alpha,
				AStride: stride,
			}
		} else {
			out = i
		}
	case fccVP8L:
		out, err = vp8l.Decode(subChunkData)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errInvalidFormat
	}

	return &Frame{
		Header: anmfHeader,
		Frame:  out,
	}, nil
}

func decodeVp8Bitstream(chunkData io.Reader, chunkLen int) (*image.YCbCr, error) {
	dec := vp8.NewDecoder()
	dec.Init(chunkData, chunkLen)

	_, err := dec.DecodeFrameHeader()
	if err != nil {
		return nil, err
	}

	i, err := dec.DecodeFrame()
	if err != nil {
		return nil, err
	}

	return i, nil
}

func decodeAlpha(chunkData io.Reader, chunkLen int, h ANMFHeader) (alpha []byte, alphaStride int, err error) {
	alphHeader := parseALPHHeader(chunkData)
	// Length of the chunk minus 1 byte for the ALPH header
	buf := make([]byte, chunkLen-1)
	n, err := io.ReadFull(chunkData, buf)
	if err != nil {
		return nil, 0, err
	}
	if n != len(buf) {
		return nil, 0, errInvalidFormat
	}

	alpha, alphaStride, err = readAlpha(bytes.NewReader(buf), h.FrameWidth-1, h.FrameHeight-1, alphHeader.Compression)
	unfilterAlpha(alpha, alphaStride, alphHeader.FilteringMethod)
	return alpha, alphaStride, nil
}

func rewrap(r io.Reader, length int) (io.Reader, error) {
	data := make([]byte, length)
	n, err := io.ReadFull(r, data)
	if err != nil {
		return nil, err
	}
	if n != length {
		return nil, errInvalidFormat
	}
	return bytes.NewReader(data), nil
}
