package jpegmeta

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/kovidgoyal/imaging/streams"
)

var invalidMarker = marker{Type: markerTypeInvalid}

type marker struct {
	Type       markerType
	DataLength int
}

func makeMarker(mType byte, r io.Reader) (marker, error) {
	var length uint16
	switch mType {

	case
		byte(markerTypeRestart0),
		byte(markerTypeRestart1),
		byte(markerTypeRestart2),
		byte(markerTypeRestart3),
		byte(markerTypeRestart4),
		byte(markerTypeRestart5),
		byte(markerTypeRestart6),
		byte(markerTypeRestart7),
		byte(markerTypeStartOfImage),
		byte(markerTypeEndOfImage):

		length = 2

	case byte(markerTypeStartOfFrameBaseline),
		byte(markerTypeStartOfFrameProgressive),
		byte(markerTypeDefineHuffmanTable),
		byte(markerTypeStartOfScan),
		byte(markerTypeDefineQuantisationTable),
		byte(markerTypeDefineRestartInterval),
		byte(markerTypeApp0),
		byte(markerTypeApp1),
		byte(markerTypeApp2),
		byte(markerTypeApp3),
		byte(markerTypeApp4),
		byte(markerTypeApp5),
		byte(markerTypeApp6),
		byte(markerTypeApp7),
		byte(markerTypeApp8),
		byte(markerTypeApp9),
		byte(markerTypeApp10),
		byte(markerTypeApp11),
		byte(markerTypeApp12),
		byte(markerTypeApp13),
		byte(markerTypeApp14),
		byte(markerTypeApp15),
		byte(markerTypeComment):

		var err error
		if err = binary.Read(r, binary.BigEndian, &length); err != nil {
			return invalidMarker, err
		}

	default:
		return invalidMarker, fmt.Errorf("unrecognised marker type %0x", mType)
	}

	return marker{
		Type:       markerType(mType),
		DataLength: int(length) - 2,
	}, nil
}

func readMarker(r io.Reader) (marker, error) {
	b, err := streams.ReadByte(r)
	if err != nil {
		return invalidMarker, err
	}

	if b != 0xff {
		return invalidMarker, fmt.Errorf("invalid marker identifier %0x", b)
	}
	if b, err = streams.ReadByte(r); err != nil {
		return invalidMarker, err
	}

	return makeMarker(b, r)
}
