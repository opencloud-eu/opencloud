//go:build !enable_vips

package preprocessor

import (
	"image"
	"io"

	"github.com/kovidgoyal/imaging"
	"github.com/pkg/errors"
)

// ImageDecoder is a converter for the image file
type ImageDecoder struct{ limit decodeLimit }

// Convert reads the image file and returns the thumbnail image
func (i ImageDecoder) Convert(r io.Reader) (any, error) {
	// bound the declared dimensions before imaging.Decode allocates the full
	// pixel buffer: a crafted header (e.g. 65535x65535) would otherwise OOM
	// the worker, the downstream dimension guard only runs after the decode
	r, err := i.limit.guardDimensions(r, func(rr io.Reader) (image.Config, error) {
		cfg, _, err := image.DecodeConfig(rr)
		return cfg, err
	})
	if err != nil {
		return nil, err
	}
	img, err := imaging.Decode(r, imaging.AutoOrientation(true))
	if err != nil {
		return nil, errors.Wrap(err, `could not decode the image`)
	}
	return img, nil
}
