package web

import (
	"context"
	"fmt"
	"io"

	"github.com/golang-jwt/jwt/v5"

	"github.com/opencloud-eu/opencloud/services/console/pkg/console"
)

const (
	tenantIDClaim = "tenantId"
	themeID       = "_console"
)

type Repository interface {
	ThemeExists(ctx context.Context, id string) (bool, error)
	ThemeRemove(ctx context.Context, id string) error
	ThemeAdd(ctx context.Context, id string, r io.Reader) error
}

type ConsoleRepository interface {
	ThemeGet(ctx context.Context, token, tenantID string) (io.ReadCloser, error)
}

type ServiceOptions struct {
	Repository        Repository        `validate:"required"`
	ConsoleRepository ConsoleRepository `validate:"required"`
}

func (o ServiceOptions) Validate() error {
	if err := validate.Struct(o); err != nil {
		return fmt.Errorf("(%w): %w", ErrOptionsInvalid, err)
	}

	return nil
}

type Service struct {
	repository        Repository
	consoleRepository ConsoleRepository
}

func NewService(o ServiceOptions) (Service, error) {
	if err := o.Validate(); err != nil {
		return Service{}, err
	}

	return Service{
		repository: o.Repository,
		//consoleRepository: o.consoleRepository,
	}, nil
}

func (s Service) ThemeApply(ctx context.Context, token *jwt.Token) error {
	tenantId, err := console.GetJWTClaim[string](token, tenantIDClaim)
	if err != nil {
		return fmt.Errorf("could not get tenantId claim from token: %w", err)
	}

	data, err := s.consoleRepository.ThemeGet(context.Background(), token.Raw, tenantId)
	if err != nil {
		return fmt.Errorf("could not get theme for tenant %s: %w", tenantId, err)
	}
	defer func() {
		_ = data.Close()
	}()

	exists, err := s.repository.ThemeExists(ctx, themeID)
	if err != nil {
		return fmt.Errorf("could not check if theme %s exists: %w", themeID, err)
	}

	if exists {
		if err := s.repository.ThemeRemove(ctx, themeID); err != nil {
			return fmt.Errorf("could not remove existing theme %s: %w", themeID, err)
		}
	}

	if err := s.repository.ThemeAdd(ctx, themeID, data); err != nil {
		return fmt.Errorf("could not add theme %s: %w", themeID, err)
	}

	return nil
}
