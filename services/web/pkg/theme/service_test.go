package theme_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
	"github.com/opencloud-eu/opencloud/services/web/pkg/theme"
)

func TestNewService(t *testing.T) {
	t.Run("fails if the options are invalid", func(t *testing.T) {
		_, err := theme.NewService(theme.ServiceOptions{})
		assert.Error(t, err)
	})

	t.Run("success if the options are valid", func(t *testing.T) {
		_, err := theme.NewService(
			theme.ServiceOptions{}.
				WithThemeFS(fsx.NewFallbackFS(fsx.NewMemMapFs(), fsx.NewMemMapFs())),
		)
		assert.NoError(t, err)
	})
}
