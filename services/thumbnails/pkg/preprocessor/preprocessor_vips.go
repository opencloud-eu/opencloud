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

func (v ImageDecoder) Convert(r io.ReadCloser) (interface{}, error) {
	src := vips.NewSource(r)
	// This fiexs the SOS invalid length but might be a security nightmare
	return vips.NewImageFromSource(src, &vips.LoadOptions{FailOnError: false})

	// Alternative with vips 1.8+
	// First test the RAW decoder, this is not implemented in NewImageFromSrc
	//
	// img, err := vips.NewDcrawloadBuffer(src, nil)
	// if err != nil {
	// 	img, err = vips.NewImageFromSource(src, nil)
	// }
	// return img, err
}
