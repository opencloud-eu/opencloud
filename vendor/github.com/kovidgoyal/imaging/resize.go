package imaging

import (
	"image"
	"math"

	"github.com/kovidgoyal/imaging/nrgb"
	"github.com/kovidgoyal/imaging/nrgba"
)

type indexWeight struct {
	index  int
	weight float64
}

func precomputeWeights(dstSize, srcSize int, filter ResampleFilter) [][]indexWeight {
	du := float64(srcSize) / float64(dstSize)
	scale := max(1.0, du)
	ru := math.Ceil(scale * filter.Support)

	out := make([][]indexWeight, dstSize)
	tmp := make([]indexWeight, 0, dstSize*int(ru+2)*2)

	for v := range dstSize {
		fu := (float64(v)+0.5)*du - 0.5

		begin := max(0, int(math.Ceil(fu-ru)))
		end := min(int(math.Floor(fu+ru)), srcSize-1)

		var sum float64
		for u := begin; u <= end; u++ {
			w := filter.Kernel((float64(u) - fu) / scale)
			if w != 0 {
				sum += w
				tmp = append(tmp, indexWeight{index: u, weight: w})
			}
		}
		if sum != 0 {
			for i := range tmp {
				tmp[i].weight /= sum
			}
		}

		out[v] = tmp
		tmp = tmp[len(tmp):]
	}

	return out
}

// Resize resizes the image to the specified width and height using the specified resampling
// filter and returns the transformed image. If one of width or height is 0, the image aspect
// ratio is preserved. When is_opaque is true, returns a nrgb.Image otherwise
// an image.NRGBA. When the image size is unchanged returns a clone with the
// same image type.
//
// Example:
//
//	dstImage := imaging.Resize(srcImage, 800, 600, imaging.Lanczos)
func ResizeWithOpacity(img image.Image, width, height int, filter ResampleFilter, is_opaque bool) image.Image {
	dstW, dstH := width, height
	if dstW < 0 || dstH < 0 || (dstW == 0 && dstH == 0) {
		if is_opaque {
			return &NRGB{}
		}
		return &image.NRGBA{}
	}
	srcW := img.Bounds().Dx()
	srcH := img.Bounds().Dy()
	if srcW <= 0 || srcH <= 0 {
		if is_opaque {
			return &NRGB{}
		}
		return &image.NRGBA{}
	}

	// If new width or height is 0 then preserve aspect ratio, minimum 1px.
	if dstW == 0 {
		tmpW := float64(dstH) * float64(srcW) / float64(srcH)
		dstW = int(math.Max(1.0, math.Floor(tmpW+0.5)))
	}
	if dstH == 0 {
		tmpH := float64(dstW) * float64(srcH) / float64(srcW)
		dstH = int(math.Max(1.0, math.Floor(tmpH+0.5)))
	}

	if srcW == dstW && srcH == dstH {
		return ClonePreservingType(img)
	}

	if filter.Support <= 0 {
		// Nearest-neighbor special case.
		if is_opaque {
			return resizeNearest(img, dstW, dstH)
		}
		return resizeNearestWithAlpha(img, dstW, dstH)
	}

	hr := func(img image.Image, dim int) image.Image {
		if is_opaque {
			return resizeHorizontal(img, dim, filter)
		}
		return resizeHorizontalWithAlpha(img, dim, filter)
	}
	vr := func(img image.Image, dim int) image.Image {
		if is_opaque {
			return resizeVertical(img, dim, filter)
		}
		return resizeVerticalWithAlpha(img, dim, filter)
	}

	if srcW != dstW && srcH != dstH {
		return vr(hr(img, dstW), dstH)
	}
	if srcW != dstW {
		return hr(img, dstW)
	}
	return vr(img, dstH)
}

func Resize(img image.Image, width, height int, filter ResampleFilter) image.Image {
	return ResizeWithOpacity(img, width, height, filter, false)
}

