// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

func init() {
	zerolog.CallerSkipFrameCount = 2
	zerolog.TimeFieldFormat = time.RFC3339Nano
}

// Mode changes the logging format.
type Mode string

const (
	// JSONMode outputs JSON.
	JSONMode Mode = "json"
	// ConsoleMode outputs human-readable logs.
	ConsoleMode Mode = "console"
)

// Option is the option to use to configure the logger.
type Option func(l *zerolog.Logger)

// New creates a new logger.
func New(opts ...Option) *zerolog.Logger {
	// create a default logger
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	zl := zerolog.New(os.Stderr).With().Timestamp().Caller().Logger()
	for _, opt := range opts {
		opt(&zl)
	}
	return &zl
}

// WithLevel is an option to configure the logging level.
func WithLevel(lvl string) Option {
	return func(l *zerolog.Logger) {
		zlvl := parseLevel(lvl)
		*l = l.Level(zlvl)
	}
}

// WithWriter is an option to configure the logging output.
func WithWriter(w io.Writer, m Mode) Option {
	return func(l *zerolog.Logger) {
		if m == ConsoleMode {
			*l = l.Output(zerolog.ConsoleWriter{Out: w, TimeFormat: "2006-01-02 15:04:05.999"})
		} else {
			*l = l.Output(w)
		}
	}
}

func parseLevel(v string) zerolog.Level {
	if v == "" {
		return zerolog.InfoLevel
	}

	lvl, err := zerolog.ParseLevel(v)
	if err != nil {
		return zerolog.InfoLevel
	}

	return lvl
}

func InitLoggerOrDie(v interface{}, logLevel string) *zerolog.Logger {
	conf := ParseLogConfOrDie(v, logLevel)
	log, err := fromConfig(conf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating logger, exiting ...")
		os.Exit(1)
	}
	return log
}

func ParseLogConfOrDie(v interface{}, logLevel string) *LogConf {
	c := &LogConf{}
	if err := mapstructure.Decode(v, c); err != nil {
		fmt.Fprintf(os.Stderr, "error decoding log config: %s\n", err.Error())
		os.Exit(1)
	}

	// if mode is not set, we use console mode, easier for devs
	if c.Mode == "" {
		c.Mode = "console"
	}

	// Give priority to the log level passed through the command line.
	if logLevel != "" {
		c.Level = logLevel
	}

	return c
}

type LogConf struct {
	Output string `mapstructure:"output"`
	Mode   string `mapstructure:"mode"`
	Level  string `mapstructure:"level"`
}

func fromConfig(conf *LogConf) (*zerolog.Logger, error) {
	if conf.Level == "" {
		conf.Level = zerolog.InfoLevel.String()
	}

	var opts []Option
	opts = append(opts, WithLevel(conf.Level))

	w, err := getWriter(conf.Output)
	if err != nil {
		return nil, err
	}

	opts = append(opts, WithWriter(w, Mode(conf.Mode)))

	l := New(opts...)
	sub := l.With().Int("pid", os.Getpid()).Logger()
	return &sub, nil
}

func getWriter(out string) (io.Writer, error) {
	if out == "stderr" || out == "" {
		return os.Stderr, nil
	}

	if out == "stdout" {
		return os.Stdout, nil
	}

	fd, err := os.OpenFile(out, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		err = errors.Wrap(err, "error creating log file: "+out)
		return nil, err
	}

	return fd, nil
}
