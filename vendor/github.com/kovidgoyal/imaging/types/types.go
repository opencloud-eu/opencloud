package types

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
)

var _ = fmt.Print

// Format is an image file format.
type Format int

type TransformType int

const (
	NoTransform TransformType = iota
	FlipHTransform
	FlipVTransform
	Rotate90Transform
	Rotate180Transform
	Rotate270Transform
	TransverseTransform
	TransposeTransform
)

// Image file formats.
const (
	UNKNOWN Format = iota
	JPEG
	PNG
	GIF
	TIFF
	WEBP
	BMP
	PBM
	PGM
	PPM
	PAM
)

var FormatExts = map[string]Format{
	"jpg":  JPEG,
	"jpeg": JPEG,
	"png":  PNG,
	"gif":  GIF,
	"tif":  TIFF,
	"tiff": TIFF,
	"webp": WEBP,
	"bmp":  BMP,
	"pbm":  PBM,
	"pgm":  PGM,
	"ppm":  PPM,
	"pam":  PAM,
}

var formatNames = map[Format]string{
	JPEG: "JPEG",
	PNG:  "PNG",
	GIF:  "GIF",
	TIFF: "TIFF",
	WEBP: "WEBP",
	BMP:  "BMP",
	PBM:  "PBM",
	PGM:  "PGM",
	PPM:  "PPM",
	PAM:  "PAM",
}

func (f Format) String() string {
	return formatNames[f]
}

func (s Format) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *Format) UnmarshalJSON(data []byte) error {
	var statusString string
	if err := json.Unmarshal(data, &statusString); err != nil {
		return err
	}
	for x, ss := range formatNames {
		if ss == statusString {
			*s = x
			return nil
		}
	}
	return fmt.Errorf("unknown image format: %s", statusString)
}

type Scanner interface {
	Scan(x1, y1, x2, y2 int, dst []uint8)
	ScanRow(x1, y1, x2, y2 int, img image.Image, row int)
	Bytes_per_channel() int
	Num_of_channels() int
	Bounds() image.Rectangle
	ReverseRow(image.Image, int)
	NewImage(r image.Rectangle) image.Image
}

type Input struct {
	Reader io.Reader
	Path   string
}
