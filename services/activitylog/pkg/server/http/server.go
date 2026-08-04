package http

import (
	"context"
	"embed"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/opencloud/pkg/account"
	"github.com/opencloud-eu/opencloud/pkg/cors"
	"github.com/opencloud-eu/opencloud/pkg/l10n"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	ohttp "github.com/opencloud-eu/opencloud/pkg/service/http"
	"github.com/opencloud-eu/opencloud/pkg/version"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"go-micro.dev/v4"
	"google.golang.org/grpc/metadata"
)

var (
	//go:embed l10n/locale
	_localeFS embed.FS

	// subfolder where the translation files are stored
	_localeSubPath = "l10n/locale"

	// domain of the activitylog service (transifex)
	_domain = "activitylog"
)

// Server initializes the http service and server.
func Server(opts ...Option) (ohttp.Service, error) {
	options := newOptions(opts...)
	service := options.Service

	newService, err := ohttp.NewService(
		ohttp.TLSConfig(options.Config.HTTP.TLS),
		ohttp.Logger(options.Logger),
		ohttp.Namespace(options.Config.HTTP.Namespace),
		ohttp.Name(options.Config.Service.Name),
		ohttp.Version(version.GetString()),
		ohttp.Address(options.Config.HTTP.Addr),
		ohttp.Context(options.Context),
		ohttp.Flags(options.Flags...),
	)
	if err != nil {
		options.Logger.Error().
			Err(err).
			Msg("Error initializing http service")
		return ohttp.Service{}, err
	}

	middlewares := []func(http.Handler) http.Handler{
		chimiddleware.RequestID,
		middleware.Version(
			options.Config.Service.Name,
			version.GetString(),
		),
		middleware.Logger(
			options.Logger,
		),
		middleware.TraceContext,
		middleware.ExtractAccountUUID(
			account.Logger(options.Logger),
			account.JWTSecret(options.Config.TokenManager.JWTSecret),
		),
		middleware.Cors(
			cors.Logger(options.Logger),
			cors.AllowedOrigins(options.Config.HTTP.CORS.AllowedOrigins),
			cors.AllowedMethods(options.Config.HTTP.CORS.AllowedMethods),
			cors.AllowedHeaders(options.Config.HTTP.CORS.AllowedHeaders),
			cors.AllowCredentials(options.Config.HTTP.CORS.AllowCredentials),
		),
	}

	mux := chi.NewMux()
	mux.Use(middlewares...)

	t := l10n.NewTranslatorFromCommonConfig(options.Config.DefaultLanguage, _domain, options.Config.TranslationPath, _localeFS, _localeSubPath)
	mux.Route(options.Config.HTTP.Root, func(r chi.Router) {
		r.Get("/graph/v1beta1/extensions/org.libregraph/activities", GetItemActivitiesHandler(options.Logger, service, options.ValueClient, t))
	})

	err = micro.RegisterHandler(newService.Server(), mux)
	if err != nil {
		options.Logger.Fatal().Err(err).Msg("failed to register the handler")
	}

	newService.Init()
	return newService, nil

}

// Service defines the business logic implementations need to provide.
type ActivityLogService interface {
	GetItemActivities(ctx context.Context, query, loc string, t l10n.Translator) ([]libregraph.Activity, error)
}

// GetActivitiesResponse is the response on GET activities requests
type GetActivitiesResponse struct {
	Activities []libregraph.Activity `json:"value"`
}

func GetItemActivitiesHandler(log log.Logger, s ActivityLogService, vc settingssvc.ValueService, t l10n.Translator) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = metadata.AppendToOutgoingContext(ctx, revactx.TokenHeader, r.Header.Get(revactx.TokenHeader))

		activeUser, ok := revactx.ContextGetUser(ctx)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		loc := l10n.MustGetUserLocale(ctx, activeUser.GetId().GetOpaqueId(), r.Header.Get(l10n.HeaderAcceptLanguage), vc)

		activities, err := s.GetItemActivities(ctx, r.URL.Query().Get("kql"), loc, t)
		if err != nil {
			log.Error().Err(err).Msg("error getting activities")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		res := GetActivitiesResponse{
			Activities: activities,
		}

		b, err := json.Marshal(res)
		if err != nil {
			log.Error().Err(err).Msg("error marshalling activities")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; odata.metadata=minimal")
		w.Header().Set("OData-Version", "4.0")
		if reqID := chimiddleware.GetReqID(ctx); reqID != "" {
			w.Header().Set("request-id", reqID)
		}
		w.Header().Set("Cache-Control", "no-cache")

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(b); err != nil {
			log.Error().Err(err).Msg("error writing response")
		}
	}
}
