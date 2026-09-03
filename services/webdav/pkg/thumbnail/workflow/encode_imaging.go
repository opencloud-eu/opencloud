//go:build !enable_vips

package workflow

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/generator"
)

// encodeForUpload turns the value produced by the preprocessing step into
// bytes for upload to the generator. Raw image bytes pass through unchanged;
// converted values are encoded in-process.
func encodeForUpload(v any, mimeType string) ([]byte, string, error) {
	switch data := v.(type) {
	case []byte:
		return data, generator.GuessExtension(mimeType), nil
	case image.Image:
		var (
			buf bytes.Buffer
			err error
		)
		if mimeType == "image/jpeg" || mimeType == "image/jpg" {
			err = jpeg.Encode(&buf, data, &jpeg.Options{Quality: 85})
			return buf.Bytes(), "jpg", err
		}
		err = png.Encode(&buf, data)
		return buf.Bytes(), "png", err
	case *gif.GIF:
		var buf bytes.Buffer
		if err := gif.EncodeAll(&buf, data); err != nil {
			return nil, "", fmt.Errorf("encode gif: %w", err)
		}
		return buf.Bytes(), "gif", nil
	default:
		return nil, "", fmt.Errorf("unsupported converted type %T", v)
	}
}
