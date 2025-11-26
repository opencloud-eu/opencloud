package imaging

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"slices"

	"github.com/kovidgoyal/imaging/nrgb"
	"github.com/kovidgoyal/imaging/nrgba"
)

var _ = fmt.Println

// New creates a new image with the specified width and height, and fills it with the specified color.
func New(width, height int, fillColor color.Color) *image.NRGBA {
	if width <= 0 || height <= 0 {
		return &image.NRGBA{}
	}

	c := color.NRGBAModel.Convert(fillColor).(color.NRGBA)
	if (c == color.NRGBA{0, 0, 0, 0}) {
		return image.NewNRGBA(image.Rect(0, 0, width, height))
	}

	return &image.NRGBA{
		Pix:    bytes.Repeat([]byte{c.R, c.G, c.B, c.A}, width*height),
		Stride: 4 * width,
		Rect:   image.Rect(0, 0, width, height),
	}
}

// Clone returns a copy of the given image.
func Clone(img image.Image) *image.NRGBA {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	src := nrgba.NewNRGBAScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	size := w * 4
	if err := run_in_parallel_over_range(0, func(start, limit int) {
		for y := start; y < limit; y++ {
			i := y * dst.Stride
			src.Scan(0, y, w, y+1, dst.Pix[i:i+size])
		}
	}, 0, h); err != nil {
		panic(err)
	}
	return dst
}

func ClonePreservingOrigin(img image.Image) *image.NRGBA {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	src := nrgba.NewNRGBAScanner(img)
	dst := image.NewNRGBA(img.Bounds())
	size := w * 4
	if err := run_in_parallel_over_range(0, func(start, limit int) {
		for y := start; y < limit; y++ {
			i := y * dst.Stride
			src.Scan(0, y, w, y+1, dst.Pix[i:i+size])
		}
	}, 0, h); err != nil {
		panic(err)
	}
	return dst
}

func AsNRGBA(src image.Image) *image.NRGBA {
	if nrgba, ok := src.(*image.NRGBA); ok {
		return nrgba
	}
	return ClonePreservingOrigin(src)
}

func AsNRGB(src image.Image) *NRGB {
	if nrgb, ok := src.(*NRGB); ok {
		return nrgb
	}
	sc := nrgb.NewNRGBScanner(src, nrgb.Color{})
	dst := sc.NewImage(src.Bounds()).(*nrgb.Image)
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	if err := run_in_parallel_over_range(0, func(start, limit int) {
		for y := start; y < limit; y++ {
			sc.ScanRow(0, y, w, y+1, dst, y)
		}
	}, 0, h); err != nil {
		panic(err)
	}
	return dst
}

// Clone an image preserving it's type for all known image types or returning an NRGBA64 image otherwise
func ClonePreservingType(src image.Image) image.Image {
	switch src := src.(type) {
	case *image.RGBA:
		dst := *src
		dst.Pix = slices.Clone(src.Pix)
		return &dst
	case *image.RGBA64:
		dst := *src
		dst.Pix = slices.Clone(src.Pix)
		return &dst
	case *image.NRGBA:
		dst := *src
		dst.Pix = slices.Clone(src.Pix)
		return &dst
	case *NRGB:
		dst := *src
		dst.Pix = slices.Clone(src.Pix)
		return &dst
	case *image.NRGBA64:
		dst := *src
		dst.Pix = slices.Clone(src.Pix)
		return &dst
	case *image.Gray:
		dst := *src
		dst.Pix = slices.Clone(src.Pix)
		return &dst
	case *image.Gray16:
		dst := *src
		dst.Pix = slices.Clone(src.Pix)
		return &dst
	case *image.Alpha:
		dst := *src
		dst.Pix = slices.Clone(src.Pix)
		return &dst
	case *image.Alpha16:
		dst := *src
		dst.Pix = slices.Clone(src.Pix)
		return &dst
	case *image.CMYK:
		dst := *src
		dst.Pix = slices.Clone(src.Pix)
		return &dst
	case *image.Paletted:
		dst := *src
		dst.Pix = slices.Clone(src.Pix)
		dst.Palette = slices.Clone(src.Palette)
		return &dst
	case *image.YCbCr:
		dst := *src
		dst.Y = slices.Clone(src.Y)
		dst.Cb = slices.Clone(src.Cb)
		dst.Cr = slices.Clone(src.Cr)
		return &dst
	case *image.NYCbCrA:
		dst := *src
		dst.Y = slices.Clone(src.Y)
		dst.Cb = slices.Clone(src.Cb)
		dst.Cr = slices.Clone(src.Cr)
		dst.A = slices.Clone(src.A)
		return &dst
	// For any other image type, fall back to a generic copy.
	// This creates an NRGBA image, which may not be the original type,
	// but ensures the image data is preserved.
	default:
		b := src.Bounds()
		dst := image.NewNRGBA64(b)
		draw.Draw(dst, b, src, b.Min, draw.Src)
		return dst
	}
}