func resizeHorizontal(img image.Image, width int, filter ResampleFilter) *nrgb.Image {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	src := nrgb.NewNRGBScanner(img, nrgb.Color{})
	dst := nrgb.NewNRGB(image.Rect(0, 0, width, h).Add(img.Bounds().Min))
	weights := precomputeWeights(width, w, filter)
	if err := run_in_parallel_over_range(0, func(start, limit int) {
		scanLine := make([]uint8, w*3)
		for y := start; y < limit; y++ {
			src.Scan(0, y, w, y+1, scanLine)
			j0 := y * dst.Stride
			for x := range weights {
				var r, g, b float64
				for _, w := range weights[x] {
					i := w.index * 3
					s := scanLine[i : i+3 : i+3]
					r += float64(s[0]) * w.weight
					g += float64(s[1]) * w.weight
					b += float64(s[2]) * w.weight
				}
				j := j0 + x*3
				d := dst.Pix[j : j+3 : j+3]
				d[0] = clamp(r)
				d[1] = clamp(g)
				d[2] = clamp(b)
			}
		}
	}, 0, h); err != nil {
		panic(err)
	}
	return dst
}

func resizeHorizontalWithAlpha(img image.Image, width int, filter ResampleFilter) *image.NRGBA {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	src := nrgba.NewNRGBAScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, width, h).Add(img.Bounds().Min))
	weights := precomputeWeights(width, w, filter)
	if err := run_in_parallel_over_range(0, func(start, limit int) {
		scanLine := make([]uint8, w*4)
		for y := start; y < limit; y++ {
			src.Scan(0, y, w, y+1, scanLine)
			j0 := y * dst.Stride
			for x := range weights {
				var r, g, b, a float64
				for _, w := range weights[x] {
					i := w.index * 4
					s := scanLine[i : i+4 : i+4]
					aw := float64(s[3]) * w.weight
					r += float64(s[0]) * aw
					g += float64(s[1]) * aw
					b += float64(s[2]) * aw
					a += aw
				}
				if a != 0 {
					aInv := 1 / a
					j := j0 + x*4
					d := dst.Pix[j : j+4 : j+4]
					d[0] = clamp(r * aInv)
					d[1] = clamp(g * aInv)
					d[2] = clamp(b * aInv)
					d[3] = clamp(a)
				}
			}
		}
	}, 0, h); err != nil {
		panic(err)
	}
	return dst
}

func resizeVertical(img image.Image, height int, filter ResampleFilter) *nrgb.Image {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	src := nrgb.NewNRGBScanner(img, nrgb.Color{})
	dst := nrgb.NewNRGB(image.Rect(0, 0, w, height).Add(img.Bounds().Min))
	weights := precomputeWeights(height, h, filter)
	if err := run_in_parallel_over_range(0, func(start, limit int) {
		scanLine := make([]uint8, h*3)
		for x := start; x < limit; x++ {
			src.Scan(x, 0, x+1, h, scanLine)
			for y := range weights {
				var r, g, b float64
				for _, w := range weights[y] {
					i := w.index * 3
					s := scanLine[i : i+3 : i+3]
					r += float64(s[0]) * w.weight
					g += float64(s[1]) * w.weight
					b += float64(s[2]) * w.weight
				}
				j := y*dst.Stride + x*3
				d := dst.Pix[j : j+3 : j+3]
				d[0] = clamp(r)
				d[1] = clamp(g)
				d[2] = clamp(b)
			}
		}
	}, 0, w); err != nil {
		panic(err)
	}
	return dst
}

func resizeVerticalWithAlpha(img image.Image, height int, filter ResampleFilter) *image.NRGBA {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	src := nrgba.NewNRGBAScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, w, height).Add(img.Bounds().Min))
	weights := precomputeWeights(height, h, filter)
	if err := run_in_parallel_over_range(0, func(start, limit int) {
		scanLine := make([]uint8, h*4)
		for x := start; x < limit; x++ {
			src.Scan(x, 0, x+1, h, scanLine)
			for y := range weights {
				var r, g, b, a float64
				for _, w := range weights[y] {
					i := w.index * 4
					s := scanLine[i : i+4 : i+4]
					aw := float64(s[3]) * w.weight
					r += float64(s[0]) * aw
					g += float64(s[1]) * aw
					b += float64(s[2]) * aw
					a += aw
				}
				if a != 0 {
					aInv := 1 / a
					j := y*dst.Stride + x*4
					d := dst.Pix[j : j+4 : j+4]
					d[0] = clamp(r * aInv)
					d[1] = clamp(g * aInv)
					d[2] = clamp(b * aInv)
					d[3] = clamp(a)
				}
			}
		}
	}, 0, w); err != nil {
		panic(err)
	}
	return dst
}

