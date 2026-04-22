package log

import (
	"io"

	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
)

type levelMap map[logrus.Level]zerolog.Level

var levelMapping = levelMap{
	logrus.PanicLevel: zerolog.PanicLevel,
	logrus.ErrorLevel: zerolog.ErrorLevel,
	logrus.TraceLevel: zerolog.TraceLevel,
	logrus.DebugLevel: zerolog.DebugLevel,
	logrus.WarnLevel:  zerolog.WarnLevel,
	logrus.InfoLevel:  zerolog.InfoLevel,
}

// LogrusWrapper around zerolog. Required because idp uses logrus internally.
type LogrusWrapper struct {
	zeroLog  *zerolog.Logger
	levelMap levelMap
}

// LogrusWrap returns a logrus logger which internally logs to /dev/null. Messages are passed to the
// underlying zerolog via hooks.
func LogrusWrap(zr zerolog.Logger) *logrus.Logger {
	lr := logrus.New()
	lr.SetOutput(io.Discard)
	lr.SetLevel(logrusLevel(zr.GetLevel()))
	lr.AddHook(&LogrusWrapper{
		zeroLog:  &zr,
		levelMap: levelMapping,
	})

	return lr
}

// Levels on which logrus hooks should be triggered
func (h *LogrusWrapper) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire called by logrus on new message
func (h *LogrusWrapper) Fire(entry *logrus.Entry) error {
	h.zeroLog.WithLevel(h.levelMap[entry.Level]).
		Fields(zeroLogFields(entry.Data)).
		Msg(entry.Message)

	return nil
}

// Convert logrus fields to zerolog
func zeroLogFields(fields logrus.Fields) map[string]any {
	fm := make(map[string]any)
	for k, v := range fields {
		fm[k] = v
	}

	return fm
}

// Convert logrus level to zerolog
func logrusLevel(level zerolog.Level) logrus.Level {
	for lrLvl, zrLvl := range levelMapping {
		if zrLvl == level {
			return lrLvl
		}
	}

	panic("Unexpected loglevel")
}
