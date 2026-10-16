package workflow

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/preprocessor"
)

// encodeForUpload turns the value produced by the preprocessing step into bytes
// for upload to the generator, and says which type they are in. Only the text
// renderer makes an image here, everything else is handed on as it was found.
func encodeForUpload(v any) ([]byte, string, error) {
	switch data := v.(type) {
	case preprocessor.ImageFile:
		return data.Data, data.ContentType, nil
	case image.Image:
		var buf bytes.Buffer
		if err := png.Encode(&buf, data); err != nil {
			return nil, "", fmt.Errorf("encode rendered image: %w", err)
		}

		return buf.Bytes(), "image/png", nil
	default:
		return nil, "", fmt.Errorf("unsupported converted type %T", v)
	}
}
