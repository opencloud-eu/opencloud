package jpegmeta

import (
	"io"

	"github.com/kovidgoyal/imaging/streams"
)

type segmentReader struct {
	reader             io.Reader
	inEntropyCodedData bool
}

func (sr *segmentReader) ReadSegment() (segment, error) {
	if sr.inEntropyCodedData {
		for {
			b, err := streams.ReadByte(sr.reader)
			if err != nil {
				return segment{}, err
			}

			if b == 0xFF {
				if b, err = streams.ReadByte(sr.reader); err != nil {
					return segment{}, err
				}

				if b != 0x00 {
					seg, err := makeSegment(b, sr.reader)
					if err != nil {
						return segment{}, err
					}

					sr.inEntropyCodedData = seg.Marker.Type == markerTypeStartOfScan ||
						(seg.Marker.Type >= markerTypeRestart0 && seg.Marker.Type <= markerTypeRestart7)

					return seg, err
				}
			}
		}
	}

	seg, err := readSegment(sr.reader)
	if err != nil {
		return seg, err
	}

	sr.inEntropyCodedData = seg.Marker.Type == markerTypeStartOfScan

	return seg, nil
}

func NewSegmentReader(r io.Reader) *segmentReader {
	return &segmentReader{
		reader: r,
	}
}
