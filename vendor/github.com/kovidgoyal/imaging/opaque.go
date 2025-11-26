package imaging

import (
	"bytes"
	"fmt"
	"image"
	"sync/atomic"

	"github.com/kovidgoyal/go-parallel"
	"github.com/kovidgoyal/imaging/nrgb"
)

var _ = fmt.Print

func is_opaque1(pix []uint8, w, h, stride int) bool {
	is_opaque := atomic.Bool{}
	is_opaque.Store(true)
	if err := parallel.Run_in_parallel_to_first_result(0, func(start, limit int, keep_going *atomic.Bool) bool {
		pix := pix[stride*start:]
		for range limit - start {
			p := pix[0:w:w]
			for _, x := range p {
				if x != 0xff {
					is_opaque.Store(false)
					return true
				}
			}
			if !keep_going.Load() {
				return false
			}
			pix = pix[stride:]
		}
		return false
	}, 0, h); err != nil {
		panic(err)
	}
	return is_opaque.Load()
}

func is_opaque8(pix []uint8, w, h, stride int) bool {
	is_opaque := atomic.Bool{}
	is_opaque.Store(true)
	if err := parallel.Run_in_parallel_to_first_result(0, func(start, limit int, keep_going *atomic.Bool) bool {
		pix := pix[stride*start:]
		for range limit - start {
			p := pix[0 : 4*w : 4*w]
			for range w {
				if p[3] != 0xff {
					is_opaque.Store(false)
					return true
				}
				p = p[4:]
			}
			if !keep_going.Load() {
				return false
			}
			pix = pix[stride:]
		}
		return false
	}, 0, h); err != nil {
		panic(err)
	}
	return is_opaque.Load()
}

func is_opaque16(pix []uint8, w, h, stride int) bool {
	is_opaque := atomic.Bool{}
	is_opaque.Store(true)
	if err := parallel.Run_in_parallel_to_first_result(0, func(start, limit int, keep_going *atomic.Bool) bool {
		pix := pix[stride*start:]
		for range limit - start {
			p := pix[0 : 8*w : 8*w]
			for range w {
				if p[6] != 0xff || p[7] != 0xff {
					is_opaque.Store(false)
					return true
				}
				p = p[8:]
			}
			if !keep_going.Load() {
				return false
			}
			pix = pix[stride:]
		}
		return false
	}, 0, h); err != nil {
		panic(err)
	}
	return is_opaque.Load()
}

func IsOpaqueType(img image.Image) (ans bool) {
	switch img.(type) {
	case *nrgb.Image, *image.CMYK, *image.YCbCr, *image.Gray, *image.Gray16:
		return true

	default:
		return false
	}
}

func IsOpaque(img image.Image) (ans bool) {
	type is_opaque interface{ Opaque() bool }
	if img.Bounds().Empty() {
		return true
	}
	switch img := img.(type) {
	case *nrgb.Image, *image.CMYK, *image.YCbCr, *image.Gray, *image.Gray16:
		return true
	case *image.NRGBA:
		return is_opaque8(img.Pix, img.Bounds().Dx(), img.Bounds().Dy(), img.Stride)
	case *image.RGBA:
		return is_opaque8(img.Pix, img.Bounds().Dx(), img.Bounds().Dy(), img.Stride)
	case *image.NRGBA64:
		return is_opaque16(img.Pix, img.Bounds().Dx(), img.Bounds().Dy(), img.Stride)
	case *image.RGBA64:
		return is_opaque16(img.Pix, img.Bounds().Dx(), img.Bounds().Dy(), img.Stride)
	case *image.NYCbCrA:
		return is_opaque1(img.A, img.Bounds().Dx(), img.Bounds().Dy(), img.AStride)
	case *image.Paletted:
		bad_colors := make([]uint8, 0, len(img.Palette))
		for i, c := range img.Palette {
			_, _, _, a := c.RGBA()
			if a != 0xffff {
				bad_colors = append(bad_colors, uint8(i))
			}
		}
		switch len(bad_colors) {
		case 0:
			return true
		case len(img.Palette):
			return false
		case 1:
			return bytes.IndexByte(img.Pix, bad_colors[0]) < 0
		default:
			is_opaque := atomic.Bool{}
			is_opaque.Store(true)
			if err := parallel.Run_in_parallel_to_first_result(0, func(start, limit int, keep_going *atomic.Bool) bool {
				for i := start; i < limit && keep_going.Load(); i++ {
					if bytes.IndexByte(img.Pix, bad_colors[i]) > -1 {
						is_opaque.Store(false)
						return true
					}
				}
				return false
			}, 0, len(bad_colors)); err != nil {
				panic(err)
			}
			return is_opaque.Load()
		}
	case is_opaque:
		return img.Opaque()
	}
	return false
}
