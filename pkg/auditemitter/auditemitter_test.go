package auditemitter

import (
	"context"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/reva/v3/pkg/audit"
	"github.com/opencloud-eu/reva/v3/pkg/events/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	logger := log.NopLogger()

	t.Run("off", func(t *testing.T) {
		e, err := New(context.Background(), "graph", Settings{}, raw.Config{}, logger)
		require.NoError(t, err)
		assert.IsType(t, audit.Discard{}, e)
		assert.NotPanics(t, func() { audit.CloseEmitter(e) })
	})

	t.Run("without endpoint", func(t *testing.T) {
		_, err := New(context.Background(), "graph", Settings{Enabled: true}, raw.Config{}, logger)
		assert.ErrorIs(t, err, audit.ErrNoEndpoint)
	})

	t.Run("unknown mode", func(t *testing.T) {
		_, err := New(context.Background(), "graph", Settings{Enabled: true, Mode: "strict"}, raw.Config{Endpoint: "127.0.0.1:9233"}, logger)
		assert.ErrorIs(t, err, audit.ErrUnknownMode)
	})
}
