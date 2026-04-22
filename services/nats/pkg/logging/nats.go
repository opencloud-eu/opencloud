package logging

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

func NewLogWrapper(logger log.Logger) *LogWrapper {
	return &LogWrapper{logger}
}

// we need to wrap our logger so we can pass it to the nats server
type LogWrapper struct {
	logger log.Logger
}

// Noticef logs a notice statement
func (l *LogWrapper) Noticef(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.logger.Info().Msg(msg)
}

// Warnf logs a warning statement
func (l *LogWrapper) Warnf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.logger.Warn().Msg(msg)
}

// Fatalf logs a fatal statement
func (l *LogWrapper) Fatalf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.logger.Fatal().Msg(msg)
}

// Errorf logs an error statement
func (l *LogWrapper) Errorf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.logger.Error().Msg(msg)
}

// Debugf logs a debug statement
func (l *LogWrapper) Debugf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.logger.Debug().Msg(msg)
}

// Tracef logs a trace statement
func (l *LogWrapper) Tracef(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.logger.Trace().Msg(msg)
}
