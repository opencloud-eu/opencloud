// Package config contains the configuration for the opencloud-thumbnails service
package config

import (
	"context"

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

// Thumbnail holds the safety limits enforced by the push endpoint. The limits
// mirror the legacy thumbnail service so a request that was rejected before is
// still rejected, but the enforcement now lives at decode time in this service
// (which owns the image) rather than in webdav.
type Thumbnail struct {
	// MaxInputWidth is the maximum width of an input image which is being processed.
	MaxInputWidth int `yaml:"max_input_width" env:"THUMBNAILS_MAX_INPUT_WIDTH" desc:"The maximum width of an input image which is being processed." introductionVersion:"1.0.0"`
	// MaxInputHeight is the maximum height of an input image which is being processed.
	MaxInputHeight int `yaml:"max_input_height" env:"THUMBNAILS_MAX_INPUT_HEIGHT" desc:"The maximum height of an input image which is being processed." introductionVersion:"1.0.0"`
}
