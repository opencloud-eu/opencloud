package pngmeta

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/kovidgoyal/imaging/prism/meta"
	"github.com/kovidgoyal/imaging/streams"
)

// Format specifies the image format handled by this package
var Format = meta.ImageFormat("PNG")

var pngSignature = [8]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}

// Load loads the metadata for a PNG image stream.
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

func read_chunk(r io.Reader, length uint32) (ans []byte, err error) {
	ans = make([]byte, length+4)
	_, err = io.ReadFull(r, ans)
	ans = ans[:len(ans)-4] // we dont care about the chunk CRC
	return
}

func skip_chunk(r io.Reader, length uint32) (err error) {
	return streams.Skip(r, int64(length)+4)
}

// Same as Load() except that no new stream is provided
func ExtractMetadata(r io.Reader) (md *meta.Data, err error) {
	metadataExtracted := false
	md = &meta.Data{Format: Format}

	defer func() {
		if r := recover(); r != nil {
			if !metadataExtracted {
				md = nil
			}
			err = fmt.Errorf("panic while extracting image metadata: %v", r)
		}
	}()

	allMetadataExtracted := func() bool {
		iccData, iccErr := md.ICCProfileData()
		return metadataExtracted && (iccData != nil || iccErr != nil)
	}

	pngSig := [8]byte{}
	if _, err := io.ReadFull(r, pngSig[:]); err != nil {
		return nil, err
	}
	if pngSig != pngSignature {
		return nil, fmt.Errorf("invalid PNG signature")
	}
	var chunk []byte

	decode := func(target any) error {
		if n, err := binary.Decode(chunk, binary.BigEndian, target); err == nil {
			chunk = chunk[n:]
			return nil
		} else {
			return err
		}
	}

parseChunks:
	for {
		ch, err := readChunkHeader(r)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		switch ch.ChunkType {

		case chunkTypeIHDR:
			if chunk, err = read_chunk(r, ch.Length); err != nil {
				return nil, err
			}
			if err = decode(&md.PixelWidth); err != nil {
				return nil, err
			}
			if err = decode(&md.PixelHeight); err != nil {
				return nil, err
			}
			md.BitsPerComponent = uint32(chunk[0])
			metadataExtracted = true
			if allMetadataExtracted() {
				break parseChunks
			}

		case chunkTypeiCCP:
			if chunk, err = read_chunk(r, ch.Length); err != nil {
				return nil, err
			}
			idx := bytes.IndexByte(chunk, 0)
			if idx < 0 || idx > 80 {
				return nil, fmt.Errorf("null terminator not found reading ICC profile name")
			}
			chunk = chunk[idx+1:]
			if len(chunk) < 1 {
				return nil, fmt.Errorf("incomplete ICCP chunk in PNG file")
			}
			if compressionMethod := chunk[0]; compressionMethod != 0x00 {
				return nil, fmt.Errorf("unknown compression method (%d)", compressionMethod)
			}
			chunk = chunk[1:]
			// Decompress ICC profile data
			zReader, err := zlib.NewReader(bytes.NewReader(chunk))
			if err != nil {
				md.SetICCProfileError(err)
				break
			}
			defer zReader.Close()
			profileData := &bytes.Buffer{}
			_, err = io.Copy(profileData, zReader)
			if err == nil {
				md.SetICCProfileData(profileData.Bytes())
				if allMetadataExtracted() {
					break parseChunks
				}
			} else {
				md.SetICCProfileError(err)
			}

		case chunkTypeIDAT, chunkTypeIEND:
			break parseChunks

		default:
			if err = skip_chunk(r, ch.Length); err != nil {
				return nil, err
			}
		}
	}

	if !metadataExtracted {
		return nil, fmt.Errorf("no metadata found")
	}

	return md, nil
}
