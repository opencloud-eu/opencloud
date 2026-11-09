package svc

import (
	"image"
	"image/color"
	"image/draw"
)

// palettedAtOrigin re-pallettes img using Floyd-Steinberg dithering onto a new
// palette whose bounds are anchored at the origin. The source's own (possibly
// offset) bounds are honored so the pixel content lands in the right place, but
// the result always starts at (0,0). This is required for gif frames: after a
// resize the frame rectangle inherits the source frame's offset, and a frame
// whose rect extends past the gif logical screen makes gif.EncodeAll fail with
// "image block is out of bounds".
func palettedAtOrigin(img image.Image, p color.Palette) *image.Paletted {
	b := img.Bounds()
	pm := image.NewPaletted(image.Rect(0, 0, b.Dx(), b.Dy()), p)
	draw.FloydSteinberg.Draw(pm, pm.Bounds(), shiftedImage(img, b.Min), image.Point{})
	return pm
}

// shiftedImage wraps img so its bounds are shifted by off, letting a draw target
// anchored at the origin receive an image whose content is defined relative to a
// different origin.
type shiftedImageWrapper struct {
	img image.Image
	off image.Point
}

func (s shiftedImageWrapper) Bounds() image.Rectangle { return s.img.Bounds().Add(s.off) }
func (s shiftedImageWrapper) At(x, y int) color.Color { return s.img.At(x-s.off.X, y-s.off.Y) }
func (s shiftedImageWrapper) ColorModel() color.Model { return s.img.ColorModel() }

func shiftedImage(img image.Image, off image.Point) image.Image {
	return shiftedImageWrapper{img: img, off: off}
}
