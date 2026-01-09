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

package tus

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"runtime"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	tusd "github.com/tus/tusd/v2/pkg/handler"
	"golang.org/x/exp/slog"

	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/datatx"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/datatx/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/datatx/metrics"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
)

func init() {
	registry.Register("tus", New)
}

type TusConfig struct {
	CorsEnabled          bool   `mapstructure:"cors_enabled"`
	CorsAllowOrigin      string `mapstructure:"cors_allow_origin"`
	CorsAllowCredentials bool   `mapstructure:"cors_allow_credentials"`
	CorsAllowMethods     string `mapstructure:"cors_allow_methods"`
	CorsAllowHeaders     string `mapstructure:"cors_allow_headers"`
	CorsMaxAge           string `mapstructure:"cors_max_age"`
	CorsExposeHeaders    string `mapstructure:"cors_expose_headers"`
}

type manager struct {
	conf      *TusConfig
	publisher events.Publisher
	log       *zerolog.Logger
}

func parseConfig(m map[string]interface{}) (*TusConfig, error) {
	c := &TusConfig{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	return c, nil
}

// New returns a datatx manager implementation that relies on HTTP PUT/GET.
func New(m map[string]interface{}, publisher events.Publisher, log *zerolog.Logger) (datatx.DataTX, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}

	l := log.With().Str("datatx", "tus").Logger()

	return &manager{
		conf:      c,
		publisher: publisher,
		log:       &l,
	}, nil
}

func (m *manager) Handler(fs storage.FS) (http.Handler, error) {
	composable, ok := fs.(storage.ComposableFS)
	if !ok {
		return nil, errtypes.NotSupported("file system does not support the tus protocol")
	}

	// A storage backend for tusd may consist of multiple different parts which
	// handle upload creation, locking, termination and so on. The composer is a
	// place where all those separated pieces are joined together. In this example
	// we only use the file store but you may plug in multiple.
	composer := tusd.NewStoreComposer()

	// let the composable storage tell tus which extensions it supports
	composable.UseIn(composer)

	config := tusd.Config{
		StoreComposer:         composer,
		NotifyCompleteUploads: true,
		Logger:                slog.New(tusdLogger{log: m.log}),
	}

	if m.conf.CorsEnabled {
		allowOrigin, err := regexp.Compile(m.conf.CorsAllowOrigin)
		if m.conf.CorsAllowOrigin != "" && err != nil {
			return nil, err
		}

		config.Cors = &tusd.CorsConfig{
			Disable:          false,
			AllowOrigin:      allowOrigin,
			AllowCredentials: m.conf.CorsAllowCredentials,
			AllowMethods:     m.conf.CorsAllowMethods,
			AllowHeaders:     m.conf.CorsAllowHeaders,
			MaxAge:           m.conf.CorsMaxAge,
			ExposeHeaders:    m.conf.CorsExposeHeaders,
		}
	}

	handler, err := tusd.NewUnroutedHandler(config)
	if err != nil {
		return nil, err
	}

	if usl, ok := fs.(storage.UploadSessionLister); ok {
		// We can currently only send updates if the fs is decomposedfs as we read very specific keys from the storage map of the tus info
		go func() {
			for {
				ev := <-handler.CompleteUploads
				// We should be able to get the upload progress with fs.GetUploadProgress, but currently tus will erase the info files
				// so we create a Progress instance here that is used to read the correct properties
				ups, err := usl.ListUploadSessions(context.Background(), storage.UploadSessionFilter{ID: &ev.Upload.ID})
				if err != nil {
					appctx.GetLogger(context.Background()).Error().Err(err).Str("session", ev.Upload.ID).Msg("failed to list upload session")
				} else {
					if len(ups) < 1 {
						appctx.GetLogger(context.Background()).Error().Str("session", ev.Upload.ID).Msg("upload session not found")
						continue
					}
					up := ups[0]
					executant := up.Executant()
					ref := up.Reference()
					if m.publisher != nil {
						if err := datatx.EmitFileUploadedEvent(up.SpaceOwner(), &executant, &ref, m.publisher); err != nil {
							appctx.GetLogger(context.Background()).Error().Err(err).Msg("failed to publish FileUploaded event")
						}
					}
				}
			}
		}()
	}

	h := handler.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sublog := m.log.With().Str("uploadid", r.URL.Path).Logger()
		r = r.WithContext(appctx.WithLogger(r.Context(), &sublog))
		method := r.Method
		// https://github.com/tus/tus-resumable-upload-protocol/blob/master/protocol.md#x-http-method-override
		if r.Header.Get("X-HTTP-Method-Override") != "" {
			method = r.Header.Get("X-HTTP-Method-Override")
		}

		switch method {
		case "POST":
			metrics.UploadsActive.Add(1)
			defer func() {
				metrics.UploadsActive.Sub(1)
			}()
			// set etag, mtime and file id
			setHeaders(fs, w, r)
			handler.PostFile(w, r)
		case "HEAD":
			handler.HeadFile(w, r)
		case "PATCH":
			metrics.UploadsActive.Add(1)
			defer func() {
				metrics.UploadsActive.Sub(1)
			}()
			// set etag, mtime and file id
			setHeaders(fs, w, r)
			handler.PatchFile(w, r)
		case "DELETE":
			handler.DelFile(w, r)
		case "GET":
			metrics.DownloadsActive.Add(1)
			defer func() {
				metrics.DownloadsActive.Sub(1)
			}()
			// NOTE: this is breaking change - allthought it does not seem to be used
			// We can make a switch here depending on some header value if that is needed
			// download.GetOrHeadFile(w, r, fs, "")
			handler.GetFile(w, r)
		default:
			w.WriteHeader(http.StatusNotImplemented)
		}
	}))

	return h, nil
}

