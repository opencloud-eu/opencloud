package web

import (
	"context"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/console/pkg/console"
)

const (
	ThemeID = "_console"
)

type CoreServiceOptions struct {
	Repository        Repository        `validate:"required"`
	ConsoleRepository ConsoleRepository `validate:"required"`
	Next              Service           `validate:"required"`
	Logger            log.Logger
}

func (o CoreServiceOptions) Validate() error {
	if err := console.Validate.Struct(o); err != nil {
		return fmt.Errorf("(%w): %w", console.ErrOptionsInvalid, err)
	}

	return nil
}

type CoreService struct {
	repository        Repository
	consoleRepository ConsoleRepository
	next              Service
	logger            log.Logger
}

func NewCoreService(o CoreServiceOptions) (CoreService, error) {
	if err := o.Validate(); err != nil {
		return CoreService{}, err
	}

	return CoreService{
		repository:        o.Repository,
		consoleRepository: o.ConsoleRepository,
		next:              o.Next,
		logger:            o.Logger,
	}, nil
}

func (s CoreService) ThemeApply(ctx context.Context) error {
	data, err := s.consoleRepository.ThemeGet(context.Background())
	if err != nil {
		return fmt.Errorf("could not get theme: %w", err)
	}
	defer func() {
		_ = data.Close()
	}()

	exists, err := s.repository.ThemeExists(ctx, ThemeID)
	if err != nil {
		return fmt.Errorf("could not check if theme %s exists: %w", ThemeID, err)
	}

	if exists {
		if err := s.repository.ThemeRemove(ctx, ThemeID); err != nil {
			return fmt.Errorf("could not remove existing theme %s: %w", ThemeID, err)
		}
	}

	if err := s.repository.ThemeAdd(ctx, ThemeID, data); err != nil {
		return fmt.Errorf("could not add theme %s: %w", ThemeID, err)
	}

	return s.next.ThemeApply(ctx)
}

func (s CoreService) ThemeRemove(ctx context.Context) error {
	exists, err := s.repository.ThemeExists(ctx, ThemeID)
	if err != nil {
		return fmt.Errorf("could not check if theme %s exists: %w", ThemeID, err)
	}

	if exists {
		if err := s.repository.ThemeRemove(ctx, ThemeID); err != nil {
			return fmt.Errorf("could not remove existing theme %s: %w", ThemeID, err)
		}
	}

	return s.next.ThemeRemove(ctx)
}
