package theme

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

// HTTPOptions defines the options to configure HTTP.
type HTTPOptions struct {
	service *Service
	logger  log.Logger
}

// WithService sets the service for HTTPOptions.
func (o HTTPOptions) WithService(s *Service) HTTPOptions {
	o.service = s
	return o
}

// WithLogger sets the logger for the Service.
func (o HTTPOptions) WithLogger(logger log.Logger) HTTPOptions {
	o.logger = logger
	return o
}

// validate validates the input parameters.
func (o HTTPOptions) validate() error {
	if o.service == nil {
		return errors.New("service is required")
	}

	return nil
}

type HTTP struct {
	service *Service
	logger  log.Logger
}

// NewHTTP initializes a new HTTP.
func NewHTTP(options HTTPOptions) (HTTP, error) {
	if err := options.validate(); err != nil {
		return HTTP{}, err
	}

	return HTTP(options), nil
}

// Get renders the theme for the given ID.
func (h HTTP) Get(w http.ResponseWriter, r *http.Request) {
	theme, err := h.service.Build(r.PathValue("id"))
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to merge themes")
		http.Error(w, ErrBuildingThemeFailed.Error(), http.StatusInternalServerError)
	}

	b, err := json.Marshal(theme)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to marshal theme")
		http.Error(w, ErrBuildingThemeFailed.Error(), http.StatusInternalServerError)
		return
	}

	if _, err = w.Write(b); err != nil {
		h.logger.Error().Err(err).Msg("failed to write response")
		http.Error(w, ErrBuildingThemeFailed.Error(), http.StatusInternalServerError)
		return
	}
}
