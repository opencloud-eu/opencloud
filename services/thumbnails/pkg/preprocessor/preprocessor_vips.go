//go:build enable_vips

package preprocessor

import (
	"io"

	"github.com/cshum/vipsgen/vips"
)

func init() {
	// vips.LoggingSettings(nil, vips.LogLevelError)

	// TODO: Hookup logging
	// vips.SetLogging(nil, vips.LogLevelDebug)
}

type ImageDecoder struct{}

func (v ImageDecoder) Convert(r io.Reader) (interface{}, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	img, err := vips.NewImageFromBuffer(buf, nil)

	// Alternative with vips 1.8+
	// First test the RAW decoder, this is not implemented in NewImageFromBuffer
	//
	// img, err := vips.NewDcrawloadBuffer(buf, nil)
	// if err != nil {
	// 	img, err = vips.NewImageFromBuffer(buf, nil)
	// }
	return img, err
}
