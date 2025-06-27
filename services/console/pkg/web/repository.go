package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	webService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/web/v0"
	"github.com/opencloud-eu/opencloud/services/console/pkg/console"
)

type GRPCRepositoryOptions struct {
	WebService webService.WebService `validate:"required"`
}

func (o GRPCRepositoryOptions) Validate() error {
	if err := validate.Struct(o); err != nil {
		return fmt.Errorf("(%w): %w", ErrOptionsInvalid, err)
	}

	return nil
}

type GRPCRepository struct {
	webService webService.WebService
}

func NewGRPCRepository(o GRPCRepositoryOptions) (GRPCRepository, error) {
	if err := o.Validate(); err != nil {
		return GRPCRepository{}, err
	}

	return GRPCRepository{
		webService: o.WebService,
	}, nil
}

func (r GRPCRepository) ThemeExists(ctx context.Context, id string) (bool, error) {
	resp, err := r.webService.ThemeExists(ctx, &webService.ThemeExistsRequest{Id: id})
	if err != nil {
		return false, fmt.Errorf("(%w) %w", ErrRequest, err)
	}

	return resp.Exists, nil
}

func (r GRPCRepository) ThemeAdd(ctx context.Context, id string, tr io.Reader) error {
	tb, err := io.ReadAll(tr)
	if err != nil {
		return fmt.Errorf("failed to read theme data: %w", err)
	}

	if _, err := r.webService.ThemeAdd(ctx, &webService.ThemeAddRequest{Id: id, Data: tb}); err != nil {
		return fmt.Errorf("(%w) %w", ErrRequest, err)
	}

	return nil
}

func (r GRPCRepository) ThemeRemove(ctx context.Context, id string) error {
	if _, err := r.webService.ThemeRemove(ctx, &webService.ThemeRemoveRequest{Id: id}); err != nil {
		return fmt.Errorf("(%w) %w", ErrRequest, err)
	}

	return nil
}

// ################################

type ConsoleHTTPRepositoryOptions struct {
	HTTPClient     *http.Client `validate:"required"`
	ConsoleAPIRoot string       `validate:"http_url"`
}

func (o ConsoleHTTPRepositoryOptions) Validate() error {
	if err := validate.Struct(o); err != nil {
		return fmt.Errorf("(%w): %w", ErrOptionsInvalid, err)
	}

	return nil
}

type ConsoleHTTPRepository struct {
	httpClient     *http.Client `validate:"required"`
	consoleAPIRoot string       `validate:"http_url"`
}

func NewConsoleHTTPRepository(o ConsoleHTTPRepositoryOptions) (ConsoleHTTPRepository, error) {
	if err := o.Validate(); err != nil {
		return ConsoleHTTPRepository{}, err
	}

	return ConsoleHTTPRepository{
		httpClient:     o.HTTPClient,
		consoleAPIRoot: o.ConsoleAPIRoot,
	}, nil
}

func (r ConsoleHTTPRepository) ThemeGet(ctx context.Context, bearer, tenantID string) (io.ReadCloser, error) {
	u, err := url.Parse(r.consoleAPIRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to parse console api root url: %v", err)
	}

	req, err := console.NewHTTPRequest(http.MethodGet, u.JoinPath("tenant", tenantID, "client", "v1", "theme").String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.SetBearerAuth(bearer)
	req.WithContext(ctx)

	res, err := r.httpClient.Do(req.AsDefault())
	switch {
	case err != nil:
		return nil, fmt.Errorf("(%w) failed to execute http request: %w", ErrRequest, err)
	case res.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("(%w) failed to fetch theme, status code: %d", ErrRequest, res.StatusCode)
	}

	return res.Body, nil
}
