//go:build !enable_vips

package thumbnail

import (
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
)

// PngEncoder encodes to png
type PngEncoder struct{}

// Encode encodes to png format
func (e PngEncoder) Encode(w io.Writer, img any) error {
	m, ok := img.(image.Image)
	if !ok {
		return errors.ErrInvalidType
	}
	return png.Encode(w, m)
}

// Types returns the png suffix
func (e PngEncoder) Types() []string {
	return []string{typePng}
}

// MimeType returns the mimetype for png files.
func (e PngEncoder) MimeType() string {
	return "image/png"
}

// JpegEncoder encodes to jpg
type JpegEncoder struct{}

// Encode encodes to jpg
func (e JpegEncoder) Encode(w io.Writer, img any) error {
	m, ok := img.(image.Image)
	if !ok {
		return errors.ErrInvalidType
	}
	return jpeg.Encode(w, m, nil)
}

// Types returns the jpg suffixes.
func (e JpegEncoder) Types() []string {
	return []string{typeJpeg, typeJpg}
}

// MimeType returns the mimetype for jpg files.
func (e JpegEncoder) MimeType() string {
	return "image/jpeg"
}