// resizeNearest is a fast nearest-neighbor resize, no filtering.
func resizeNearestWithAlpha(img image.Image, width, height int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, width, height).Add(img.Bounds().Min))
	dx := float64(img.Bounds().Dx()) / float64(width)
	dy := float64(img.Bounds().Dy()) / float64(height)

	if dx > 1 && dy > 1 {
		src := nrgba.NewNRGBAScanner(img)
		if err := run_in_parallel_over_range(0, func(start, limit int) {
			for y := start; y < limit; y++ {
				srcY := int((float64(y) + 0.5) * dy)
				dstOff := y * dst.Stride
				for x := range width {
					srcX := int((float64(x) + 0.5) * dx)
					src.Scan(srcX, srcY, srcX+1, srcY+1, dst.Pix[dstOff:dstOff+4])
					dstOff += 4
				}
			}
		}, 0, height); err != nil {
			panic(err)
		}
	} else {
		src := toNRGBA(img)
		if err := run_in_parallel_over_range(0, func(start, limit int) {
			for y := start; y < limit; y++ {
				srcY := int((float64(y) + 0.5) * dy)
				srcOff0 := srcY * src.Stride
				dstOff := y * dst.Stride
				for x := range width {
					srcX := int((float64(x) + 0.5) * dx)
					srcOff := srcOff0 + srcX*4
					copy(dst.Pix[dstOff:dstOff+4], src.Pix[srcOff:srcOff+4])
					dstOff += 4
				}
			}
		}, 0, height); err != nil {
			panic(err)
		}
	}
	return dst
}

func resizeNearest(img image.Image, width, height int) *nrgb.Image {
	dst := nrgb.NewNRGB(image.Rect(0, 0, width, height).Add(img.Bounds().Min))
	dx := float64(img.Bounds().Dx()) / float64(width)
	dy := float64(img.Bounds().Dy()) / float64(height)

	if dx > 1 && dy > 1 {
		src := nrgb.NewNRGBScanner(img, nrgb.Color{})
		if err := run_in_parallel_over_range(0, func(start, limit int) {
			for y := start; y < limit; y++ {
				srcY := int((float64(y) + 0.5) * dy)
				dstOff := y * dst.Stride
				for x := range width {
					srcX := int((float64(x) + 0.5) * dx)
					src.Scan(srcX, srcY, srcX+1, srcY+1, dst.Pix[dstOff:dstOff+3])
					dstOff += 3
				}
			}
		}, 0, height); err != nil {
			panic(err)
		}
	} else {
		src := AsNRGB(img)
		if err := run_in_parallel_over_range(0, func(start, limit int) {
			for y := start; y < limit; y++ {
				srcY := int((float64(y) + 0.5) * dy)
				srcOff0 := srcY * src.Stride
				dstOff := y * dst.Stride
				for x := range width {
					srcX := int((float64(x) + 0.5) * dx)
					srcOff := srcOff0 + srcX*3
					copy(dst.Pix[dstOff:dstOff+3], src.Pix[srcOff:srcOff+3])
					dstOff += 3
				}
			}
		}, 0, height); err != nil {
			panic(err)
		}
	}
	return dst
}

// Fit scales down the image using the specified resample filter to fit the specified
// maximum width and height and returns the transformed image.
//
// Example:
//
//	dstImage := imaging.Fit(srcImage, 800, 600, imaging.Lanczos)
func Fit(img image.Image, width, height int, filter ResampleFilter) image.Image {
	maxW, maxH := width, height

	if maxW <= 0 || maxH <= 0 {
		return &NRGB{}
	}

	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	if srcW <= 0 || srcH <= 0 {
		return &NRGB{}
	}

	if srcW <= maxW && srcH <= maxH {
		return ClonePreservingType(img)
	}

	srcAspectRatio := float64(srcW) / float64(srcH)
	maxAspectRatio := float64(maxW) / float64(maxH)

	var newW, newH int
	if srcAspectRatio > maxAspectRatio {
		newW = maxW
		newH = int(float64(newW) / srcAspectRatio)
	} else {
		newH = maxH
		newW = int(float64(newH) * srcAspectRatio)
	}

	return Resize(img, newW, newH, filter)
}

