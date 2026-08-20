//go:build enable_vips

package thumbnail

import (
	"encoding/base64"
	"reflect"
	"testing"

	"github.com/davidbyttow/govips/v2/vips"
)

func TestRegisterOptionalMimeTypesWithDecoder(t *testing.T) {
	tests := []struct {
		name          string
		canDecodeHEIC bool
		canDecodeAVIF bool
		expected      map[string]struct{}
	}{
		{
			name:     "no optional codec",
			expected: map[string]struct{}{},
		},
		{
			name:          "HEVC decoder only",
			canDecodeHEIC: true,
			expected: mimeTypeSet(
				"image/heic",
				"image/heic-sequence",
				"image/heif",
				"image/heif-sequence",
			),
		},
		{
			name:          "AV1 decoder only",
			canDecodeAVIF: true,
			expected:      mimeTypeSet("image/avif"),
		},
		{
			name:          "both optional codecs",
			canDecodeHEIC: true,
			canDecodeAVIF: true,
			expected: mimeTypeSet(
				"image/heic",
				"image/heic-sequence",
				"image/heif",
				"image/heif-sequence",
				"image/avif",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canDecode := func(probe string) bool {
				switch probe {
				case heicDecodeProbe:
					return tt.canDecodeHEIC
				case avifDecodeProbe:
					return tt.canDecodeAVIF
				default:
					t.Fatalf("unexpected decoder probe")
					return false
				}
			}

			got := map[string]struct{}{}
			registerOptionalMimeTypesWithDecoder(got, canDecode)

			if !reflect.DeepEqual(tt.expected, got) {
				t.Errorf("registered MIME types = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDecoderProbesUseExpectedImageTypes(t *testing.T) {
	tests := []struct {
		name      string
		probe     string
		imageType vips.ImageType
	}{
		{name: "HEIC", probe: heicDecodeProbe, imageType: vips.ImageTypeHEIF},
		{name: "AVIF", probe: avifDecodeProbe, imageType: vips.ImageTypeAVIF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			probe, err := base64.StdEncoding.DecodeString(tt.probe)
			if err != nil {
				t.Fatalf("decode probe: %v", err)
			}
			if got := vips.DetermineImageType(probe); got != tt.imageType {
				t.Errorf("probe image type = %v, want %v", got, tt.imageType)
			}
		})
	}
}

func mimeTypeSet(mimeTypes ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(mimeTypes))
	for _, mimeType := range mimeTypes {
		set[mimeType] = struct{}{}
	}
	return set
}
