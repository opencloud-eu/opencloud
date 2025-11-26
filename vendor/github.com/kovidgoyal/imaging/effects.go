package imaging

import (
	"image"
	"math"

	"github.com/kovidgoyal/imaging/nrgba"
)

func gaussianBlurKernel(x, sigma float64) float64 {
	return math.Exp(-(x*x)/(2*sigma*sigma)) / (sigma * math.Sqrt(2*math.Pi))
}

// Blur produces a blurred version of the image using a Gaussian function.
// Sigma parameter must be positive and indicates how much the image will be blurred.
//
// Example:
//
//	dstImage := imaging.Blur(srcImage, 3.5)
func Blur(img image.Image, sigma float64) *image.NRGBA {
	if sigma <= 0 {
		return Clone(img)
	}

	radius := int(math.Ceil(sigma * 3.0))
	kernel := make([]float64, radius+1)

	for i := 0; i <= radius; i++ {
		kernel[i] = gaussianBlurKernel(float64(i), sigma)
	}

	return blurVertical(blurHorizontal(img, kernel), kernel)
}

func blurHorizontal(img image.Image, kernel []float64) *image.NRGBA {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	src := nrgba.NewNRGBAScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	radius := len(kernel) - 1

	if err := run_in_parallel_over_range(0, func(start, limit int) {
		scanLine := make([]uint8, w*4)
		scanLineF := make([]float64, len(scanLine))
		for y := start; y < limit; y++ {
			src.Scan(0, y, w, y+1, scanLine)
			for i, v := range scanLine {
				scanLineF[i] = float64(v)
			}
			for x := range w {
				minv := max(0, x-radius)
				maxv := min(x+radius, w-1)
				var r, g, b, a, wsum float64
				for ix := minv; ix <= maxv; ix++ {
					i := ix * 4
					weight := kernel[absint(x-ix)]
					wsum += weight
					s := scanLineF[i : i+4 : i+4]
					wa := s[3] * weight
					r += s[0] * wa
					g += s[1] * wa
					b += s[2] * wa
					a += wa
				}
				if a != 0 {
					aInv := 1 / a
					j := y*dst.Stride + x*4
					d := dst.Pix[j : j+4 : j+4]
					d[0] = clamp(r * aInv)
					d[1] = clamp(g * aInv)
					d[2] = clamp(b * aInv)
					d[3] = clamp(a / wsum)
				}
			}
		}
	}, 0, h); err != nil {
		panic(err)
	}

	return dst
}

func blurVertical(img image.Image, kernel []float64) *image.NRGBA {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	src := nrgba.NewNRGBAScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	radius := len(kernel) - 1

	if err := run_in_parallel_over_range(0, func(start, limit int) {
		scanLine := make([]uint8, h*4)
		scanLineF := make([]float64, len(scanLine))
		for x := start; x < limit; x++ {
			src.Scan(x, 0, x+1, h, scanLine)
			for i, v := range scanLine {
				scanLineF[i] = float64(v)
			}
			for y := range h {
				minv := max(0, y-radius)
				maxv := min(y+radius, h-1)
				var r, g, b, a, wsum float64
				for iy := minv; iy <= maxv; iy++ {
					i := iy * 4
					weight := kernel[absint(y-iy)]
					wsum += weight
					s := scanLineF[i : i+4 : i+4]
					wa := s[3] * weight
					r += s[0] * wa
					g += s[1] * wa
					b += s[2] * wa
					a += wa
				}
				if a != 0 {
					aInv := 1 / a
					j := y*dst.Stride + x*4
					d := dst.Pix[j : j+4 : j+4]
					d[0] = clamp(r * aInv)
					d[1] = clamp(g * aInv)
					d[2] = clamp(b * aInv)
					d[3] = clamp(a / wsum)
				}
			}
		}
	}, 0, w); err != nil {
		panic(err)
	}

	return dst
}

// Sharpen produces a sharpened version of the image.
// Sigma parameter must be positive and indicates how much the image will be sharpened.
//
// Example:
//
//	dstImage := imaging.Sharpen(srcImage, 3.5)
func Sharpen(img image.Image, sigma float64) *image.NRGBA {
	if sigma <= 0 {
		return Clone(img)
	}

	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	src := nrgba.NewNRGBAScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	blurred := Blur(img, sigma)

	if err := run_in_parallel_over_range(0, func(start, limit int) {
		scanLine := make([]uint8, w*4)
		for y := start; y < limit; y++ {
			src.Scan(0, y, w, y+1, scanLine)
			j := y * dst.Stride
			for i := 0; i < w*4; i++ {
				val := int(scanLine[i])<<1 - int(blurred.Pix[j])
				if val < 0 {
					val = 0
				} else if val > 0xff {
					val = 0xff
				}
				dst.Pix[j] = uint8(val)
				j++
			}
		}
	}, 0, h); err != nil {
		panic(err)
	}

	return dst
}