func setHeaders(fs storage.FS, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := path.Base(r.URL.Path)
	datastore, ok := fs.(tusd.DataStore)
	if !ok {
		appctx.GetLogger(ctx).Error().Interface("fs", fs).Msg("storage is not a tus datastore")
		return
	}
	upload, err := datastore.GetUpload(ctx, id)
	if err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Msg("could not get upload from storage")
		return
	}
	info, err := upload.GetInfo(ctx)
	if err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Msg("could not get upload info for upload")
		return
	}
	expires := info.MetaData["expires"]
	if expires != "" {
		w.Header().Set(net.HeaderTusUploadExpires, expires)
	}
	resourceid := &provider.ResourceId{
		StorageId: info.MetaData["providerID"],
		SpaceId:   info.Storage["SpaceRoot"],
		OpaqueId:  info.Storage["NodeId"],
	}
	w.Header().Set(net.HeaderOCFileID, storagespace.FormatResourceID(resourceid))
}

// tusdLogger is a logger implementation (slog) for tusd that uses zerolog.
type tusdLogger struct {
	log *zerolog.Logger
}

// Handle handles the record
func (l tusdLogger) Handle(_ context.Context, r slog.Record) error {
	var logev *zerolog.Event
	switch r.Level {
	case slog.LevelDebug:
		logev = l.log.Debug()
	case slog.LevelInfo:
		logev = l.log.Info()
	case slog.LevelWarn:
		logev = l.log.Warn()
	case slog.LevelError:
		logev = l.log.Error()
	}
	// Extract caller information from slog.Record only for debug and info levels
	if (r.Level == slog.LevelDebug || r.Level == slog.LevelInfo) && r.PC != 0 {
		frames := runtime.CallersFrames([]uintptr{r.PC})
		frame, _ := frames.Next()
		// add line using zerolog's caller format
		logev = logev.Str("line", fmt.Sprintf("%s:%d", frame.File, frame.Line))
	}
	r.Attrs(func(a slog.Attr) bool {
		// Resolve the Attr's value before doing anything else.
		a.Value = a.Value.Resolve()
		// Ignore empty Attrs.
		if a.Equal(slog.Attr{}) {
			return true
		}
		switch a.Value.Kind() {
		case slog.KindBool:
			logev = logev.Bool(a.Key, a.Value.Bool())
		case slog.KindInt64:
			logev = logev.Int64(a.Key, a.Value.Int64())
		default:
			logev = logev.Str(a.Key, a.Value.String())
		}
		return true
	})
	logev.Msg(r.Message)
	return nil
}

// Enabled returns true
func (l tusdLogger) Enabled(_ context.Context, _ slog.Level) bool { return true }

// WithAttrs creates a new logger with the given attributes
func (l tusdLogger) WithAttrs(attr []slog.Attr) slog.Handler {
	fields := make(map[string]interface{}, len(attr))
	for _, a := range attr {
		switch a.Value.Kind() {
		case slog.KindBool:
			fields[a.Key] = a.Value.Bool()
		case slog.KindInt64:
			fields[a.Key] = a.Value.Int64()
		case slog.KindUint64:
			fields[a.Key] = a.Value.Uint64()
		case slog.KindFloat64:
			fields[a.Key] = a.Value.Float64()
		default:
			fields[a.Key] = a.Value.String()
		}
	}
	c := l.log.With().Fields(fields).Logger()
	sLog := tusdLogger{log: &c}
	return sLog
}

// WithGroup is not implemented
func (l tusdLogger) WithGroup(name string) slog.Handler { return l }
