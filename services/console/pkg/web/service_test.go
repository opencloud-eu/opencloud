package web_test

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/console/mocks"
	"github.com/opencloud-eu/opencloud/services/console/pkg/console"
	"github.com/opencloud-eu/opencloud/services/console/pkg/web"
)

func TestService_NewService(t *testing.T) {
	t.Run("valid options", func(t *testing.T) {
		_, err := web.NewService(web.ServiceOptions{})
		assert.ErrorIs(t, err, web.ErrOptionsInvalid)
	})
}

func TestService_ThemeApply(t *testing.T) {
	options := web.ServiceOptions{
		Repository:        mocks.NewRepository(t),
		ConsoleRepository: mocks.NewConsoleRepository(t),
	}
	service, err := web.NewService(options)
	assert.NoError(t, err)

	t.Run("fails without tenantID claim", func(t *testing.T) {
		err := service.ThemeApply(t.Context(), jwt.New(jwt.SigningMethodHS256))
		assert.ErrorIs(t, err, console.ErrJWTTokenUnknownClaim)
	})
}