// Fill creates an image with the specified dimensions and fills it with the scaled source image.
// To achieve the correct aspect ratio without stretching, the source image will be cropped.
//
// Example:
//
//	dstImage := imaging.Fill(srcImage, 800, 600, imaging.Center, imaging.Lanczos)
func Fill(img image.Image, width, height int, anchor Anchor, filter ResampleFilter) image.Image {
	dstW, dstH := width, height

	if dstW <= 0 || dstH <= 0 {
		return &NRGB{}
	}

	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	if srcW <= 0 || srcH <= 0 {
		return &NRGB{}
	}

	if srcW == dstW && srcH == dstH {
		return ClonePreservingType(img)
	}

	if srcW >= 100 && srcH >= 100 {
		return cropAndResize(img, dstW, dstH, anchor, filter)
	}
	return resizeAndCrop(img, dstW, dstH, anchor, filter)
}

// cropAndResize crops the image to the smallest possible size that has the required aspect ratio using
// the given anchor point, then scales it to the specified dimensions and returns the transformed image.
//
// This is generally faster than resizing first, but may result in inaccuracies when used on small source images.
func cropAndResize(img image.Image, width, height int, anchor Anchor, filter ResampleFilter) image.Image {
	dstW, dstH := width, height

	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	srcAspectRatio := float64(srcW) / float64(srcH)
	dstAspectRatio := float64(dstW) / float64(dstH)

	var tmp *image.NRGBA
	if srcAspectRatio < dstAspectRatio {
		cropH := float64(srcW) * float64(dstH) / float64(dstW)
		tmp = CropAnchor(img, srcW, int(math.Max(1, cropH)+0.5), anchor)
	} else {
		cropW := float64(srcH) * float64(dstW) / float64(dstH)
		tmp = CropAnchor(img, int(math.Max(1, cropW)+0.5), srcH, anchor)
	}

	return Resize(tmp, dstW, dstH, filter)
}

// resizeAndCrop resizes the image to the smallest possible size that will cover the specified dimensions,
// crops the resized image to the specified dimensions using the given anchor point and returns
// the transformed image.
func resizeAndCrop(img image.Image, width, height int, anchor Anchor, filter ResampleFilter) *image.NRGBA {
	dstW, dstH := width, height

	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	srcAspectRatio := float64(srcW) / float64(srcH)
	dstAspectRatio := float64(dstW) / float64(dstH)

	var tmp image.Image
	if srcAspectRatio < dstAspectRatio {
		tmp = Resize(img, dstW, 0, filter)
	} else {
		tmp = Resize(img, 0, dstH, filter)
	}

	return CropAnchor(tmp, dstW, dstH, anchor)
}

// Thumbnail scales the image up or down using the specified resample filter, crops it
// to the specified width and hight and returns the transformed image.
//
// Example:
//
//	dstImage := imaging.Thumbnail(srcImage, 100, 100, imaging.Lanczos)
func Thumbnail(img image.Image, width, height int, filter ResampleFilter) image.Image {
	return Fill(img, width, height, Center, filter)
}

// ResampleFilter specifies a resampling filter to be used for image resizing.
//
//	General filter recommendations:
//
//	- Lanczos
//		A high-quality resampling filter for photographic images yielding sharp results.
//
//	- CatmullRom
//		A sharp cubic filter that is faster than Lanczos filter while providing similar results.
//
//	- MitchellNetravali
//		A cubic filter that produces smoother results with less ringing artifacts than CatmullRom.
//
//	- Linear
//		Bilinear resampling filter, produces a smooth output. Faster than cubic filters.
//
//	- Box
//		Simple and fast averaging filter appropriate for downscaling.
//		When upscaling it's similar to NearestNeighbor.
//
//	- NearestNeighbor
//		Fastest resampling filter, no antialiasing.
type ResampleFilter struct {
	Support float64
	Kernel  func(float64) float64
}

// NearestNeighbor is a nearest-neighbor filter (no anti-aliasing).
var NearestNeighbor ResampleFilter

// Box filter (averaging pixels).
var Box ResampleFilter

// Linear filter.
var Linear ResampleFilter

// Hermite cubic spline filter (BC-spline; B=0; C=0).
var Hermite ResampleFilter

// MitchellNetravali is Mitchell-Netravali cubic filter (BC-spline; B=1/3; C=1/3).
var MitchellNetravali ResampleFilter

// CatmullRom is a Catmull-Rom - sharp cubic filter (BC-spline; B=0; C=0.5).
var CatmullRom ResampleFilter