// Ensure image has origin at (0, 0). Note that this destroys the original
// image and returns a new image with the same data, but origin shifted.
func NormalizeOrigin(src image.Image) image.Image {
	r := src.Bounds()
	if r.Min.X == 0 && r.Min.Y == 0 {
		return src
	}
	r = image.Rect(0, 0, r.Dx(), r.Dy())
	switch src := src.(type) {
	case *image.RGBA:
		dst := *src
		*src = image.RGBA{}
		dst.Rect = r
		return &dst
	case *image.RGBA64:
		dst := *src
		*src = image.RGBA64{}
		dst.Rect = r
		return &dst
	case *image.NRGBA:
		dst := *src
		*src = image.NRGBA{}
		dst.Rect = r
		return &dst
	case *NRGB:
		dst := *src
		*src = NRGB{}
		dst.Rect = r
		return &dst
	case *image.NRGBA64:
		dst := *src
		*src = image.NRGBA64{}
		dst.Rect = r
		return &dst
	case *image.Gray:
		dst := *src
		*src = image.Gray{}
		dst.Rect = r
		return &dst
	case *image.Gray16:
		dst := *src
		*src = image.Gray16{}
		dst.Rect = r
		return &dst
	case *image.Alpha:
		dst := *src
		*src = image.Alpha{}
		dst.Rect = r
		return &dst
	case *image.Alpha16:
		dst := *src
		*src = image.Alpha16{}
		dst.Rect = r
		return &dst
	case *image.CMYK:
		dst := *src
		*src = image.CMYK{}
		dst.Rect = r
		return &dst
	case *image.Paletted:
		dst := *src
		*src = image.Paletted{}
		dst.Rect = r
		return &dst
	case *image.YCbCr:
		dst := *src
		*src = image.YCbCr{}
		dst.Rect = r
		return &dst
	case *image.NYCbCrA:
		dst := *src
		*src = image.NYCbCrA{}
		dst.Rect = r
		return &dst
	// For any other image type, fall back to a generic copy.
	// This creates an NRGBA image, which may not be the original type,
	// but ensures the image data is preserved.
	default:
		b := src.Bounds()
		dst := image.NewNRGBA64(b)
		draw.Draw(dst, b, src, b.Min, draw.Src)
		dst.Rect = r
		return dst
	}
}

// Anchor is the anchor point for image alignment.
type Anchor int

// Anchor point positions.
const (
	Center Anchor = iota
	TopLeft
	Top
	TopRight
	Left
	Right
	BottomLeft
	Bottom
	BottomRight
)

func anchorPt(b image.Rectangle, w, h int, anchor Anchor) image.Point {
	var x, y int
	switch anchor {
	case TopLeft:
		x = b.Min.X
		y = b.Min.Y
	case Top:
		x = b.Min.X + (b.Dx()-w)/2
		y = b.Min.Y
	case TopRight:
		x = b.Max.X - w
		y = b.Min.Y
	case Left:
		x = b.Min.X
		y = b.Min.Y + (b.Dy()-h)/2
	case Right:
		x = b.Max.X - w
		y = b.Min.Y + (b.Dy()-h)/2
	case BottomLeft:
		x = b.Min.X
		y = b.Max.Y - h
	case Bottom:
		x = b.Min.X + (b.Dx()-w)/2
		y = b.Max.Y - h
	case BottomRight:
		x = b.Max.X - w
		y = b.Max.Y - h
	default:
		x = b.Min.X + (b.Dx()-w)/2
		y = b.Min.Y + (b.Dy()-h)/2
	}
	return image.Pt(x, y)
}

