package service

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/log"
	websvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/web/v0"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
	"github.com/opencloud-eu/opencloud/services/web/pkg/theme"
)

// NewHandler returns a service implementation for Service.
func NewHandler(opts ...Option) (websvc.WebServiceHandler, error) {
	options := newOptions(opts...)
	logger := options.Logger
	cfg := options.Config

	themeService, err := theme.NewService(
		theme.ServiceOptions{}.
			WithThemeFS(options.ThemeFS.Primary()),
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize theme service: %w", err)
	}

	return &Service{
		log:          logger,
		cfg:          cfg,
		themeService: themeService,
	}, nil
}

type Service struct {
	log          log.Logger
	cfg          *config.Config
	themeService *theme.Service
}

func (s Service) ThemeAdd(_ context.Context, req *websvc.ThemeAddRequest, res *websvc.ThemeAddResponse) error {
	zr, err := zip.NewReader(bytes.NewReader(req.Data), int64(len(req.Data)))
	if err != nil {
		return err
	}

	if err := s.themeService.Add(req.Id, zr); err != nil {
		return fmt.Errorf("could not add theme %s: %w", req.Id, err)
	}

	return nil
}

func (s Service) ThemeRemove(_ context.Context, req *websvc.ThemeRemoveRequest, res *websvc.ThemeRemoveResponse) error {
	if err := s.themeService.Remove(req.Id); err != nil {
		return fmt.Errorf("could not remove theme %s: %w", req.Id, err)
	}

	return nil
}

func (s Service) ThemeExists(_ context.Context, req *websvc.ThemeExistsRequest, res *websvc.ThemeExistsResponse) error {
	res.Exists = s.themeService.Exists(req.Id)

	return nil
}
