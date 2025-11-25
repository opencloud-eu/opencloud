package imaging

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/kovidgoyal/go-parallel"
	"github.com/kovidgoyal/imaging/nrgb"
	"github.com/kovidgoyal/imaging/prism/meta/icc"
)

var _ = fmt.Print

// unpremultiply and convert to normalized float
func unpremultiply8(r_, a_ uint8) float64 {
	r, a := uint16(r_), uint16(a_)
	return float64((r*math.MaxUint8)/a) / math.MaxUint8
}

// unpremultiply and convert to normalized float
func unpremultiply(r, a uint32) float64 {
	return float64((r*math.MaxUint16)/a) / math.MaxUint16
}

func f8(x uint8) float64    { return float64(x) / math.MaxUint8 }
func f8i(x float64) uint8   { return uint8(x * math.MaxUint8) }
func f16(x uint16) float64  { return float64(x) / math.MaxUint16 }
func f16i(x float64) uint16 { return uint16(x * math.MaxUint16) }

func convert(tr *icc.Pipeline, image_any image.Image) (ans image.Image, err error) {
	t := tr.Transform
	b := image_any.Bounds()
	width, height := b.Dx(), b.Dy()
	ans = image_any
	var f func(start, limit int)
	switch img := image_any.(type) {
	case *NRGB:
		f = func(start, limit int) {
			for y := start; y < limit; y++ {
				row := img.Pix[img.Stride*y:]
				_ = row[3*(width-1)]
				for range width {
					r := row[0:3:3]
					fr, fg, fb := t(f8(r[0]), f8(r[1]), f8(r[2]))
					r[0], r[1], r[2] = f8i(fr), f8i(fg), f8i(fb)
					row = row[3:]
				}
			}
		}
	case *image.NRGBA:
		f = func(start, limit int) {
			for y := start; y < limit; y++ {
				row := img.Pix[img.Stride*y:]
				_ = row[4*(width-1)]
				for range width {
					r := row[0:3:3]
					fr, fg, fb := t(f8(r[0]), f8(r[1]), f8(r[2]))
					r[0], r[1], r[2] = f8i(fr), f8i(fg), f8i(fb)
					row = row[4:]
				}
			}
		}
	case *image.NRGBA64:
		f = func(start, limit int) {
			for y := start; y < limit; y++ {
				row := img.Pix[img.Stride*y:]
				_ = row[8*(width-1)]
				for range width {
					s := row[0:8:8]
					fr := f16(uint16(s[0])<<8 | uint16(s[1]))
					fg := f16(uint16(s[2])<<8 | uint16(s[3]))
					fb := f16(uint16(s[4])<<8 | uint16(s[5]))
					fr, fg, fb = t(fr, fg, fb)
					r, g, b := f16i(fr), f16i(fg), f16i(fb)
					s[0], s[1] = uint8(r>>8), uint8(r)
					s[2], s[3] = uint8(g>>8), uint8(g)
					s[4], s[5] = uint8(b>>8), uint8(b)
					row = row[8:]
				}
			}
		}
	case *image.RGBA:
		d := image.NewNRGBA(b)
		ans = d
		f = func(start, limit int) {
			for y := start; y < limit; y++ {
				row := img.Pix[img.Stride*y:]
				drow := d.Pix[d.Stride*y:]
				_ = row[4*(width-1)]
				_ = drow[4*(width-1)]
				for range width {
					r := row[0:4:4]
					dr := drow[0:4:4]
					dr[3] = r[3]
					if a := row[3]; a != 0 {
						fr, fg, fb := t(unpremultiply8(r[0], a), unpremultiply8(r[1], a), unpremultiply8(r[2], a))
						dr[0], dr[1], dr[2] = f8i(fr), f8i(fg), f8i(fb)
					}
					row = row[4:]
					drow = drow[4:]
				}
			}
		}
	case *image.RGBA64:
		d := image.NewNRGBA64(b)
		ans = d
		f = func(start, limit int) {
			for y := start; y < limit; y++ {
				row := img.Pix[img.Stride*y:]
				drow := d.Pix[d.Stride*y:]
				_ = row[8*(width-1)]
				_ = drow[8*(width-1)]
				for range width {
					s, dr := row[0:8:8], drow[0:8:8]
					dr[6], dr[7] = s[6], s[7]
					a := uint32(s[6])<<8 | uint32(s[7])
					if a != 0 {
						fr := unpremultiply((uint32(s[0])<<8 | uint32(s[1])), a)
						fg := unpremultiply((uint32(s[2])<<8 | uint32(s[3])), a)
						fb := unpremultiply((uint32(s[4])<<8 | uint32(s[5])), a)
						fr, fg, fb = t(fr, fg, fb)
						r, g, b := f16i(fr), f16i(fg), f16i(fb)
						dr[0], dr[1] = uint8(r>>8), uint8(r)
						dr[2], dr[3] = uint8(g>>8), uint8(g)
						dr[4], dr[5] = uint8(b>>8), uint8(b)
					}
					row = row[8:]
					drow = drow[8:]
				}
			}
		}
	case *image.Paletted:
		for i, c := range img.Palette {
			r, g, b, a := c.RGBA()
			if a != 0 {
				fr, fg, fb := unpremultiply(r, a), unpremultiply(g, a), unpremultiply(b, a)
				fr, fg, fb = t(fr, fg, fb)
				img.Palette[i] = color.NRGBA64{R: f16i(fr), G: f16i(fg), B: f16i(fb), A: uint16(a)}
			}
		}
		return
	case *image.CMYK:
		g := tr.TransformGeneral
		d := nrgb.NewNRGB(b)
		ans = d
		f = func(start, limit int) {
			var inp, outp [4]float64
			i, o := inp[:], outp[:]
			for y := start; y < limit; y++ {
				row := img.Pix[img.Stride*y:]
				drow := d.Pix[d.Stride*y:]
				_ = row[4*(width-1)]
				_ = drow[3*(width-1)]
				for range width {
					r := row[0:4:4]
					inp[0], inp[1], inp[2], inp[3] = f8(r[0]), f8(r[1]), f8(r[2]), f8(r[3])
					g(o, i)
					r = drow[0:3:3]
					r[0], r[1], r[2] = f8i(outp[0]), f8i(outp[1]), f8i(outp[2])
					row = row[4:]
					drow = drow[3:]
				}
			}
		}
	case *image.YCbCr:
		d := nrgb.NewNRGB(b)
		ans = d
		f = func(start, limit int) {
			for y := start; y < limit; y++ {
				ybase := y * img.YStride
				row := d.Pix[d.Stride*y:]
				yy := y + b.Min.Y
				for x := b.Min.X; x < b.Max.X; x++ {
					iy := ybase + (x - b.Min.X)
					ic := img.COffset(x, yy)
					// We use this rather than color.YCbCrToRGB for greater accuracy
					r, g, bb, _ := color.YCbCr{img.Y[iy], img.Cb[ic], img.Cr[ic]}.RGBA()
					fr, fg, fb := t(f16(uint16(r)), f16(uint16(g)), f16(uint16(bb)))
					rr := row[0:3:3]
					rr[0], rr[1], rr[2] = f8i(fr), f8i(fg), f8i(fb)
					row = row[3:]
				}
			}
		}
	case *image.NYCbCrA:
		d := image.NewNRGBA(b)
		ans = d
		f = func(start, limit int) {
			for y := start; y < limit; y++ {
				ybase := y * img.YStride
				row := d.Pix[d.Stride*y:]
				yy := y + b.Min.Y
				for x := b.Min.X; x < b.Max.X; x++ {
					rr := row[0:4:4]
					rr[3] = img.A[img.AOffset(x, yy)]
					if rr[3] != 0 {
						iy := ybase + (x - b.Min.X)
						ic := img.COffset(x, yy)
						// We use this rather than color.YCbCrToRGB for greater accuracy
						r, g, bb, _ := color.YCbCr{img.Y[iy], img.Cb[ic], img.Cr[ic]}.RGBA()
						fr, fg, fb := t(f16(uint16(r)), f16(uint16(g)), f16(uint16(bb)))
						rr[0], rr[1], rr[2] = f8i(fr), f8i(fg), f8i(fb)
						row = row[4:]
					}
				}
			}
		}
	case draw.Image:
		f = func(start, limit int) {
			for y := b.Min.Y + start; y < b.Min.Y+limit; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					r16, g16, b16, a16 := img.At(x, y).RGBA()
					if a16 != 0 {
						fr, fg, fb := unpremultiply(r16, a16), unpremultiply(g16, a16), unpremultiply(b16, a16)
						fr, fg, fb = t(fr, fg, fb)
						img.Set(x, y, &color.NRGBA64{R: f16i(fr), G: f16i(fg), B: f16i(fb)})
					}
				}
			}
		}
	default:
		d := image.NewNRGBA64(b)
		ans = d
		f = func(start, limit int) {
			for y := start; y < limit; y++ {
				row := d.Pix[d.Stride*y:]
				for x := range width {
					r16, g16, b16, a16 := img.At(x+b.Min.X, y+b.Min.Y).RGBA()
					if a16 != 0 {
						fr, fg, fb := unpremultiply(r16, a16), unpremultiply(g16, a16), unpremultiply(b16, a16)
						fr, fg, fb = t(fr, fg, fb)
						r, g, b := f16i(fr), f16i(fg), f16i(fb)
						s := row[0:8:8]
						row = row[8:]
						s[0], s[1] = uint8(r>>8), uint8(r)
						s[2], s[3] = uint8(g>>8), uint8(g)
						s[4], s[5] = uint8(b>>8), uint8(b)
						s[6] = uint8(a16 >> 8)
						s[7] = uint8(a16)
					}
				}
			}
		}
	}
	err = parallel.Run_in_parallel_over_range(0, f, 0, height)
	return
}
