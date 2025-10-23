package jpegmeta

import "fmt"

type markerType int

const (
	markerTypeInvalid                 markerType = 0x00
	markerTypeStartOfFrameBaseline    markerType = 0xc0
	markerTypeStartOfFrameProgressive markerType = 0xc2
	markerTypeDefineHuffmanTable      markerType = 0xc4
	markerTypeRestart0                markerType = 0xd0
	markerTypeRestart1                markerType = 0xd1
	markerTypeRestart2                markerType = 0xd2
	markerTypeRestart3                markerType = 0xd3
	markerTypeRestart4                markerType = 0xd4
	markerTypeRestart5                markerType = 0xd5
	markerTypeRestart6                markerType = 0xd6
	markerTypeRestart7                markerType = 0xd7
	markerTypeStartOfImage            markerType = 0xd8
	markerTypeEndOfImage              markerType = 0xd9
	markerTypeStartOfScan             markerType = 0xda
	markerTypeDefineQuantisationTable markerType = 0xdb
	markerTypeDefineRestartInterval   markerType = 0xdd
	markerTypeApp0                    markerType = 0xe0
	markerTypeApp1                    markerType = 0xe1
	markerTypeApp2                    markerType = 0xe2
	markerTypeApp3                    markerType = 0xe3
	markerTypeApp4                    markerType = 0xe4
	markerTypeApp5                    markerType = 0xe5
	markerTypeApp6                    markerType = 0xe6
	markerTypeApp7                    markerType = 0xe7
	markerTypeApp8                    markerType = 0xe8
	markerTypeApp9                    markerType = 0xe9
	markerTypeApp10                   markerType = 0xea
	markerTypeApp11                   markerType = 0xeb
	markerTypeApp12                   markerType = 0xec
	markerTypeApp13                   markerType = 0xed
	markerTypeApp14                   markerType = 0xee
	markerTypeApp15                   markerType = 0xef
	markerTypeComment                 markerType = 0xfe
)

func (mt markerType) String() string {
	switch mt {
	case markerTypeStartOfFrameBaseline:
		return "SOF0"
	case markerTypeStartOfFrameProgressive:
		return "SOF2"
	case markerTypeDefineHuffmanTable:
		return "DHT"
	case markerTypeRestart0:
		return "RST0"
	case markerTypeRestart1:
		return "RST1"
	case markerTypeRestart2:
		return "RST2"
	case markerTypeRestart3:
		return "RST3"
	case markerTypeRestart4:
		return "RST4"
	case markerTypeRestart5:
		return "RST5"
	case markerTypeRestart6:
		return "RST6"
	case markerTypeRestart7:
		return "RST7"
	case markerTypeStartOfImage:
		return "SOI"
	case markerTypeEndOfImage:
		return "EOI"
	case markerTypeStartOfScan:
		return "SOS"
	case markerTypeDefineQuantisationTable:
		return "DQT"
	case markerTypeDefineRestartInterval:
		return "DRI"
	case markerTypeApp0:
		return "APP0"
	case markerTypeApp1:
		return "APP1"
	case markerTypeApp2:
		return "APP2"
	case markerTypeApp3:
		return "APP3"
	case markerTypeApp4:
		return "APP4"
	case markerTypeApp5:
		return "APP5"
	case markerTypeApp6:
		return "APP6"
	case markerTypeApp7:
		return "APP7"
	case markerTypeApp8:
		return "APP8"
	case markerTypeApp9:
		return "APP9"
	case markerTypeApp10:
		return "APP10"
	case markerTypeApp11:
		return "APP11"
	case markerTypeApp12:
		return "APP12"
	case markerTypeApp13:
		return "APP13"
	case markerTypeApp14:
		return "APP14"
	case markerTypeApp15:
		return "APP15"
	case markerTypeComment:
		return "COM"
	default:
		return fmt.Sprintf("Unknown (%0x)", byte(mt))
	}
}
