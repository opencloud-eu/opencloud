//go:build enable_vips

package thumbnail

import (
	"encoding/base64"
	"sync"

	"github.com/davidbyttow/govips/v2/vips"
)

const (
	// The probes are stripped 1x1 black images. Decoding an actual pixel checks
	// both the libvips HEIF loader and the codec backend used by libheif.
	heicDecodeProbe = "AAAAGGZ0eXBoZWljAAAAAG1pZjFoZWljAAABaG1ldGEAAAAAAAAAIWhkbHIAAAAAAAAAAHBpY3QAAAAAAAAAAAAAAAAAAAAAImlsb2MAAAAAREAAAQABAAAAAAGIAAEAAAAAAAAAIgAAACNpaW5mAAAAAAABAAAAFWluZmUCAAAAAAEAAGh2YzEAAAAADnBpdG0AAAAAAAEAAADoaXBycAAAAMlpcGNvAAAAdWh2Y0MBA3AAAAAAAAAAAAAe8AD8/fj4AAAPA2AAAQAYQAEMAf//A3AAAAMAkAAAAwAAAwAeugJAYQABAClCAQEDcAAAAwCQAAADAAADAB6gIIEFluqumubgIaDAgAAADIAAAAMAhGIAAQAGRAHBc8GJAAAAFGlzcGUAAAAAAAAAQAAAAEAAAAAoY2xhcAAAAAEAAAABAAAAAQAAAAH////BAAAAAv///8EAAAACAAAAEHBpeGkAAAAAAwgICAAAABdpcG1hAAAAAAAAAAEAAQSBAgSDAAAAKm1kYXQAAAAeKAGvBRISTOD6A73n/ug+iFmAr7tlWKOZXGCTU7eA"
	avifDecodeProbe = "AAAAHGZ0eXBhdmlmAAAAAG1pZjFhdmlmbWlhZgAAANZtZXRhAAAAAAAAACFoZGxyAAAAAAAAAABwaWN0AAAAAAAAAAAAAAAAAAAAACJpbG9jAAAAAERAAAEAAQAAAAAA+gABAAAAAAAAAB4AAAAjaWluZgAAAAAAAQAAABVpbmZlAgAAAAABAABhdjAxAAAAAA5waXRtAAAAAAABAAAAVmlwcnAAAAA4aXBjbwAAAAxhdjFDgQAMAAAAABRpc3BlAAAAAAAAAAEAAAABAAAAEHBpeGkAAAAAAwgICAAAABZpcG1hAAAAAAAAAAEAAQOBAgMAAAAmbWRhdBIACggYAAaICGg0IDIQH/eHhRXAACwAAJA1jjx+3A=="
)

var (
	optionalMimeTypesOnce sync.Once

	// SupportedMimeTypes contains an all mimetypes which are supported by the thumbnailer.
	SupportedMimeTypes = map[string]struct{}{
		"image/png":                         {},
		"image/jpg":                         {},
		"image/jpeg":                        {},
		"image/gif":                         {},
		"image/bmp":                         {},
		"image/x-ms-bmp":                    {},
		"image/tiff":                        {},
		"text/plain":                        {},
		"audio/flac":                        {},
		"audio/mpeg":                        {},
		"audio/ogg":                         {},
		"application/vnd.geogebra.slides":   {},
		"application/vnd.geogebra.pinboard": {},
		"image/webp":                        {},
	}

	// heifMimeTypes are the mimetypes commonly used for HEVC-coded images in a
	// HEIF container.
	heifMimeTypes = []string{
		"image/heic",
		"image/heic-sequence",
		"image/heif",
		"image/heif-sequence",
	}

	// avifMimeTypes go through the same loader but are coded as AV1.
	avifMimeTypes = []string{
		"image/avif",
	}
)

func initializeSupportedMimeTypes() {
	optionalMimeTypesOnce.Do(func() {
		// Running a decoder during package initialization can deadlock because
		// libvips evaluates images on worker threads. Defer the probes until the
		// MIME type set is first queried at runtime.
		vips.LoggingSettings(nil, vips.LogLevelError)
		registerOptionalMimeTypes()
	})
}

// registerOptionalMimeTypes adds MIME types that libvips can read only through
// optional loader and codec packages. OpenCloud does not ship an HEVC decoder
// because of the patent situation (see README.md). Registering MIME types only
// when decoding succeeds keeps oc:has-preview and graph thumbnail links in sync
// with what the thumbnails service can deliver.
func registerOptionalMimeTypes() {
	registerOptionalMimeTypesWithDecoder(SupportedMimeTypes, canDecodeImage)
}

func registerOptionalMimeTypesWithDecoder(supportedMimeTypes map[string]struct{}, canDecode func(string) bool) {
	if canDecode(heicDecodeProbe) {
		for _, mimeType := range heifMimeTypes {
			supportedMimeTypes[mimeType] = struct{}{}
		}
	}
	if canDecode(avifDecodeProbe) {
		for _, mimeType := range avifMimeTypes {
			supportedMimeTypes[mimeType] = struct{}{}
		}
	}
}

// canDecodeImage forces libvips to decode one pixel. Creating an ImageRef alone
// is not enough because libvips evaluates image operations lazily.
func canDecodeImage(encodedProbe string) bool {
	probe, err := base64.StdEncoding.DecodeString(encodedProbe)
	if err != nil {
		return false
	}

	img, err := vips.NewImageFromBuffer(probe)
	if err != nil {
		return false
	}
	defer img.Close()

	_, err = img.GetPoint(0, 0)
	return err == nil
}
