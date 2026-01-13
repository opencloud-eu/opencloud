package log

import "github.com/opencloud-eu/opencloud/pkg/shared"

// Configure initializes a service-specific logger instance.
func Configure(name string, commons *shared.Commons, localServiceLogLevel string) Logger {
	return NewLogger(
		Name(name),
		Level(localServiceLogLevel),
		Pretty(commons.Log.Pretty),
		Color(commons.Log.Color),
		File(commons.Log.File),
	)
}
