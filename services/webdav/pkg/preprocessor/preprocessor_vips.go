//go:build enable_vips

package preprocessor

import (
	"io"

	"github.com/davidbyttow/govips/v2/vips"
)

func init() {
	vips.LoggingSettings(nil, vips.LogLevelError)
}

type ImageDecoder struct{}

func (v ImageDecoder) Convert(r io.Reader) (interface{}, error) {
	img, err := vips.NewImageFromReader(r)
	return img, err
}
