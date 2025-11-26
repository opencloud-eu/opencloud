package nrgb

import (
	"fmt"
	"image"
	"image/color"

	"github.com/kovidgoyal/imaging/types"
)

var _ = fmt.Print

type Color struct {
	R, G, B uint8
}

func (c Color) AsSharp() string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

func (c Color) RGBA() (r, g, b, a uint32) {
	r = uint32(c.R)
	r |= r << 8
	g = uint32(c.G)
	g |= g << 8
	b = uint32(c.B)
	b |= b << 8
	a = 65535 // (255 << 8 | 255)
	return
}

// Image is an in-memory image whose At method returns Color values.
type Image struct {
	// Pix holds the image's pixels, in R, G, B order. The pixel at
	// (x, y) starts at Pix[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*3].
	Pix []uint8
	// Stride is the Pix stride (in bytes) between vertically adjacent pixels.
	Stride int
	// Rect is the image's bounds.
	Rect image.Rectangle
}

func nrgbModel(c color.Color) color.Color {
	switch q := c.(type) {
	case Color:
		return c
	case color.NRGBA:
		return Color{q.R, q.G, q.B}
	case color.NRGBA64:
		return Color{uint8(q.R >> 8), uint8(q.G >> 8), uint8(q.B >> 8)}
	}
	r, g, b, a := c.RGBA()
	switch a {
	case 0xffff:
		return Color{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}
	case 0:
		return Color{0, 0, 0}
	default:
		// Since Color.RGBA returns an alpha-premultiplied color, we should have r <= a && g <= a && b <= a.
		r = (r * 0xffff) / a
		g = (g * 0xffff) / a
		b = (b * 0xffff) / a
		return Color{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}
	}
}

var Model color.Model = color.ModelFunc(nrgbModel)

func (p *Image) ColorModel() color.Model { return Model }

func (p *Image) Bounds() image.Rectangle { return p.Rect }

func (p *Image) At(x, y int) color.Color {
	return p.NRGBAt(x, y)
}

func (p *Image) NRGBAt(x, y int) Color {
	if !(image.Point{x, y}.In(p.Rect)) {
		return Color{}
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+3 : i+3] // Small cap improves performance, see https://golang.org/issue/27857
	return Color{s[0], s[1], s[2]}
}

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).
func (p *Image) PixOffset(x, y int) int {
	return (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)*3
}

func (p *Image) Set(x, y int, c color.Color) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+3 : i+3] // Small cap improves performance, see https://golang.org/issue/27857
	q := nrgbModel(c).(Color)
	s[0], s[1], s[2] = q.R, q.G, q.B
}

func (p *Image) SetRGBA64(x, y int, c color.RGBA64) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}
	r, g, b, a := uint32(c.R), uint32(c.G), uint32(c.B), uint32(c.A)
	if (a != 0) && (a != 0xffff) {
		r = (r * 0xffff) / a
		g = (g * 0xffff) / a
		b = (b * 0xffff) / a
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+3 : i+3] // Small cap improves performance, see https://golang.org/issue/27857
	s[0] = uint8(r >> 8)
	s[1] = uint8(g >> 8)
	s[2] = uint8(b >> 8)
}

func (p *Image) SetNRGBA(x, y int, c color.NRGBA) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+3 : i+3] // Small cap improves performance, see https://golang.org/issue/27857
	s[0] = c.R
	s[1] = c.G
	s[2] = c.B
}

// SubImage returns an image representing the portion of the image p visible
// through r. The returned value shares pixels with the original image.
func (p *Image) SubImage(r image.Rectangle) image.Image {
	r = r.Intersect(p.Rect)
	// If r1 and r2 are Rectangles, r1.Intersect(r2) is not guaranteed to be inside
	// either r1 or r2 if the intersection is empty. Without explicitly checking for
	// this, the Pix[i:] expression below can panic.
	if r.Empty() {
		return &Image{}
	}
	i := p.PixOffset(r.Min.X, r.Min.Y)
	return &Image{
		Pix:    p.Pix[i:],
		Stride: p.Stride,
		Rect:   r,
	}
}

// Opaque scans the entire image and reports whether it is fully opaque.
func (p *Image) Opaque() bool { return true }

type scanner_rgb struct {
	image            image.Image
	w, h             int
	palette          []Color
	opaque_base      []float64
	opaque_base_uint []uint8
}

