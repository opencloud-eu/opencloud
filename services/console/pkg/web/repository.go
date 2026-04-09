package web

import (
	"context"
	"fmt"
	"io"

	"github.com/opencloud-eu/opencloud/pkg/log"
	webService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/web/v0"
	"github.com/opencloud-eu/opencloud/services/console/pkg/console"
)

type GRPCRepositoryOptions struct {
	WebService webService.WebService `validate:"required"`
	Logger     log.Logger
}

func (o GRPCRepositoryOptions) Validate() error {
	if err := console.Validate.Struct(o); err != nil {
		return fmt.Errorf("(%w): %w", console.ErrOptionsInvalid, err)
	}

	return nil
}

type GRPCRepository struct {
	webService webService.WebService
	logger     log.Logger
}

func NewGRPCRepository(o GRPCRepositoryOptions) (GRPCRepository, error) {
	if err := o.Validate(); err != nil {
		return GRPCRepository{}, err
	}

	return GRPCRepository{
		webService: o.WebService,
		logger:     o.Logger,
	}, nil
}

func (r GRPCRepository) ThemeExists(ctx context.Context, id string) (bool, error) {
	resp, err := r.webService.ThemeExists(ctx, &webService.ThemeExistsRequest{Id: id})
	if err != nil {
		return false, fmt.Errorf("(%w) %w", console.ErrRequest, err)
	}

	return resp.Exists, nil
}

func (r GRPCRepository) ThemeAdd(ctx context.Context, id string, tr io.Reader) error {
	tb, err := io.ReadAll(tr)
	if err != nil {
		return fmt.Errorf("failed to read theme data: %w", err)
	}

	if _, err := r.webService.ThemeAdd(ctx, &webService.ThemeAddRequest{Id: id, Data: tb}); err != nil {
		return fmt.Errorf("(%w) %w", console.ErrRequest, err)
	}

	return nil
}

func (r GRPCRepository) ThemeRemove(ctx context.Context, id string) error {
	if _, err := r.webService.ThemeRemove(ctx, &webService.ThemeRemoveRequest{Id: id}); err != nil {
		return fmt.Errorf("(%w) %w", console.ErrRequest, err)
	}

	return nil
}
