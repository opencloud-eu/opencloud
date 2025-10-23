package jpegmeta

import (
	"io"
)

var invalidSegment = segment{Marker: invalidMarker}

type segment struct {
	Marker marker
	Data   []byte
}

func makeSegment(markerType byte, r io.Reader) (segment, error) {
	m, err := makeMarker(markerType, r)
	return segment{Marker: m}, err
}

func readSegment(r io.Reader) (segment, error) {
	m, err := readMarker(r)
	if err != nil {
		return invalidSegment, err
	}

	seg := segment{
		Marker: m,
	}
	if m.DataLength > 0 {
		seg.Data = make([]byte, m.DataLength)

		_, err := io.ReadFull(r, seg.Data)
		if err != nil {
			return invalidSegment, err
		}
	}

	return seg, nil
}
