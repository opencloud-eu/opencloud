package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	ehmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/eventhistory/v0"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestDeliveryFailureLogsEvent(t *testing.T) {
	for _, failure := range []string{"lookup", "SMTP"} {
		t.Run(failure, func(t *testing.T) {
			n, g, _, ch := newDeliveryFixture()
			var output bytes.Buffer
			n.logger = log.Logger{Logger: zerolog.New(&output)}
			if failure == "lookup" {
				g.lookupErr = errors.New("directory offline")
			} else {
				ch.err = errors.New("SMTP offline")
			}
			n.handleSpaceShared(events.SpaceShared{Executant: &user.UserId{OpaqueId: "alice"}, GranteeUserID: g.recipient.Id, ID: &provider.StorageSpaceId{OpaqueId: "storage!space"}}, "share-event")
			require.Empty(t, ch.messages)
			var entry map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &entry))
			require.Equal(t, "error", entry["level"])
			require.NotEmpty(t, entry["error"])
			require.Equal(t, "SpaceShared", entry["event"])
			require.Equal(t, "share-event", entry["eventId"])
			require.Equal(t, "brian", entry["userId"])
		})
	}
}

func TestGroupedEmailFailureLogsContextAndChecksStoredEvents(t *testing.T) {
	for _, failure := range []string{"gateway", "authentication", "lookup", "SMTP"} {
		t.Run(failure, func(t *testing.T) {
			n, g, settings, ch := newDeliveryFixture()
			settings.interval = "daily"
			e := events.SpaceMembershipExpired{GranteeUserID: g.recipient.Id, SpaceID: &provider.StorageSpaceId{OpaqueId: "storage!space"}, SpaceName: "Project"}
			body, err := json.Marshal(e)
			require.NoError(t, err)
			n.registeredEvents = map[string]events.Unmarshaller{"events.SpaceMembershipExpired": events.SpaceMembershipExpired{}}
			n.userEventStore.historyClient = deliveryHistory{event: &ehmsg.Event{Type: "events.SpaceMembershipExpired", Event: body}}
			n.handleSpaceMembershipExpired(e, "queued-event")
			keys, err := n.userEventStore.listKeys("daily")
			require.NoError(t, err)
			require.Len(t, keys, 1)
			var output bytes.Buffer
			n.logger = log.Logger{Logger: zerolog.New(&output)}
			switch failure {
			case "gateway":
				n.gatewaySelector = deliverySelector{err: errors.New("no gateway")}
			case "authentication":
				g.authErr = errors.New("authentication unavailable")
			case "lookup":
				g.lookupErr = errors.New("directory offline")
			case "SMTP":
				ch.err = errors.New("SMTP offline")
			}
			logger := n.logger.With().Str("event", "SendEmailsEvent").Str("eventId", "digest-job").Logger()
			n.createGroupedMail(context.Background(), logger, keys[0])
			require.Empty(t, ch.messages)
			remaining, err := n.userEventStore.listKeys("daily")
			require.NoError(t, err)
			if failure == "gateway" || failure == "authentication" {
				require.Equal(t, keys, remaining, "gateway or authentication failure must leave events stored")
			} else {
				require.Empty(t, remaining, "events have already been removed when lookup or SMTP fails")
			}
			var entry map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &entry))
			require.Equal(t, "error", entry["level"])
			require.NotEmpty(t, entry["error"])
			require.Equal(t, "SendEmailsEvent", entry["event"])
			require.Equal(t, "digest-job", entry["eventId"])
			require.Equal(t, keys[0], entry["key"])
			if failure == "lookup" || failure == "SMTP" {
				require.Equal(t, "brian", entry["userId"])
			}
		})
	}
}
