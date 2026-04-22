//go:build !enable_vips

package preprocessor

import (
	"io"

	"github.com/kovidgoyal/imaging"
	"github.com/pkg/errors"
)

// ImageDecoder is a converter for the image file
type ImageDecoder struct{}

// Convert reads the image file and returns the thumbnail image
func (i ImageDecoder) Convert(r io.Reader) (any, error) {
	img, err := imaging.Decode(r, imaging.AutoOrientation(true))
	if err != nil {
		return nil, errors.Wrap(err, `could not decode the image`)
	}
	return img, nil
}
