//go:build !enable_vips

package workflow

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
)

// encodeForUpload turns the value produced by the preprocessing step into
// bytes for upload to the generator, and says which type they are in.
func encodeForUpload(v any, mimeType string) ([]byte, string, error) {
	switch data := v.(type) {
	case image.Image:
		var buf bytes.Buffer
		if mimeType == "image/jpeg" || mimeType == "image/jpg" {
			err := jpeg.Encode(&buf, data, &jpeg.Options{Quality: 85})

			return buf.Bytes(), "image/jpeg", err
		}

		err := png.Encode(&buf, data)

		return buf.Bytes(), "image/png", err
	case *gif.GIF:
		var buf bytes.Buffer
		if err := gif.EncodeAll(&buf, data); err != nil {
			return nil, "", fmt.Errorf("encode gif: %w", err)
		}

		return buf.Bytes(), "image/gif", nil
	default:
		return nil, "", fmt.Errorf("unsupported converted type %T", v)
	}
}
