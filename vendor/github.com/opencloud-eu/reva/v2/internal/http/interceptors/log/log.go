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

package log

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/rs/zerolog"
)

// New returns a new HTTP middleware that logs HTTP requests and responses.
// TODO(labkode): maybe log to another file?
func New() func(http.Handler) http.Handler {
	return handler
}

// handler is a logging middleware
func handler(h http.Handler) http.Handler {
	return newLoggingHandler(h)
}

func newLoggingHandler(h http.Handler) http.Handler {
	return loggingHandler{handler: h}
}

type loggingHandler struct {
	handler http.Handler
}

func (h loggingHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log := appctx.GetLogger(req.Context())
	t := time.Now()
	logger := makeLogger(w)
	url := *req.URL
	h.handler.ServeHTTP(logger, req)
	writeLog(log, req, url, t, logger.Status(), logger.Size())
}

func makeLogger(w http.ResponseWriter) loggingResponseWriter {
	var logger loggingResponseWriter = &responseLogger{w: w, status: http.StatusOK}
	if _, ok := w.(http.Hijacker); ok {
		logger = &hijackLogger{responseLogger{w: w, status: http.StatusOK}}
	}
	h, ok1 := logger.(http.Hijacker)
	c, ok2 := w.(http.CloseNotifier)
	if ok1 && ok2 {
		return hijackCloseNotifier{logger, h, c}
	}
	if ok2 {
		return &closeNotifyWriter{logger, c}
	}
	return logger
}

func writeLog(log *zerolog.Logger, req *http.Request, url url.URL, ts time.Time, status, size int) {
	end := time.Now()
	host, _, err := net.SplitHostPort(req.RemoteAddr)

	if err != nil {
		host = req.RemoteAddr
	}

	uri := req.RequestURI
	u := req.URL.String()

	if req.ProtoMajor == 2 && req.Method == "CONNECT" {
		uri = req.Host
	}
	if uri == "" {
		uri = url.RequestURI()
	}

	diff := end.Sub(ts).Nanoseconds()

	var event *zerolog.Event
	switch {
	case status < 400:
		event = log.Debug()
	case status < 500:
		event = log.Warn()
	default:
		event = log.Error()
	}

	event.Str("host", host).Str("method", req.Method).
		Str("uri", uri).Str("url", u).Str("proto", req.Proto).Int("status", status).
		Int("size", size).
		Str("start", ts.Format("02/Jan/2006:15:04:05 -0700")).
		Str("end", end.Format("02/Jan/2006:15:04:05 -0700")).Int("time_ns", int(diff)).
		Msg("http")
}

type loggingResponseWriter interface {
	commonLoggingResponseWriter
	http.Pusher
}

func (l *responseLogger) Push(target string, opts *http.PushOptions) error {
	p, ok := l.w.(http.Pusher)
	if !ok {
		return fmt.Errorf("responseLogger does not implement http.Pusher")
	}
	return p.Push(target, opts)
}

type commonLoggingResponseWriter interface {
	http.ResponseWriter
	http.Flusher
	Status() int
	Size() int
	Unwrap() http.ResponseWriter
}

// responseLogger is wrapper of http.ResponseWriter that keeps track of its HTTP
// status code and body size
type responseLogger struct {
	w      http.ResponseWriter
	status int
	size   int
}

func (l *responseLogger) Header() http.Header {
	return l.w.Header()
}

func (l *responseLogger) Write(b []byte) (int, error) {
	size, err := l.w.Write(b)
	l.size += size
	return size, err
}

func (l *responseLogger) WriteHeader(s int) {
	l.w.WriteHeader(s)
	l.status = s
}

func (l *responseLogger) Status() int {
	return l.status
}

func (l *responseLogger) Size() int {
	return l.size
}

func (l *responseLogger) Flush() {
	f, ok := l.w.(http.Flusher)
	if ok {
		f.Flush()
	}
}

func (l *responseLogger) Unwrap() http.ResponseWriter {
	return l.w
}

type hijackLogger struct {
	responseLogger
}

func (l *hijackLogger) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h := l.responseLogger.w.(http.Hijacker)
	conn, rw, err := h.Hijack()
	if err == nil && l.responseLogger.status == 0 {
		l.responseLogger.status = http.StatusSwitchingProtocols
	}
	return conn, rw, err
}

type closeNotifyWriter struct {
	loggingResponseWriter
	http.CloseNotifier
}

type hijackCloseNotifier struct {
	loggingResponseWriter
	http.Hijacker
	http.CloseNotifier
}
