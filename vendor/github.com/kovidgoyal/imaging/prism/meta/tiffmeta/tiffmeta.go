package tiffmeta

import (
	"fmt"
	"image/color"
	"io"
	"strings"

	"github.com/kovidgoyal/imaging/prism/meta"
	"github.com/kovidgoyal/imaging/types"
	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/image/tiff"
)

var _ = fmt.Print

func BitsPerComponent(c color.Model) uint32 {
	switch c {
	case color.RGBAModel, color.NRGBAModel, color.YCbCrModel, color.CMYKModel:
		return 8
	case color.GrayModel:
		return 8
	case color.Gray16Model:
		return 16
	case color.AlphaModel:
		return 8
	case color.Alpha16Model:
		return 16
	default:
		// This handles paletted images and other custom color models.
		// For a palette, each color in the palette has its own depth.
		// We can check the bit depth by converting a color from the model to RGBA.
		// The `Convert` method is part of the color.Model interface.
		// A fully opaque red color is used for this check.
		r, g, b, a := c.Convert(color.RGBA{R: 255, A: 255}).RGBA()

		// The values returned by RGBA() are 16-bit alpha-premultiplied values (0-65535).
		// If the highest value is <= 255, it's an 8-bit model.
		if r|g|b|a <= 0xff {
			return 8
		} else {
			return 16
		}
	}
}

func ExtractMetadata(r_ io.Reader) (md *meta.Data, err error) {
	r := r_.(io.ReadSeeker)
	pos, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}
	c, err := tiff.DecodeConfig(r)
	if err != nil {
		if strings.Contains(err.Error(), "malformed header") {
			err = nil
		}
		return nil, err
	}
	md = &meta.Data{
		Format: types.TIFF, PixelWidth: uint32(c.Width), PixelHeight: uint32(c.Height),
		BitsPerComponent: BitsPerComponent(c.ColorModel),
	}
	if _, err = r.Seek(pos, io.SeekStart); err != nil {
		return nil, err
	}
	if e, err := exif.Decode(r); err == nil {
		md.SetExif(e)
	} else {
		md.SetExifError(err)
	}
	return md, nil
}
