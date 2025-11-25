package nrgba

import (
	"image"
	"image/color"

	"github.com/kovidgoyal/imaging/nrgb"
	"github.com/kovidgoyal/imaging/types"
)

type scanner struct {
	image   image.Image
	w, h    int
	palette []color.NRGBA
}

func (s scanner) Bytes_per_channel() int  { return 1 }
func (s scanner) Num_of_channels() int    { return 4 }
func (s scanner) Bounds() image.Rectangle { return s.image.Bounds() }
func (s scanner) NewImage(r image.Rectangle) image.Image {
	return image.NewNRGBA(r)
}

func newScanner(img image.Image) *scanner {
	s := &scanner{
		image: img,
		w:     img.Bounds().Dx(),
		h:     img.Bounds().Dy(),
	}
	if img, ok := img.(*image.Paletted); ok {
		s.palette = make([]color.NRGBA, max(256, len(img.Palette)))
		for i := 0; i < len(img.Palette); i++ {
			s.palette[i] = color.NRGBAModel.Convert(img.Palette[i]).(color.NRGBA)
		}
	}
	return s
}

func reverse4(pix []uint8) {
	if len(pix) <= 4 {
		return
	}
	i := 0
	j := len(pix) - 4
	for i < j {
		pi := pix[i : i+4 : i+4]
		pj := pix[j : j+4 : j+4]
		pi[0], pj[0] = pj[0], pi[0]
		pi[1], pj[1] = pj[1], pi[1]
		pi[2], pj[2] = pj[2], pi[2]
		pi[3], pj[3] = pj[3], pi[3]
		i += 4
		j -= 4
	}
}

func (s *scanner) ReverseRow(img image.Image, row int) {
	d := img.(*image.NRGBA)
	pos := row * d.Stride
	r := d.Pix[pos : pos+d.Stride : pos+d.Stride]
	reverse4(r)
}

func (s *scanner) ScanRow(x1, y1, x2, y2 int, img image.Image, row int) {
	d := img.(*image.NRGBA)
	pos := row * d.Stride
	r := d.Pix[pos : pos+d.Stride : pos+d.Stride]
	s.Scan(x1, y1, x2, y2, r)
}