// Crop cuts out a rectangular region with the specified bounds
// from the image and returns the cropped image.
func Crop(img image.Image, rect image.Rectangle) *image.NRGBA {
	r := rect.Intersect(img.Bounds()).Sub(img.Bounds().Min)
	if r.Empty() {
		return &image.NRGBA{}
	}
	if r.Eq(img.Bounds().Sub(img.Bounds().Min)) {
		return Clone(img)
	}

	src := nrgba.NewNRGBAScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	rowSize := r.Dx() * 4
	if err := run_in_parallel_over_range(0, func(start, limit int) {
		for y := start; y < limit; y++ {
			i := (y - r.Min.Y) * dst.Stride
			src.Scan(r.Min.X, y, r.Max.X, y+1, dst.Pix[i:i+rowSize])
		}
	}, r.Min.Y, r.Max.Y); err != nil {
		panic(err)
	}
	return dst
}

// CropAnchor cuts out a rectangular region with the specified size
// from the image using the specified anchor point and returns the cropped image.
func CropAnchor(img image.Image, width, height int, anchor Anchor) *image.NRGBA {
	srcBounds := img.Bounds()
	pt := anchorPt(srcBounds, width, height, anchor)
	r := image.Rect(0, 0, width, height).Add(pt)
	b := srcBounds.Intersect(r)
	return Crop(img, b)
}

// CropCenter cuts out a rectangular region with the specified size
// from the center of the image and returns the cropped image.
func CropCenter(img image.Image, width, height int) *image.NRGBA {
	return CropAnchor(img, width, height, Center)
}

// Paste pastes the img image to the background image at the specified position and returns the combined image.
func Paste(background, img image.Image, pos image.Point) *image.NRGBA {
	dst := Clone(background)
	pos = pos.Sub(background.Bounds().Min)
	pasteRect := image.Rectangle{Min: pos, Max: pos.Add(img.Bounds().Size())}
	interRect := pasteRect.Intersect(dst.Bounds())
	if interRect.Empty() {
		return dst
	}
	if interRect.Eq(dst.Bounds()) {
		return Clone(img)
	}

	src := nrgba.NewNRGBAScanner(img)
	if err := run_in_parallel_over_range(0, func(start, limit int) {
		for y := start; y < limit; y++ {
			x1 := interRect.Min.X - pasteRect.Min.X
			x2 := interRect.Max.X - pasteRect.Min.X
			y1 := y - pasteRect.Min.Y
			y2 := y1 + 1
			i1 := y*dst.Stride + interRect.Min.X*4
			i2 := i1 + interRect.Dx()*4
			src.Scan(x1, y1, x2, y2, dst.Pix[i1:i2])
		}
	}, interRect.Min.Y, interRect.Max.Y); err != nil {
		panic(err)
	}
	return dst
}

// PasteCenter pastes the img image to the center of the background image and returns the combined image.
func PasteCenter(background, img image.Image) *image.NRGBA {
	bgBounds := background.Bounds()
	bgW := bgBounds.Dx()
	bgH := bgBounds.Dy()
	bgMinX := bgBounds.Min.X
	bgMinY := bgBounds.Min.Y

	centerX := bgMinX + bgW/2
	centerY := bgMinY + bgH/2

	x0 := centerX - img.Bounds().Dx()/2
	y0 := centerY - img.Bounds().Dy()/2

	return Paste(background, img, image.Pt(x0, y0))
}

