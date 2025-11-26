package webp

import (
	"bytes"
	"errors"
	"io"

	"golang.org/x/image/riff"
)

var (
	errInvalidHeader = errors.New("could not read an 8 byte header, sub-chunk is not valid")
)

// SubChunkReader helps in reading riff data from an existing chunk which are comprised of sub-chunks.
// A good example would be ANMF chunks of animated webp files. These chunks can contain headers, ALPH chunks
// and VP8 or VP8L chunks within the main riff data chunk.
type SubChunkReader struct {
	r io.Reader
}

// Next will return the FourCC, data, and data length of a subchunk.
// The io.Reader returned for data will not be the same as the provided reader
// and is safe to discord without fully reading the contents.
// Will return an error if the format is invalid or an underlying read operation fails.
func (c SubChunkReader) Next() (riff.FourCC, io.Reader, uint32, error) {
	header := make([]byte, 8)
	n, err := io.ReadFull(c.r, header)
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			return riff.FourCC{}, nil, 0, errInvalidHeader
		}
		return riff.FourCC{}, nil, 0, err
	}
	if n != 8 {
		return riff.FourCC{}, nil, 0, errInvalidHeader
	}

	fourCC := riff.FourCC{header[0], header[1], header[2], header[3]}
	chunkLen := u32(header[4:8])
	buf := make([]byte, chunkLen)
	n, err = io.ReadFull(c.r, buf)
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			return riff.FourCC{}, nil, 0, errInvalidFormat
		}
		return riff.FourCC{}, nil, 0, err
	}
	if n != int(chunkLen) {
		return riff.FourCC{}, nil, 0, errInvalidFormat
	}

	// if chunkLen was odd, we need to maintain a 2-byte boundary per RIFF spec.
	// in this case read off a single byte of padding to re-align with the next
	// fourCC header
	if chunkLen%2 == 1 {
		n, err := c.r.Read([]byte{0})
		if err != nil {
			return riff.FourCC{}, nil, 0, err
		}
		if n != 1 {
			return riff.FourCC{}, nil, 0, errInvalidFormat
		}
	}

	return fourCC, bytes.NewReader(buf), chunkLen, nil
}

func NewSubChunkReader(r io.Reader) *SubChunkReader {
	return &SubChunkReader{
		r: r,
	}
}
