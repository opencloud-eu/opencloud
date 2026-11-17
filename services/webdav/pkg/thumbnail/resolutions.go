package thumbnail

import (
	"fmt"
	"image"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
	_resolutionSeparator = "x"

	// legacyMaxEdge caps a side a legacy resolution contributes. Those boxes were
	// written for the old service, which had no ceiling and listed boxes up to
	// 7680x4320 by default, and 2048 is the largest box the built-in generator
	// accepts by default (THUMBNAILS_MAX_OUTPUT_WIDTH and _HEIGHT).
	legacyMaxEdge = 2048
)

// ParseResolution returns an image.Rectangle representing the resolution given as a string
func ParseResolution(s string) (image.Rectangle, error) {
	parts := strings.Split(s, _resolutionSeparator)
	if len(parts) != 2 {
		return image.Rectangle{}, fmt.Errorf("failed to parse resolution: %s. Expected format <width>x<height>", s)
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return image.Rectangle{}, fmt.Errorf("width: %s has an invalid value. Expected an integer", parts[0])
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return image.Rectangle{}, fmt.Errorf("height: %s has an invalid value. Expected an integer", parts[1])
	}
	return image.Rect(0, 0, width, height), nil
}

// Sizes is the ladder of edge lengths a generated thumbnail may have. Each axis
// of a request is snapped onto it separately, so the ladder stands for every
// combination of its lengths without the orientation of the request mattering.
type Sizes struct {
	asc []int
}

// ParseSizes builds the ladder from configuration. An entry is either a plain
// edge length ("512") or a legacy resolution ("500x280"), which contributes both
// of its sides, capped at legacyMaxEdge, so an existing box configuration keeps
// working against the default generator.
func ParseSizes(strs []string) (*Sizes, error) {
	seen := map[int]struct{}{}
	for _, s := range strs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if strings.Contains(s, _resolutionSeparator) {
			r, err := ParseResolution(s)
			if err != nil {
				return nil, errors.Wrap(err, "could not parse thumbnail sizes")
			}
			seen[min(r.Dx(), legacyMaxEdge)] = struct{}{}
			seen[min(r.Dy(), legacyMaxEdge)] = struct{}{}
			continue
		}
		v, err := strconv.Atoi(s)
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse thumbnail size %q", s)
		}
		seen[v] = struct{}{}
	}

	sizes := &Sizes{}
	for v := range seen {
		if v > 0 {
			sizes.asc = append(sizes.asc, v)
		}
	}
	sort.Ints(sizes.asc)
	return sizes, nil
}

// Snap returns the box to request from the generator. Both axes are rounded up
// to the next configured length, so the result always covers what was asked for
// and a client can do the last bit of cropping itself. An axis of zero is
// unconstrained and takes the largest configured length, which keeps the output
// bounded in both directions.
func (s *Sizes) Snap(requested image.Rectangle) image.Rectangle {
	return image.Rect(0, 0, s.snap(requested.Dx()), s.snap(requested.Dy()))
}

// Max returns the largest configured length, or zero when nothing is configured.
func (s *Sizes) Max() int {
	if len(s.asc) == 0 {
		return 0
	}
	return s.asc[len(s.asc)-1]
}

func (s *Sizes) snap(v int) int {
	if len(s.asc) == 0 {
		return v
	}
	if v <= 0 {
		return s.Max()
	}
	for _, c := range s.asc {
		if c >= v {
			return c
		}
	}
	return s.Max()
}