// scan scans the given rectangular region of the image into dst.
func (s *scanner) Scan(x1, y1, x2, y2 int, dst []uint8) {
	_ = dst[4*(x2-x1)*(y2-y1)-1]
	switch img := s.image.(type) {
	case *nrgb.Image:
		if x2 == x1+1 {
			j := 0
			i := y1*img.Stride + x1*3
			for y := y1; y < y2; y++ {
				d := dst[j : j+4 : j+4]
				s := img.Pix[i : i+3 : i+3]
				d[0] = s[0]
				d[1] = s[1]
				d[2] = s[2]
				d[3] = 255
				j += 4
				i += img.Stride
			}
		} else {
			d := dst
			for y := y1; y < y2; y++ {
				s := img.Pix[y*img.Stride+x1*3:]
				for range x2 - x1 {
					d[0], d[1], d[2], d[3] = s[0], s[1], s[2], 255
					d, s = d[4:], s[3:]
				}
			}
		}
	case *image.NRGBA:
		size := (x2 - x1) * 4
		j := 0
		i := y1*img.Stride + x1*4
		if size == 4 {
			for y := y1; y < y2; y++ {
				d := dst[j : j+4 : j+4]
				s := img.Pix[i : i+4 : i+4]
				d[0] = s[0]
				d[1] = s[1]
				d[2] = s[2]
				d[3] = s[3]
				j += size
				i += img.Stride
			}
		} else {
			for y := y1; y < y2; y++ {
				copy(dst[j:j+size], img.Pix[i:i+size])
				j += size
				i += img.Stride
			}
		}

	case *image.NRGBA64:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*8
			for x := x1; x < x2; x++ {
				s := img.Pix[i : i+8 : i+8]
				d := dst[j : j+4 : j+4]
				d[0] = s[0]
				d[1] = s[2]
				d[2] = s[4]
				d[3] = s[6]
				j += 4
				i += 8
			}
		}

	case *image.RGBA:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*4
			for x := x1; x < x2; x++ {
				d := dst[j : j+4 : j+4]
				a := img.Pix[i+3]
				switch a {
				case 0:
					d[0] = 0
					d[1] = 0
					d[2] = 0
					d[3] = a
				case 0xff:
					s := img.Pix[i : i+4 : i+4]
					d[0] = s[0]
					d[1] = s[1]
					d[2] = s[2]
					d[3] = a
				default:
					s := img.Pix[i : i+4 : i+4]
					r16 := uint16(s[0])
					g16 := uint16(s[1])
					b16 := uint16(s[2])
					a16 := uint16(a)
					d[0] = uint8(r16 * 0xff / a16)
					d[1] = uint8(g16 * 0xff / a16)
					d[2] = uint8(b16 * 0xff / a16)
					d[3] = a
				}
				j += 4
				i += 4
			}
		}

	case *image.RGBA64:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*8
			for x := x1; x < x2; x++ {
				s := img.Pix[i : i+8 : i+8]
				d := dst[j : j+4 : j+4]
				a := s[6]
				switch a {
				case 0:
					d[0] = 0
					d[1] = 0
					d[2] = 0
				case 0xff:
					d[0] = s[0]
					d[1] = s[2]
					d[2] = s[4]
				default:
					r32 := uint32(s[0])<<8 | uint32(s[1])
					g32 := uint32(s[2])<<8 | uint32(s[3])
					b32 := uint32(s[4])<<8 | uint32(s[5])
					a32 := uint32(s[6])<<8 | uint32(s[7])
					d[0] = uint8((r32 * 0xffff / a32) >> 8)
					d[1] = uint8((g32 * 0xffff / a32) >> 8)
					d[2] = uint8((b32 * 0xffff / a32) >> 8)
				}
				d[3] = a
				j += 4
				i += 8
			}
		}

	case *image.Gray:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1
			for x := x1; x < x2; x++ {
				c := img.Pix[i]
				d := dst[j : j+4 : j+4]
				d[0] = c
				d[1] = c
				d[2] = c
				d[3] = 0xff
				j += 4
				i++
			}
		}

	case *image.Gray16:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1*2
			for x := x1; x < x2; x++ {
				c := img.Pix[i]
				d := dst[j : j+4 : j+4]
				d[0] = c
				d[1] = c
				d[2] = c
				d[3] = 0xff
				j += 4
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
					d := dst[0:4:4]
					d[0], d[1], d[2] = color.YCbCrToRGB(Y[x], Cb[x], Cr[x])
					d[3] = 255
					dst = dst[4:]
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

					d := dst[j : j+4 : j+4]
					d[0], d[1], d[2] = color.YCbCrToRGB(img.Y[iy], img.Cb[ic], img.Cr[ic])
					d[3] = 0xff

					iy++
					j += 4
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
					d := dst[0:4:4]
					d[0], d[1], d[2] = color.YCbCrToRGB(Y[x], Cb[x], Cr[x])
					d[3] = A[x]
					dst = dst[4:]
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

					d := dst[j : j+4 : j+4]
					d[0], d[1], d[2] = color.YCbCrToRGB(img.Y[iy], img.Cb[ic], img.Cr[ic])
					d[3] = img.A[ia]

					iy++
					j += 4
				}
			}
		}

	case *image.Paletted:
		j := 0
		for y := y1; y < y2; y++ {
			i := y*img.Stride + x1
			for x := x1; x < x2; x++ {
				c := s.palette[img.Pix[i]]
				d := dst[j : j+4 : j+4]
				d[0] = c.R
				d[1] = c.G
				d[2] = c.B
				d[3] = c.A
				j += 4
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
				d := dst[j : j+4 : j+4]
				switch a16 {
				case 0xffff:
					d[0] = uint8(r16 >> 8)
					d[1] = uint8(g16 >> 8)
					d[2] = uint8(b16 >> 8)
					d[3] = 0xff
				case 0:
					d[0] = 0
					d[1] = 0
					d[2] = 0
					d[3] = 0
				default:
					d[0] = uint8(((r16 * 0xffff) / a16) >> 8)
					d[1] = uint8(((g16 * 0xffff) / a16) >> 8)
					d[2] = uint8(((b16 * 0xffff) / a16) >> 8)
					d[3] = uint8(a16 >> 8)
				}
				j += 4
			}
		}
	}
}

func NewNRGBAScanner(source_image image.Image) types.Scanner {
	return newScanner(source_image)
}
