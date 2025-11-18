package imaging

import (
	"image"
	"math"
	"runtime"
	"sync/atomic"

	"github.com/kovidgoyal/go-parallel"
)

var max_procs atomic.Int64

// SetMaxProcs limits the number of concurrent processing goroutines to the given value.
// A value <= 0 clears the limit.
func SetMaxProcs(value int) {
	max_procs.Store(int64(value))
}

// Run the specified function in parallel over chunks from the specified range.
// If the function panics, it is turned into a regular error.
func run_in_parallel_over_range(num_procs int, f func(int, int), start, limit int) (err error) {
	if num_procs <= 0 {
		num_procs = runtime.GOMAXPROCS(0)
		if mp := int(max_procs.Load()); mp > 0 {
			num_procs = min(num_procs, mp)
		}
	}
	return parallel.Run_in_parallel_over_range(num_procs, f, start, limit)
}

// absint returns the absolute value of i.
func absint(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

// clamp rounds and clamps float64 value to fit into uint8.
func clamp(x float64) uint8 {
	v := int64(x + 0.5)
	if v > 255 {
		return 255
	}
	if v > 0 {
		return uint8(v)
	}
	return 0
}

func toNRGBA(img image.Image) *image.NRGBA {
	if img, ok := img.(*image.NRGBA); ok {
		return &image.NRGBA{
			Pix:    img.Pix,
			Stride: img.Stride,
			Rect:   img.Rect.Sub(img.Rect.Min),
		}
	}
	return Clone(img)
}

// rgbToHSL converts a color from RGB to HSL.
func rgbToHSL(r, g, b uint8) (float64, float64, float64) {
	rr := float64(r) / 255
	gg := float64(g) / 255
	bb := float64(b) / 255

	max := math.Max(rr, math.Max(gg, bb))
	min := math.Min(rr, math.Min(gg, bb))

	l := (max + min) / 2

	if max == min {
		return 0, 0, l
	}

	var h, s float64
	d := max - min
	if l > 0.5 {
		s = d / (2 - max - min)
	} else {
		s = d / (max + min)
	}

	switch max {
	case rr:
		h = (gg - bb) / d
		if g < b {
			h += 6
		}
	case gg:
		h = (bb-rr)/d + 2
	case bb:
		h = (rr-gg)/d + 4
	}
	h /= 6

	return h, s, l
}

// hslToRGB converts a color from HSL to RGB.
func hslToRGB(h, s, l float64) (uint8, uint8, uint8) {
	var r, g, b float64
	if s == 0 {
		v := clamp(l * 255)
		return v, v, v
	}

	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q

	r = hueToRGB(p, q, h+1/3.0)
	g = hueToRGB(p, q, h)
	b = hueToRGB(p, q, h-1/3.0)

	return clamp(r * 255), clamp(g * 255), clamp(b * 255)
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	if t < 1/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1/2.0 {
		return q
	}
	if t < 2/3.0 {
		return p + (q-p)*(2/3.0-t)*6
	}
	return p
}
