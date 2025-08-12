//go:build enable_vips

package thumbnail

import (
	"bytes"
	"image"
	"strings"

	"github.com/cshum/vipsgen/vips"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
	"golang.org/x/image/bmp"
)

// SimpleGenerator is the default image generator and is used for all image types expect gif.
type SimpleGenerator struct {
	crop    vips.Interesting
	size    vips.Size
	process string
}

func NewSimpleGenerator(filetype, process string) (SimpleGenerator, error) {
	switch strings.ToLower(process) {
	case "thumbnail":
		return SimpleGenerator{crop: vips.InterestingAttention, process: process, size: vips.SizeBoth}, nil
	case "fit":
		return SimpleGenerator{crop: vips.InterestingNone, process: process, size: vips.SizeBoth}, nil
	case "resize":
		return SimpleGenerator{crop: vips.InterestingNone, process: process, size: vips.SizeForce}, nil
	default:
		return SimpleGenerator{crop: vips.InterestingAttention, process: process, size: vips.SizeBoth}, nil
	}
}

// ProcessorID returns the processor identification.
func (g SimpleGenerator) ProcessorID() string {
	return g.process
}

// Generate generates a alternative image version.
func (g SimpleGenerator) Generate(size image.Rectangle, img interface{}) (interface{}, error) {
	var m *vips.Image
	var err error
	switch img.(type) {
	case *image.RGBA:
		// This comes from the txt preprocessor
		var buf bytes.Buffer
		if err = bmp.Encode(&buf, img.(*image.RGBA)); err != nil {
			return nil, err
		}
		m, err = vips.NewImageFromBuffer(buf.Bytes(), nil)
		if err != nil {
			return nil, err
		}

	case *vips.Image:
		m = img.(*vips.Image)
	default:
		return nil, errors.ErrInvalidType
	}
	if err := m.ThumbnailImage(size.Dx(), &vips.ThumbnailImageOptions{Crop: g.crop, Size: g.size}); err != nil {
		return nil, err
	}

	if err := m.RemoveExif(); err != nil {
		return nil, err
	}

	return m, nil
}

func (g SimpleGenerator) Dimensions(img interface{}) (image.Rectangle, error) {
	switch img.(type) {
	case *image.RGBA:
		m := img.(*image.RGBA)
		return m.Bounds(), nil
	case *vips.Image:
		m := img.(*vips.Image)
		return image.Rect(0, 0, m.Width(), m.Height()), nil
	default:
		return image.Rectangle{}, errors.ErrInvalidType
	}
}
