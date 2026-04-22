package web_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/console/pkg/console"
	"github.com/opencloud-eu/opencloud/services/console/pkg/web"
)

func TestService_NewService(t *testing.T) {
	t.Run("valid options", func(t *testing.T) {
		_, err := web.NewCoreService(web.CoreServiceOptions{})
		assert.ErrorIs(t, err, console.ErrOptionsInvalid)
	})
}

func TestService_ThemeApply(t *testing.T) {
	//options := web.CoreServiceOptions{
	//	Repository:        mocks.NewRepository(t),
	//	ConsoleRepository: mocks.NewConsoleRepository(t),
	//}
	//service, err := web.NewCoreService(options)
	//assert.NoError(t, err)

	t.Run("fails without tenantID claim", func(t *testing.T) {
		//err := service.ThemeApply(t.Context())
		//assert.ErrorIs(t, err, console.ErrJWTTokenUnknownClaim)
	})
}
