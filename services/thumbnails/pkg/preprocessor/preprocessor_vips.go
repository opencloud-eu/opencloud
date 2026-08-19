//go:build enable_vips

package preprocessor

import (
	"context"
	"io"

	"github.com/davidbyttow/govips/v2/vips"
)

func init() {
	vips.LoggingSettings(nil, vips.LogLevelError)
}

type ImageDecoder struct{}

func (v ImageDecoder) Convert(_ context.Context, r io.Reader) (interface{}, error) {
	img, err := vips.NewImageFromReader(r)
	return img, err
}