// Overlay draws the img image over the background image at given position
// and returns the combined image. Opacity parameter is the opacity of the img
// image layer, used to compose the images, it must be from 0.0 to 1.0.
//
// Examples:
//
//	// Draw spriteImage over backgroundImage at the given position (x=50, y=50).
//	dstImage := imaging.Overlay(backgroundImage, spriteImage, image.Pt(50, 50), 1.0)
//
//	// Blend two opaque images of the same size.
//	dstImage := imaging.Overlay(imageOne, imageTwo, image.Pt(0, 0), 0.5)
func Overlay(background, img image.Image, pos image.Point, opacity float64) *image.NRGBA {
	opacity = math.Min(math.Max(opacity, 0.0), 1.0) // Ensure 0.0 <= opacity <= 1.0.
	dst := Clone(background)
	pos = pos.Sub(background.Bounds().Min)
	pasteRect := image.Rectangle{Min: pos, Max: pos.Add(img.Bounds().Size())}
	interRect := pasteRect.Intersect(dst.Bounds())
	if interRect.Empty() {
		return dst
	}
	src := nrgba.NewNRGBAScanner(img)
	if err := run_in_parallel_over_range(0, func(start, limit int) {
		scanLine := make([]uint8, interRect.Dx()*4)
		for y := start; y < limit; y++ {
			x1 := interRect.Min.X - pasteRect.Min.X
			x2 := interRect.Max.X - pasteRect.Min.X
			y1 := y - pasteRect.Min.Y
			y2 := y1 + 1
			src.Scan(x1, y1, x2, y2, scanLine)
			i := y*dst.Stride + interRect.Min.X*4
			j := 0
			for x := interRect.Min.X; x < interRect.Max.X; x++ {
				d := dst.Pix[i : i+4 : i+4]
				r1 := float64(d[0])
				g1 := float64(d[1])
				b1 := float64(d[2])
				a1 := float64(d[3])

				s := scanLine[j : j+4 : j+4]
				r2 := float64(s[0])
				g2 := float64(s[1])
				b2 := float64(s[2])
				a2 := float64(s[3])

				coef2 := opacity * a2 / 255
				coef1 := (1 - coef2) * a1 / 255
				coefSum := coef1 + coef2
				coef1 /= coefSum
				coef2 /= coefSum

				d[0] = uint8(r1*coef1 + r2*coef2)
				d[1] = uint8(g1*coef1 + g2*coef2)
				d[2] = uint8(b1*coef1 + b2*coef2)
				d[3] = uint8(math.Min(a1+a2*opacity*(255-a1)/255, 255))

				i += 4
				j += 4
			}
		}
	}, interRect.Min.Y, interRect.Max.Y); err != nil {
		panic(err)
	}
	return dst
}

// OverlayCenter overlays the img image to the center of the background image and
// returns the combined image. Opacity parameter is the opacity of the img
// image layer, used to compose the images, it must be from 0.0 to 1.0.
func OverlayCenter(background, img image.Image, opacity float64) *image.NRGBA {
	bgBounds := background.Bounds()
	bgW := bgBounds.Dx()
	bgH := bgBounds.Dy()
	bgMinX := bgBounds.Min.X
	bgMinY := bgBounds.Min.Y

	centerX := bgMinX + bgW/2
	centerY := bgMinY + bgH/2

	x0 := centerX - img.Bounds().Dx()/2
	y0 := centerY - img.Bounds().Dy()/2

	return Overlay(background, img, image.Point{x0, y0}, opacity)
}

// Paste the image onto the specified background color.
func PasteOntoBackground(img image.Image, bg color.Color) image.Image {
	if IsOpaque(img) {
		return img
	}
	_, _, _, a := bg.RGBA()
	bg_is_opaque := a == 0xffff
	var base draw.Image
	if bg_is_opaque {
		// use premult as its faster and will be converted to NRGB anyway
		base = image.NewRGBA(img.Bounds())
	} else {
		base = image.NewNRGBA(img.Bounds())
	}
	bgi := image.NewUniform(bg)
	draw.Draw(base, base.Bounds(), bgi, image.Point{}, draw.Src)
	draw.Draw(base, base.Bounds(), img, img.Bounds().Min, draw.Over)
	if bg_is_opaque {
		return AsNRGB(base)
	}
	return base
}

// Return contiguous non-premultiplied RGB pixel data for this image with 8 bits per channel
func AsRGBData8(img image.Image) (pix []uint8) {
	b := img.Bounds()
	n := AsNRGB(img)
	if n.Stride == b.Dx()*3 {
		return n.Pix
	}
	pix = make([]uint8, 0, b.Dx()*b.Dy()*3)
	for y := range b.Dy() {
		pix = append(pix, n.Pix[y*n.Stride:y*(n.Stride+1)]...)
	}
	return pix
}

// Return contiguous non-premultiplied RGBA pixel data for this image with 8 bits per channel
func AsRGBAData8(img image.Image) (pix []uint8) {
	b := img.Bounds()
	n := AsNRGBA(img)
	if n.Stride == b.Dx()*4 {
		return n.Pix
	}
	pix = make([]uint8, 0, b.Dx()*b.Dy()*4)
	for y := range b.Dy() {
		pix = append(pix, n.Pix[y*n.Stride:y*(n.Stride+1)]...)
	}
	return pix
}
