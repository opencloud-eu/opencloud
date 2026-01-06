package jpegmeta

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/kovidgoyal/go-parallel"
	"github.com/kovidgoyal/imaging/prism/meta"
	"github.com/kovidgoyal/imaging/streams"
	"github.com/kovidgoyal/imaging/types"
)

const exifSignature = "Exif\x00\x00"

var iccProfileIdentifier = []byte("ICC_PROFILE\x00")

// Load loads the metadata for a JPEG image stream.
//
// Only as much of the stream is consumed as necessary to extract the metadata;
// the returned stream contains a buffered copy of the consumed data such that
// reading from it will produce the same results as fully reading the input
// stream. This provides a convenient way to load the full image after loading
// the metadata.
func Load(r io.Reader) (md *meta.Data, imgStream io.Reader, err error) {
	imgStream, err = streams.CallbackWithSeekable(r, func(r io.Reader) (err error) {
		md, err = ExtractMetadata(r)
		return
	})
	return
}

// Same as Load() except that no new stream is provided
func ExtractMetadata(r io.Reader) (md *meta.Data, err error) {
	metadataExtracted := false
	md = &meta.Data{Format: types.JPEG}
	segReader := NewSegmentReader(r)

	defer func() {
		if r := recover(); r != nil {
			if !metadataExtracted {
				md = nil
			}
			err = parallel.Format_stacktrace_on_panic(r, 1)
		}
	}()

	var iccProfileChunks [][]byte
	var iccProfileChunksExtracted int
	var exif []byte

	allMetadataExtracted := func() bool {
		return metadataExtracted &&
			iccProfileChunks != nil &&
			iccProfileChunksExtracted == len(iccProfileChunks) &&
			exif != nil
	}

	soiSegment, err := segReader.ReadSegment()
	if err != nil {
		if q := err.Error(); strings.Contains(q, "invalid marker identifier") || strings.Contains(q, "unrecognised marker type") {
			err = nil
		}
		return nil, err
	}
	if soiSegment.Marker.Type != markerTypeStartOfImage {
		return nil, nil
	}

parseSegments:
	for {
		segment, err := segReader.ReadSegment()
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("unexpected EOF")
			}
			return nil, err
		}

		switch segment.Marker.Type {

		case markerTypeStartOfFrameBaseline,
			markerTypeStartOfFrameProgressive:
			md.BitsPerComponent = uint32(segment.Data[0])
			md.PixelHeight = uint32(segment.Data[1])<<8 | uint32(segment.Data[2])
			md.PixelWidth = uint32(segment.Data[3])<<8 | uint32(segment.Data[4])
			metadataExtracted = true

			if allMetadataExtracted() {
				break parseSegments
			}

		case markerTypeStartOfScan,
			markerTypeEndOfImage:
			break parseSegments

		case markerTypeApp1:
			if bytes.HasPrefix(segment.Data, []byte(exifSignature)) {
				exif = segment.Data
			}
		case markerTypeApp2:
			if len(segment.Data) < len(iccProfileIdentifier)+2 {
				continue
			}

			for i := range iccProfileIdentifier {
				if segment.Data[i] != iccProfileIdentifier[i] {
					continue parseSegments
				}
			}

			iccData, iccErr := md.ICCProfileData()
			if iccData != nil || iccErr != nil {
				continue
			}

			chunkTotal := segment.Data[len(iccProfileIdentifier)+1]
			if iccProfileChunks == nil {
				iccProfileChunks = make([][]byte, chunkTotal)
			} else if int(chunkTotal) != len(iccProfileChunks) {
				md.SetICCProfileError(fmt.Errorf("inconsistent ICC profile chunk count"))
				continue
			}

			chunkNum := segment.Data[len(iccProfileIdentifier)]
			if chunkNum == 0 || int(chunkNum) > len(iccProfileChunks) {
				md.SetICCProfileError(fmt.Errorf("invalid ICC profile chunk number"))
				continue
			}
			if iccProfileChunks[chunkNum-1] != nil {
				md.SetICCProfileError(fmt.Errorf("duplicated ICC profile chunk"))
				continue
			}
			iccProfileChunksExtracted++
			iccProfileChunks[chunkNum-1] = segment.Data[len(iccProfileIdentifier)+2:]

			if allMetadataExtracted() {
				break parseSegments
			}
		}
	}

	if !metadataExtracted {
		return nil, fmt.Errorf("no metadata found")
	}
	md.SetExifData(exif)

	// Incomplete or missing ICC profile
	if len(iccProfileChunks) != iccProfileChunksExtracted {
		_, iccErr := md.ICCProfileData()
		if iccErr == nil {
			md.SetICCProfileError(fmt.Errorf("incomplete ICC profile data"))
		}
		return md, nil
	}

	iccProfileData := bytes.Buffer{}
	for i := range iccProfileChunks {
		iccProfileData.Write(iccProfileChunks[i])
	}
	md.SetICCProfileData(iccProfileData.Bytes())

	return md, nil
}
