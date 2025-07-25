package svc

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/riandyrn/otelchi"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/config"
	tjwt "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/service/jwt"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail"
)

type contextKey string

const (
	keyContextKey contextKey = "key"
)

// Service defines the service handlers.
type Service interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	GetThumbnail(w http.ResponseWriter, r *http.Request)
}

// NewService returns a service implementation for Service.
func NewService(opts ...Option) Service {
	options := newOptions(opts...)

	m := chi.NewMux()
	m.Use(options.Middleware...)

	m.Use(
		otelchi.Middleware(
			"thumbnails",
			otelchi.WithChiRoutes(m),
			otelchi.WithTracerProvider(options.TraceProvider),
			otelchi.WithPropagators(tracing.GetPropagator()),
		),
	)

	logger := options.Logger
	resolutions, err := thumbnail.ParseResolutions(options.Config.Thumbnail.Resolutions)
	if err != nil {
		logger.Fatal().Err(err).Msg("resolutions not configured correctly")
	}
	svc := Thumbnails{
		config: options.Config,
		mux:    m,
		logger: options.Logger,
		manager: thumbnail.NewSimpleManager(
			resolutions,
			options.ThumbnailStorage,
			logger,
			options.Config.Thumbnail.MaxInputWidth,
			options.Config.Thumbnail.MaxInputHeight,
		),
	}

	m.Route(options.Config.HTTP.Root, func(r chi.Router) {
		r.Use(svc.TransferTokenValidator)
		r.Get("/data", svc.GetThumbnail)
	})

	_ = chi.Walk(m, func(method string, route string, _ http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		options.Logger.Debug().Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})

	return svc
}

// Thumbnails implements the business logic for Service.
type Thumbnails struct {
	config  *config.Config
	logger  log.Logger
	mux     *chi.Mux
	manager thumbnail.Manager
}

// ServeHTTP implements the Service interface.
func (s Thumbnails) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// GetThumbnail implements the Service interface.
func (s Thumbnails) GetThumbnail(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.SubloggerWithRequestID(r.Context())
	key := r.Context().Value(keyContextKey).(string)

	thumbnailBytes, err := s.manager.GetThumbnail(key)
	if err != nil {
		logger.Debug().
			Err(err).
			Str("key", key).
			Msg("could not get the thumbnail")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Length", strconv.Itoa(len(thumbnailBytes)))
	if _, err = w.Write(thumbnailBytes); err != nil {
		logger.Error().
			Err(err).
			Str("key", key).
			Msg("could not write the thumbnail response")
	}
}

// TransferTokenValidator validates a transfer token
func (s Thumbnails) TransferTokenValidator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Transfer-Token")
		if tokenString == "" {
			next.ServeHTTP(w, r)
			return
		}

		logger := s.logger.SubloggerWithRequestID(r.Context())
		token, err := jwt.ParseWithClaims(tokenString, &tjwt.ThumbnailClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(s.config.Thumbnail.TransferSecret), nil
		})
		if err != nil {
			logger.Debug().
				Err(err).
				Str("transfer-token", tokenString).
				Msg("failed to parse transfer token")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(*tjwt.ThumbnailClaims); ok && token.Valid {
			ctx := context.WithValue(r.Context(), keyContextKey, claims.Key)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		logger.Debug().Msg("invalid transfer token")
		w.WriteHeader(http.StatusUnauthorized)
	})
}
