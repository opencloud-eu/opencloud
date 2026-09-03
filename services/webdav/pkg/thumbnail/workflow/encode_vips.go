//go:build enable_vips

package workflow

import (
	"fmt"
	"image"
	"image/gif"

	vips "github.com/davidbyttow/govips/v2/vips"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/generator"
)

// encodeForUpload turns the value produced by the preprocessing step into
// bytes for upload to the generator. Raw image bytes pass through unchanged;
// converted values are encoded in-process via libvips.
func encodeForUpload(v any, mimeType string) ([]byte, string, error) {
	switch data := v.(type) {
	case []byte:
		return data, generator.GuessExtension(mimeType), nil
	case image.Image:
		img, err := vips.NewImageFromGoImage(data)
		if err != nil {
			return nil, "", fmt.Errorf("create vips image: %w", err)
		}
		defer img.Close()

		if mimeType == "image/jpeg" || mimeType == "image/jpg" {
			buf, _, err := img.ExportJpeg(&vips.JpegExportParams{Quality: 85})
			return buf, "jpg", err
		}
		buf, _, err := img.ExportPng(&vips.PngExportParams{})
		return buf, "png", err
	case *gif.GIF:
		if len(data.Image) == 0 {
			return nil, "", fmt.Errorf("gif has no frames")
		}
		img, err := vips.NewImageFromGoImage(data.Image[0])
		if err != nil {
			return nil, "", fmt.Errorf("create vips image: %w", err)
		}
		defer img.Close()

		buf, _, err := img.ExportGIF(&vips.GifExportParams{})
		return buf, "gif", err
	default:
		return nil, "", fmt.Errorf("unsupported converted type %T", v)
	}
}