func (s scanner_rgb) Bytes_per_channel() int                 { return 1 }
func (s scanner_rgb) Num_of_channels() int                   { return 3 }
func (s scanner_rgb) Bounds() image.Rectangle                { return s.image.Bounds() }
func (s scanner_rgb) NewImage(r image.Rectangle) image.Image { return NewNRGB(r) }

func blend(dest []uint8, base []float64, r, g, b, a uint8) {
	alpha := float64(a) / 255.0
	dest[0] = uint8(alpha*float64(r) + (1.0-alpha)*base[0])
	dest[1] = uint8(alpha*float64(g) + (1.0-alpha)*base[1])
	dest[2] = uint8(alpha*float64(b) + (1.0-alpha)*base[2])
}

func reverse3(pix []uint8) {
	if len(pix) <= 3 {
		return
	}
	i := 0
	j := len(pix) - 3
	for i < j {
		pi := pix[i : i+3 : i+3]
		pj := pix[j : j+3 : j+3]
		pi[0], pj[0] = pj[0], pi[0]
		pi[1], pj[1] = pj[1], pi[1]
		pi[2], pj[2] = pj[2], pi[2]
		i += 3
		j -= 3
	}
}

func (s *scanner_rgb) ReverseRow(img image.Image, row int) {
	d := img.(*Image)
	pos := row * d.Stride
	r := d.Pix[pos : pos+d.Stride : pos+d.Stride]
	reverse3(r)
}

func (s *scanner_rgb) ScanRow(x1, y1, x2, y2 int, img image.Image, row int) {
	d := img.(*Image)
	pos := row * d.Stride
	r := d.Pix[pos : pos+d.Stride : pos+d.Stride]
	s.Scan(x1, y1, x2, y2, r)
}

func newScannerRGB(img image.Image, opaque_base Color) *scanner_rgb {
	s := &scanner_rgb{
		image: img, w: img.Bounds().Dx(), h: img.Bounds().Dy(),
		opaque_base:      []float64{float64(opaque_base.R), float64(opaque_base.G), float64(opaque_base.B)}[0:3:3],
		opaque_base_uint: []uint8{opaque_base.R, opaque_base.G, opaque_base.B}[0:3:3],
	}
	if img, ok := img.(*image.Paletted); ok {
		s.palette = make([]Color, max(256, len(img.Palette)))
		d := [3]uint8{0, 0, 0}
		ds := d[:]
		for i := 0; i < len(img.Palette); i++ {
			r, g, b, a := img.Palette[i].RGBA()
			switch a {
			case 0:
				s.palette[i] = opaque_base
			case 0xffff:
				s.palette[i] = Color{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8)}
			default:
				blend(ds, s.opaque_base, uint8((r*0xffff/a)>>8), uint8((g*0xffff/a)>>8), uint8((b*0xffff/a)>>8), uint8(a>>8))
				s.palette[i] = Color{R: d[0], G: d[1], B: d[2]}
			}
		}
	}
	return s
}

func (s *scanner_rgb) blend8(d []uint8, a uint8) {
	switch a {
	case 0:
		d[0] = s.opaque_base_uint[0]
		d[1] = s.opaque_base_uint[1]
		d[2] = s.opaque_base_uint[2]
	case 0xff:
	default:
		blend(d, s.opaque_base, d[0], d[1], d[2], a)
	}
}

