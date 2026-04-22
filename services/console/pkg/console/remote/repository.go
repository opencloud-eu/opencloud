package remote

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/golang-jwt/jwt/v5"

	"github.com/opencloud-eu/opencloud/pkg/log"

	"github.com/opencloud-eu/opencloud/services/console/pkg/console"
)

type HTTPRepositoryOptions struct {
	Client     *http.Client `validate:"required"`
	Token      *jwt.Token   `validate:"required"`
	URLBuilder console.URLBuilder
	Claims     *console.Claims `validate:"required"`
	Logger     log.Logger
}

func (o HTTPRepositoryOptions) Validate() error {
	if err := console.Validate.Struct(o); err != nil {
		return fmt.Errorf("(%w): %w", console.ErrOptionsInvalid, err)
	}

	return nil
}

type HTTPRepository struct {
	client     *http.Client
	token      *jwt.Token
	claims     *console.Claims
	urlBuilder console.URLBuilder
	logger     log.Logger
}

func NewHTTPRepository(o HTTPRepositoryOptions) (HTTPRepository, error) {
	if err := o.Validate(); err != nil {
		return HTTPRepository{}, err
	}

	return HTTPRepository{
		client:     o.Client,
		token:      o.Token,
		urlBuilder: o.URLBuilder,
		claims:     o.Claims,
		logger:     o.Logger,
	}, nil
}

func (r HTTPRepository) ThemeGet(ctx context.Context) (io.ReadCloser, error) {
	req, err := console.NewHTTPRequest(http.MethodGet, r.urlBuilder.APIUrl("deployment", "theme").String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.SetBearerAuth(r.token.Raw)
	req.WithContext(ctx)

	res, err := r.client.Do(req.AsDefault())
	switch {
	case err != nil:
		return nil, fmt.Errorf("(%w) failed to execute http request: %w", console.ErrRequest, err)
	case res.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("(%w) failed to fetch theme, status code: %d", console.ErrRequest, res.StatusCode)
	}

	return res.Body, nil
}
