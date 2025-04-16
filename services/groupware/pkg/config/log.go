package config

// Log defines the available log configuration.
type Log struct {
	Level  string `mapstructure:"level" env:"OC_LOG_LEVEL;THUMBNAILS_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Pretty bool   `mapstructure:"pretty" env:"OC_LOG_PRETTY;THUMBNAILS_LOG_PRETTY" desc:"Activates pretty log output." introductionVersion:"1.0.0"`
	Color  bool   `mapstructure:"color" env:"OC_LOG_COLOR;THUMBNAILS_LOG_COLOR" desc:"Activates colorized log output." introductionVersion:"1.0.0"`
	File   string `mapstructure:"file" env:"OC_LOG_FILE;THUMBNAILS_LOG_FILE" desc:"The path to the log file. Activates logging to this file if set." introductionVersion:"1.0.0"`
}
