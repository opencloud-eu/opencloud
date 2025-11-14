package imaging

import (
	"image"
	"sync"

	"github.com/kovidgoyal/imaging/nrgba"
)

// Histogram returns a normalized histogram of an image.
//
// Resulting histogram is represented as an array of 256 floats, where
// histogram[i] is a probability of a pixel being of a particular luminance i.
func Histogram(img image.Image) [256]float64 {
	var mu sync.Mutex
	var histogram [256]float64
	var total float64

	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if w == 0 || h == 0 {
		return histogram
	}
	src := nrgba.NewNRGBAScanner(img)

	if err := run_in_parallel_over_range(0, func(start, limit int) {
		var tmpHistogram [256]float64
		var tmpTotal float64
		scanLine := make([]uint8, w*4)
		for y := start; y < limit; y++ {
			src.Scan(0, y, w, y+1, scanLine)
			i := 0
			for range w {
				s := scanLine[i : i+3 : i+3]
				r := s[0]
				g := s[1]
				b := s[2]
				y := 0.299*float32(r) + 0.587*float32(g) + 0.114*float32(b)
				tmpHistogram[int(y+0.5)]++
				tmpTotal++
				i += 4
			}
		}
		mu.Lock()
		for i := range 256 {
			histogram[i] += tmpHistogram[i]
		}
		total += tmpTotal
		mu.Unlock()
	}, 0, h); err != nil {
		panic(err)
	}

	for i := range 256 {
		histogram[i] = histogram[i] / total
	}
	return histogram
}
