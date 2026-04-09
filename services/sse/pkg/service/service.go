package service

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Service struct {
	handler http.Handler
}

func New(mux *chi.Mux, sseHandler http.Handler) (Service, error) {
	mux.Get("/ocs/v2.php/apps/notifications/api/v1/notifications/sse", sseHandler.ServeHTTP)

	return Service{
		handler: mux,
	}, nil
}

func (s Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}
