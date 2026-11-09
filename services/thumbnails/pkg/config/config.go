// Package config contains the configuration for the opencloud-thumbnails service
package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/opencloud-eu/reva/v2/pkg/bytesize"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	LogLevel string `yaml:"loglevel" env:"OC_LOG_LEVEL;THUMBNAILS_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Debug    Debug  `yaml:"debug"`

	HTTP HTTP `yaml:"http"`

	Thumbnail Thumbnail `yaml:"thumbnail"`

	Context context.Context `yaml:"-"`
}

// Thumbnail holds the safety limits enforced by the push endpoint. This service
// owns the image, so it is the one that can check a source before decoding it;
// webdav decides what size is asked for.
type Thumbnail struct {
	MaxInputWidth   int `yaml:"max_input_width" env:"THUMBNAILS_MAX_INPUT_WIDTH" desc:"The maximum width of an input image which is being processed." introductionVersion:"1.0.0"`
	MaxOutputWidth  int `yaml:"max_output_width" env:"THUMBNAILS_MAX_OUTPUT_WIDTH" desc:"The maximum width of a thumbnail that may be asked for. A larger box is refused with HTTP 422. webdav never asks for more than its largest configured resolution, so this bounds everything else that reaches the endpoint." introductionVersion:"9.0.0"`
	MaxOutputHeight int `yaml:"max_output_height" env:"THUMBNAILS_MAX_OUTPUT_HEIGHT" desc:"The maximum height of a thumbnail that may be asked for." introductionVersion:"9.0.0"`
	MaxInputHeight  int `yaml:"max_input_height" env:"THUMBNAILS_MAX_INPUT_HEIGHT" desc:"The maximum height of an input image which is being processed." introductionVersion:"1.0.0"`
	// 0 (the default) means unlimited; a request arriving while the limit is
	// reached gets HTTP 429 with a Retry-After.
	MaxConcurrentRequests int    `yaml:"max_concurrent_requests" env:"THUMBNAILS_MAX_CONCURRENT_REQUESTS" desc:"Maximum number of concurrent thumbnail generation requests. Default is 0 which is unlimited." introductionVersion:"9.0.0"`
	MaxInputImageFileSize string `yaml:"max_input_image_file_size" env:"OC_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE;THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE" desc:"The maximum size of an image posted to the generator. Usable common abbreviations: [KB, KiB, MB, MiB, GB, GiB], example: 50MB. An empty value means unlimited." introductionVersion:"1.0.0"`
	// together with the resolution limits this bounds the memory one request can ask for
	MaxAnimationFrames int `yaml:"max_animation_frames" env:"THUMBNAILS_MAX_ANIMATION_FRAMES" desc:"The maximum number of frames an animated image may have. A file with more frames is refused before any of them is decoded. Default is 100, 0 is unlimited." introductionVersion:"9.0.0"`
}

// MaxInputBytes is the configured maximum size of a posted image, 0 for unlimited.
func (t Thumbnail) MaxInputBytes() (uint64, error) {
	if strings.TrimSpace(t.MaxInputImageFileSize) == "" {
		return 0, nil
	}

	size, err := bytesize.Parse(t.MaxInputImageFileSize)
	if err != nil {
		return 0, fmt.Errorf("invalid THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE %q: %w", t.MaxInputImageFileSize, err)
	}

	return size.Bytes(), nil
}
