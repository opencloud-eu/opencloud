package gifmeta

import (
	"fmt"
	"image/gif"
	"io"
	"strings"
	"time"

	"github.com/kovidgoyal/imaging/prism/meta"
	"github.com/kovidgoyal/imaging/types"
)

var _ = fmt.Print

func ExtractMetadata(r io.Reader) (md *meta.Data, err error) {
	c, err := gif.DecodeConfig(r)
	if err != nil {
		if strings.Contains(err.Error(), "gif: can't recognize format") {
			err = nil
		}
		return nil, err
	}
	md = &meta.Data{
		Format: types.GIF, PixelWidth: uint32(c.Width), PixelHeight: uint32(c.Height),
		BitsPerComponent: 8, HasFrames: true,
	}
	return md, nil
}

func CalcMinimumGap(gaps []int) (min_gap int) {
	// Some broken GIF images have all zero gaps, browsers with their usual
	// idiot ideas render these with a default 100ms gap https://bugzilla.mozilla.org/show_bug.cgi?id=125137
	// Browsers actually force a 100ms gap at any zero gap frame, but that
	// just means it is impossible to deliberately use zero gap frames for
	// sophisticated blending, so we dont do that.
	max_gap := 0
	for _, g := range gaps {
		max_gap = max(max_gap, g)
	}
	if max_gap <= 0 {
		min_gap = 10
	}
	return min_gap
}

func CalculateFrameDelay(delay, min_gap int) time.Duration {
	delay_ms := max(min_gap, delay)
	return time.Duration(delay_ms) * 10 * time.Millisecond
}