// BSpline is a smooth cubic filter (BC-spline; B=1; C=0).
var BSpline ResampleFilter

// Gaussian is a Gaussian blurring filter.
var Gaussian ResampleFilter

// Bartlett is a Bartlett-windowed sinc filter (3 lobes).
var Bartlett ResampleFilter

// Lanczos filter (3 lobes).
var Lanczos ResampleFilter

// Hann is a Hann-windowed sinc filter (3 lobes).
var Hann ResampleFilter

// Hamming is a Hamming-windowed sinc filter (3 lobes).
var Hamming ResampleFilter

// Blackman is a Blackman-windowed sinc filter (3 lobes).
var Blackman ResampleFilter

// Welch is a Welch-windowed sinc filter (parabolic window, 3 lobes).
var Welch ResampleFilter

// Cosine is a Cosine-windowed sinc filter (3 lobes).
var Cosine ResampleFilter

func bcspline(x, b, c float64) float64 {
	var y float64
	x = math.Abs(x)
	if x < 1.0 {
		y = ((12-9*b-6*c)*x*x*x + (-18+12*b+6*c)*x*x + (6 - 2*b)) / 6
	} else if x < 2.0 {
		y = ((-b-6*c)*x*x*x + (6*b+30*c)*x*x + (-12*b-48*c)*x + (8*b + 24*c)) / 6
	}
	return y
}

func sinc(x float64) float64 {
	if x == 0 {
		return 1
	}
	return math.Sin(math.Pi*x) / (math.Pi * x)
}

func init() {
	NearestNeighbor = ResampleFilter{
		Support: 0.0, // special case - not applying the filter
	}

	Box = ResampleFilter{
		Support: 0.5,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x <= 0.5 {
				return 1.0
			}
			return 0
		},
	}

	Linear = ResampleFilter{
		Support: 1.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 1.0 {
				return 1.0 - x
			}
			return 0
		},
	}

	Hermite = ResampleFilter{
		Support: 1.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 1.0 {
				return bcspline(x, 0.0, 0.0)
			}
			return 0
		},
	}

	MitchellNetravali = ResampleFilter{
		Support: 2.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 2.0 {
				return bcspline(x, 1.0/3.0, 1.0/3.0)
			}
			return 0
		},
	}

	CatmullRom = ResampleFilter{
		Support: 2.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 2.0 {
				return bcspline(x, 0.0, 0.5)
			}
			return 0
		},
	}

	BSpline = ResampleFilter{
		Support: 2.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 2.0 {
				return bcspline(x, 1.0, 0.0)
			}
			return 0
		},
	}

	Gaussian = ResampleFilter{
		Support: 2.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 2.0 {
				return math.Exp(-2 * x * x)
			}
			return 0
		},
	}

	Bartlett = ResampleFilter{
		Support: 3.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 3.0 {
				return sinc(x) * (3.0 - x) / 3.0
			}
			return 0
		},
	}

	Lanczos = ResampleFilter{
		Support: 3.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 3.0 {
				return sinc(x) * sinc(x/3.0)
			}
			return 0
		},
	}

	Hann = ResampleFilter{
		Support: 3.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 3.0 {
				return sinc(x) * (0.5 + 0.5*math.Cos(math.Pi*x/3.0))
			}
			return 0
		},
	}

	Hamming = ResampleFilter{
		Support: 3.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 3.0 {
				return sinc(x) * (0.54 + 0.46*math.Cos(math.Pi*x/3.0))
			}
			return 0
		},
	}

	Blackman = ResampleFilter{
		Support: 3.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 3.0 {
				return sinc(x) * (0.42 - 0.5*math.Cos(math.Pi*x/3.0+math.Pi) + 0.08*math.Cos(2.0*math.Pi*x/3.0))
			}
			return 0
		},
	}

	Welch = ResampleFilter{
		Support: 3.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 3.0 {
				return sinc(x) * (1.0 - (x * x / 9.0))
			}
			return 0
		},
	}

	Cosine = ResampleFilter{
		Support: 3.0,
		Kernel: func(x float64) float64 {
			x = math.Abs(x)
			if x < 3.0 {
				return sinc(x) * math.Cos((math.Pi/2.0)*(x/3.0))
			}
			return 0
		},
	}
}
