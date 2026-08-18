//go:build enable_vips

package preprocessor

import (
	"io"

	"github.com/davidbyttow/govips/v2/vips"

	thumbnailerErrors "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
)

func init() {
	vips.LoggingSettings(nil, vips.LogLevelError)
}

type ImageDecoder struct{ limit decodeLimit }

func (v ImageDecoder) Convert(r io.Reader) (interface{}, error) {
	// NewImageFromReader is header-lazy, Width/Height read the header without
	// materializing pixels: reject oversized sources before ThumbnailWithSize
	img, err := vips.NewImageFromReader(r)
	if err != nil {
		return nil, err
	}
	if v.limit.exceeded(img.Width(), img.Height()) {
		return nil, thumbnailerErrors.ErrImageTooLarge
	}
	return img, nil
}
