//go:build enable_vips

package preprocessor

import (
	"io"

	vips "github.com/davidbyttow/govips/v2/vips"
	"github.com/kovidgoyal/imaging"
	"github.com/pkg/errors"
)

func init() {
	vips.LoggingSettings(nil, vips.LogLevelError)
}

// ImageDecoder is a converter for the image file. It decodes with imaging so the
// downstream encode step always receives an image.Image (the vips-backed
// encoder re-wraps it into libvips itself). Returning a *vips.ImageRef here
// would leak a C image past the workflow's ownership boundary.
type ImageDecoder struct{}

func (i ImageDecoder) Convert(r io.Reader) (any, error) {
	img, err := imaging.Decode(r, imaging.AutoOrientation(true))
	if err != nil {
		return nil, errors.Wrap(err, `could not decode the image`)
	}
	return img, nil
}