// scan scans the given rectangular region of the image into dst.
func (s *scanner_rgb) Scan(x1, y1, x2, y2 int, dst []uint8) {
	_ = dst[3*(x2-x1)*(y2-y1)-1]
	switch img := s.image.(type) {
	case *image.NRGBA:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*4
			for x := x1; x < x2; x++ {
				blend(dst[j:j+3:j+3], s.opaque_base, img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3])
				j += 3
				i += 4
			}
		}

	case *image.NRGBA64:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*8
			for x := x1; x < x2; x++ {
				blend(dst[j:j+3:j+3], s.opaque_base, img.Pix[i], img.Pix[i+2], img.Pix[i+4], img.Pix[i+6])
				j += 3
				i += 8
			}
		}

	case *image.RGBA:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*4
			for x := x1; x < x2; x++ {
				d := dst[j : j+3 : j+3]
				a := img.Pix[i+3]
				switch a {
				case 0:
					d[0] = s.opaque_base_uint[0]
					d[1] = s.opaque_base_uint[1]
					d[2] = s.opaque_base_uint[2]
				case 0xff:
					s := img.Pix[i : i+3 : i+3]
					d[0] = s[0]
					d[1] = s[1]
					d[2] = s[2]
				default:
					r16 := uint16(img.Pix[i])
					g16 := uint16(img.Pix[i+1])
					b16 := uint16(img.Pix[i+2])
					a16 := uint16(a)
					blend(d, s.opaque_base, uint8(r16*0xff/a16), uint8(g16*0xff/a16), uint8(b16*0xff/a16), a)
				}
				j += 3
				i += 4
			}
		}

	case *image.RGBA64:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*8
			for x := x1; x < x2; x++ {
				src := img.Pix[i : i+8 : i+8]
				d := dst[j : j+3 : j+3]
				a := src[6]
				switch a {
				case 0:
					d[0] = s.opaque_base_uint[0]
					d[1] = s.opaque_base_uint[1]
					d[2] = s.opaque_base_uint[2]
				case 0xff:
					d[0] = src[0]
					d[1] = src[2]
					d[2] = src[4]
				default:
					r32 := uint32(src[0])<<8 | uint32(src[1])
					g32 := uint32(src[2])<<8 | uint32(src[3])
					b32 := uint32(src[4])<<8 | uint32(src[5])
					a32 := uint32(src[6])<<8 | uint32(src[7])
					blend(d, s.opaque_base, uint8((r32*0xffff/a32)>>8), uint8((g32*0xffff/a32)>>8), uint8((b32*0xffff/a32)>>8), a)
				}
				j += 3
				i += 8
			}
		}

	case *image.Gray:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1
			for x := x1; x < x2; x++ {
				c := img.Pix[i]
				d := dst[j : j+3 : j+3]
				d[0] = c
				d[1] = c
				d[2] = c
				j += 3
				i++
			}
		}

	case *image.Gray16:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*2
			for x := x1; x < x2; x++ {
				c := img.Pix[i]
				d := dst[j : j+3 : j+3]
				d[0] = c
				d[1] = c
				d[2] = c
				j += 3
				i += 2
			}
		}

	case *image.YCbCr:
		if img.SubsampleRatio == image.YCbCrSubsampleRatio444 {
			Y := img.Y[y1*img.YStride:]
			Cb := img.Cb[y1*img.CStride:]
			Cr := img.Cr[y1*img.CStride:]
			for range y2 - y1 {
				for x := x1; x < x2; x++ {
					d := dst[0:3:3]
					d[0], d[1], d[2] = color.YCbCrToRGB(Y[x], Cb[x], Cr[x])
					dst = dst[3:]
				}
				Y, Cb, Cr = Y[img.YStride:], Cb[img.CStride:], Cr[img.CStride:]
			}
		} else {
			j := 0
			x1 += img.Rect.Min.X
			x2 += img.Rect.Min.X
			y1 += img.Rect.Min.Y
			y2 += img.Rect.Min.Y

			hy := img.Rect.Min.Y / 2
			hx := img.Rect.Min.X / 2
			for y := y1; y < y2; y++ {
				iy := (y-img.Rect.Min.Y)*img.YStride + (x1 - img.Rect.Min.X)

				var yBase int
				switch img.SubsampleRatio {
				case image.YCbCrSubsampleRatio422:
					yBase = (y - img.Rect.Min.Y) * img.CStride
				case image.YCbCrSubsampleRatio420, image.YCbCrSubsampleRatio440:
					yBase = (y/2 - hy) * img.CStride
				}

				for x := x1; x < x2; x++ {
					var ic int
					switch img.SubsampleRatio {
					case image.YCbCrSubsampleRatio440:
						ic = yBase + (x - img.Rect.Min.X)
					case image.YCbCrSubsampleRatio422, image.YCbCrSubsampleRatio420:
						ic = yBase + (x/2 - hx)
					default:
						ic = img.COffset(x, y)
					}
					d := dst[j : j+3 : j+3]
					d[0], d[1], d[2] = color.YCbCrToRGB(img.Y[iy], img.Cb[ic], img.Cr[ic])
					iy++
					j += 3
				}
			}
		}
	case *image.NYCbCrA:
		if img.SubsampleRatio == image.YCbCrSubsampleRatio444 {
			Y := img.Y[y1*img.YStride:]
			A := img.A[y1*img.AStride:]
			Cb := img.Cb[y1*img.CStride:]
			Cr := img.Cr[y1*img.CStride:]
			for range y2 - y1 {
				for x := x1; x < x2; x++ {
					d := dst[0:3:3]
					d[0], d[1], d[2] = color.YCbCrToRGB(Y[x], Cb[x], Cr[x])
					s.blend8(d, A[x])
					dst = dst[3:]
				}
				Y, Cb, Cr = Y[img.YStride:], Cb[img.CStride:], Cr[img.CStride:]
				A = A[img.AStride:]
			}
		} else {
			j := 0
			x1 += img.Rect.Min.X
			x2 += img.Rect.Min.X
			y1 += img.Rect.Min.Y
			y2 += img.Rect.Min.Y

			hy := img.Rect.Min.Y / 2
			hx := img.Rect.Min.X / 2
			for y := y1; y < y2; y++ {
				iy := (y-img.Rect.Min.Y)*img.YStride + (x1 - img.Rect.Min.X)
				ia := (y-img.Rect.Min.Y)*img.AStride + (x1 - img.Rect.Min.X)

				var yBase int
				switch img.SubsampleRatio {
				case image.YCbCrSubsampleRatio422:
					yBase = (y - img.Rect.Min.Y) * img.CStride
				case image.YCbCrSubsampleRatio420, image.YCbCrSubsampleRatio440:
					yBase = (y/2 - hy) * img.CStride
				}

				for x := x1; x < x2; x++ {
					var ic int
					switch img.SubsampleRatio {
					case image.YCbCrSubsampleRatio440:
						ic = yBase + (x - img.Rect.Min.X)
					case image.YCbCrSubsampleRatio422, image.YCbCrSubsampleRatio420:
						ic = yBase + (x/2 - hx)
					default:
						ic = img.COffset(x, y)
					}
					d := dst[j : j+3 : j+3]
					d[0], d[1], d[2] = color.YCbCrToRGB(img.Y[iy], img.Cb[ic], img.Cr[ic])
					s.blend8(d, img.A[ia])
					iy++
					j += 3
				}
			}
		}

	case *image.Paletted:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1
			for x := x1; x < x2; x++ {
				c := s.palette[img.Pix[i]]
				d := dst[j : j+3 : j+3]
				d[0] = c.R
				d[1] = c.G
				d[2] = c.B
				j += 3
				i++
			}
		}

	default:
		j := 0
		b := s.image.Bounds()
		x1 += b.Min.X
		x2 += b.Min.X
		y1 += b.Min.Y
		y2 += b.Min.Y
		for y := y1; y < y2; y++ {
			for x := x1; x < x2; x++ {
				r16, g16, b16, a16 := s.image.At(x, y).RGBA()
				d := dst[j : j+3 : j+3]
				switch a16 {
				case 0xffff:
					d[0] = uint8(r16 >> 8)
					d[1] = uint8(g16 >> 8)
					d[2] = uint8(b16 >> 8)
				case 0:
					d[0] = s.opaque_base_uint[0]
					d[1] = s.opaque_base_uint[1]
					d[2] = s.opaque_base_uint[2]
				default:
					blend(d, s.opaque_base, uint8(((r16*0xffff)/a16)>>8), uint8(((g16*0xffff)/a16)>>8), uint8(((b16*0xffff)/a16)>>8), uint8(a16>>8))
				}
				j += 3
			}
		}
	}
}

func NewNRGB(r image.Rectangle) *Image {
	return &Image{
		Pix:    make([]uint8, 3*r.Dx()*r.Dy()),
		Stride: 3 * r.Dx(),
		Rect:   r,
	}
}

func NewNRGBWithContiguousRGBPixels(p []byte, left, top, width, height int) (*Image, error) {
	const bpp = 3
	if expected := bpp * width * height; expected != len(p) {
		return nil, fmt.Errorf("the image width and height dont match the size of the specified pixel data: width=%d height=%d sz=%d != %d", width, height, len(p), expected)
	}
	return &Image{
		Pix:    p,
		Stride: bpp * width,
		Rect:   image.Rectangle{image.Point{left, top}, image.Point{left + width, top + height}},
	}, nil
}

func NewNRGBScanner(source_image image.Image, opaque_base Color) types.Scanner {
	return newScannerRGB(source_image, opaque_base)
}
