// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/opencloud-eu/opencloud/pkg/cors"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	ohttp "github.com/opencloud-eu/opencloud/pkg/service/http"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"go-micro.dev/v4"
)

// Server initializes the http service and server.
func Server(opts ...Option) (ohttp.Service, error) {
	options := newOptions(opts...)

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

	mux.Route(options.Config.HTTP.Root, func(r chi.Router) {
		r.Post("/v1beta1/extensions/org.libregraph/guestLinks/redeem", RedeemHandler(options.Logger, options.Service, options.Config))
		r.Post("/v1beta1/extensions/org.libregraph/guestLinks/verify/token", RedeemHandler(options.Logger, options.Service, options.Config))
		r.Post("/v1beta1/extensions/org.libregraph/guestLinks/renew", RenewHandler(options.Logger, options.Service))
		r.Post("/v1beta1/extensions/org.libregraph/guestLinks/verify/pin", VerifyPinHandler(options.Logger, options.Service, options.Config))
	})

	err = micro.RegisterHandler(newService.Server(), mux)
	if err != nil {
		options.Logger.Fatal().Err(err).Msg("failed to register the handler")
	}

	newService.Init()
	return newService, nil
}
