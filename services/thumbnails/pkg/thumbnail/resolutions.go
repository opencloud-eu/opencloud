package thumbnail

import (
	"fmt"
	"image"
	"math"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
	_resolutionSeparator = "x"
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

// Resolutions is a list of image.Rectangle representing resolutions.
type Resolutions []image.Rectangle

func (rs Resolutions) contains(resolution image.Rectangle) bool {
	for _, current := range rs {
		if current == resolution {
			return true
		}
	}
	return false
}

// ParseResolutions creates an instance of Resolutions from resolution strings.
func ParseResolutions(strs []string) (Resolutions, error) {
	rs := make(Resolutions, 0, len(strs))
	for _, s := range strs {
		r, err := ParseResolution(s)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse resolutions")
		}
		rs = append(rs, r)
	}
	return rs, nil
}

// ClosestMatch returns the resolution which is closest to the provided resolution.
// If there is no exact match the resolution will be the next higher one.
// If the given resolution is bigger than all available resolutions the biggest available one is used.
func (rs Resolutions) ClosestMatch(requested image.Rectangle, sourceSize image.Rectangle) image.Rectangle {
	isLandscape := sourceSize.Dx() > sourceSize.Dy()
	sourceLen := dimensionLength(sourceSize, isLandscape)
	requestedLen := dimensionLength(requested, isLandscape)
	isSourceSmaller := sourceLen < requestedLen

	// We don't want to scale images up.
	if isSourceSmaller {
		return sourceSize
	}

	if len(rs) == 0 {
		return requested
	}

	var match image.Rectangle
	// Since we want to search for the smallest difference we start with the highest possible number
	minDiff := math.MaxInt32

	for _, current := range rs {
		cLen := dimensionLength(current, isLandscape)
		diff := requestedLen - cLen
		if diff > 0 {
			// current is smaller
			continue
		}

		// Convert diff to positive value
		// Multiplying by -1 is safe since we aren't getting positive numbers here
		// because of the check above
		absDiff := diff * -1
		if absDiff < minDiff {
			minDiff = absDiff
			match = current
		}
	}

	if (match == image.Rectangle{}) {
		match = rs[len(rs)-1]
	}
	return match
}

func dimensionLength(rect image.Rectangle, isLandscape bool) int {
	if isLandscape {
		return rect.Dx()
	}
	return rect.Dy()
}
