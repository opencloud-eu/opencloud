package thumbnail

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
	_resolutionSeparator = "x"
)

// ParseResolution returns an image.Rectangle representing the resolution given as a string.
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

// Resolutions is a list of supported target resolutions, split at parse time into
// landscape (width >= height, including squares) and portrait (height > width)
// groups so a request can be matched against the group that fits its orientation.
type Resolutions struct {
	landscape []image.Rectangle
	portrait  []image.Rectangle
}

// ParseResolutions creates a Resolutions instance from resolution strings,
// partitioning each into the landscape or portrait group by its orientation.
func ParseResolutions(strs []string) (*Resolutions, error) {
	rs := &Resolutions{}
	for _, s := range strs {
		r, err := ParseResolution(s)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse resolutions")
		}
		if r.Dx() >= r.Dy() {
			rs.landscape = append(rs.landscape, r)
		} else {
			rs.portrait = append(rs.portrait, r)
		}
	}
	return rs, nil
}

// Match returns the target resolution to request from the generator for a given
// requested size. It is source-less: the choice depends only on the requested
// orientation and dominant dimension, never on the (unknown to webdav) source
// size. The request's orientation selects the group (width >= height -> landscape,
// else portrait); within that group the resolution whose dominant dimension is the
// smallest one still at least the requested dominant dimension is returned (the
// closest fit without going under), or the largest in the group if none qualifies.
// An empty list yields the request itself.
func (rs *Resolutions) Match(requested image.Rectangle) image.Rectangle {
	group := rs.portrait
	if requested.Dx() >= requested.Dy() {
		group = rs.landscape
	}

	if len(group) == 0 {
		return requested
	}

	reqDom := dominant(requested)

	var match image.Rectangle
	found := false
	for _, current := range group {
		cDom := dominant(current)
		if cDom < reqDom {
			continue
		}
		if !found || cDom < dominant(match) {
			match = current
			found = true
		}
	}

	if !found {
		for _, current := range group {
			if !found || dominant(current) > dominant(match) {
				match = current
				found = true
			}
		}
	}
	return match
}

// dominant returns the longer dimension of a rectangle.
func dominant(r image.Rectangle) int {
	if r.Dx() > r.Dy() {
		return r.Dx()
	}
	return r.Dy()
}
