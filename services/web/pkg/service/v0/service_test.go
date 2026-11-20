package svc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/web/mocks"
	"github.com/opencloud-eu/opencloud/services/web/pkg/announcement"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
)

func TestCurrentAnnouncement(t *testing.T) {
	newWeb := func(store *announcement.Store) Web {
		return Web{logger: log.NopLogger(), announcementStore: store}
	}
	// storeReturning builds a store whose backing bucket returns the given JSON for the announcement key.
	storeReturning := func(t *testing.T, value string) *announcement.Store {
		entry := mocks.NewKeyValueEntry(t)
		entry.EXPECT().Value().Return([]byte(value))
		kv := mocks.NewKeyValue(t)
		kv.EXPECT().Get(mock.Anything, "announcement").Return(entry, nil)
		return announcement.NewStore(kv)
	}
	// emptyStore builds a store whose backing bucket has no announcement.
	emptyStore := func(t *testing.T) *announcement.Store {
		kv := mocks.NewKeyValue(t)
		kv.EXPECT().Get(mock.Anything, "announcement").Return(nil, jetstream.ErrKeyNotFound)
		return announcement.NewStore(kv)
	}

	t.Run("nil when there is no store", func(t *testing.T) {
		require.Nil(t, newWeb(nil).currentAnnouncement(context.Background()))
	})

	t.Run("nil when the store is empty", func(t *testing.T) {
		require.Nil(t, newWeb(emptyStore(t)).currentAnnouncement(context.Background()))
	})

	t.Run("nil when disabled", func(t *testing.T) {
		s := storeReturning(t, `{"enabled":false,"bannerText":"hi","infoText":"info"}`)
		require.Nil(t, newWeb(s).currentAnnouncement(context.Background()))
	})

	t.Run("nil when enabled but the banner text is empty", func(t *testing.T) {
		s := storeReturning(t, `{"enabled":true,"bannerText":"","infoText":"info"}`)
		require.Nil(t, newWeb(s).currentAnnouncement(context.Background()))
	})

	t.Run("returns banner and info text when enabled with a banner text", func(t *testing.T) {
		s := storeReturning(t, `{"enabled":true,"bannerText":"hi","infoText":"info"}`)
		require.Equal(t, &config.Announcement{BannerText: "hi", InfoText: "info"}, newWeb(s).currentAnnouncement(context.Background()))
	})
}

func TestGetPayloadServer(t *testing.T) {
	for _, tt := range []struct {
		name   string
		server string
		want   any
	}{
		// An empty server must stay unset so the client falls back to the browser origin.
		{name: "empty", server: "", want: nil},
		{name: "without trailing slash", server: "https://cloud.example.com", want: "https://cloud.example.com/"},
		{name: "with trailing slashes", server: "https://cloud.example.com///", want: "https://cloud.example.com/"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Web.Config.Server = tt.server
			payload, err := Web{logger: log.NopLogger(), config: cfg}.getPayload(context.Background())
			require.NoError(t, err)

			var got map[string]any
			require.NoError(t, json.Unmarshal(payload, &got))
			require.Equal(t, tt.want, got["server"])
		})
	}
}
