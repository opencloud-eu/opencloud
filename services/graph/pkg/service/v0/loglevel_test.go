package svc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5/middleware"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

// logEntry logs one event with fn into a buffer and returns the decoded entry.
func logEntry(t *testing.T, fn func(logger *log.Logger) *zerolog.Event) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	logger := &log.Logger{Logger: zerolog.New(&buf)}
	fn(logger).Msg("test")
	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	return entry
}

func TestErrorEvent(t *testing.T) {
	t.Run("unexpected failure: error level with the error text", func(t *testing.T) {
		err := errorcode.New(errorcode.GeneralException, "share manager unavailable")
		entry := logEntry(t, func(l *log.Logger) *zerolog.Event { return errorEvent(l, err) })
		assert.Equal(t, "error", entry["level"])
		assert.Equal(t, "generalException", entry["errorCode"])
		assert.Equal(t, err.Error(), entry["error"])
	})
	t.Run("expected client error: debug level with the error text", func(t *testing.T) {
		err := errorcode.New(errorcode.ItemNotFound, "share not found")
		entry := logEntry(t, func(l *log.Logger) *zerolog.Event { return errorEvent(l, err) })
		assert.Equal(t, "debug", entry["level"])
		assert.Equal(t, "itemNotFound", entry["errorCode"])
		assert.Equal(t, err.Error(), entry["error"])
	})
}

func TestFallbackErrorEvent(t *testing.T) {
	notFound := errorcode.New(errorcode.ItemNotFound, "not found")
	failure := errorcode.New(errorcode.GeneralException, "share manager unavailable")

	t.Run("the other attempt failed unexpectedly: error level, both texts", func(t *testing.T) {
		entry := logEntry(t, func(l *log.Logger) *zerolog.Event { return fallbackErrorEvent(l, notFound, failure, "lookupError") })
		assert.Equal(t, "error", entry["level"])
		assert.Equal(t, "itemNotFound", entry["errorCode"])
		assert.Equal(t, notFound.Error(), entry["error"])
		assert.Equal(t, "generalException", entry["lookupErrorCode"])
		assert.Equal(t, failure.Error(), entry["lookupError"])
	})
	t.Run("the returned error is unexpected: error level, both texts", func(t *testing.T) {
		entry := logEntry(t, func(l *log.Logger) *zerolog.Event { return fallbackErrorEvent(l, failure, notFound, "lookupError") })
		assert.Equal(t, "error", entry["level"])
		assert.Equal(t, failure.Error(), entry["error"])
		assert.Equal(t, "itemNotFound", entry["lookupErrorCode"])
		assert.Equal(t, notFound.Error(), entry["lookupError"])
	})
	t.Run("both attempts failed expectedly: debug level, both texts", func(t *testing.T) {
		entry := logEntry(t, func(l *log.Logger) *zerolog.Event { return fallbackErrorEvent(l, notFound, notFound, "lookupError") })
		assert.Equal(t, "debug", entry["level"])
		assert.Equal(t, notFound.Error(), entry["error"])
		assert.Equal(t, notFound.Error(), entry["lookupError"])
	})
	t.Run("without another attempt: like errorEvent", func(t *testing.T) {
		entry := logEntry(t, func(l *log.Logger) *zerolog.Event { return fallbackErrorEvent(l, failure, nil, "lookupError") })
		assert.Equal(t, "error", entry["level"])
		assert.Equal(t, failure.Error(), entry["error"])
		assert.NotContains(t, entry, "lookupErrorCode")
	})
}

// failingGatewaySelector is a gateway selector that never finds a gateway.
type failingGatewaySelector struct{ err error }

func (s failingGatewaySelector) Next(...pool.Option) (gateway.GatewayAPIClient, error) {
	return nil, s.err
}

// TestHelperGatewaySelectionFailureIsLoggedAsError checks that every helper that selects a gateway on its
// own logs a failed selection at error level with the selector's error, whichever operation calls it.
// The entry carries the request id, so it can be matched with the entry of the calling operation,
// and the permission id when the helper works on a single permission.
func TestHelperGatewaySelectionFailureIsLoggedAsError(t *testing.T) {
	const listMsg = "could not select next gateway client"
	const lookupMsg = "selecting gatewaySelector failed"
	const requestID = "request-42"
	const permissionID = "permissionid"
	ctx := context.WithValue(context.Background(), middleware.RequestIDKey, requestID)
	itemID := &storageprovider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "3"}

	tests := []struct {
		name         string
		message      string
		permissionID string
		call         func(s DriveItemPermissionsService) error
	}{
		{"listSharesWithSpaceRootFilter", listMsg, "", func(s DriveItemPermissionsService) error {
			_, err := s.listSharesWithSpaceRootFilter(ctx, false, nil, driveItemsByResourceID{})
			return err
		}},
		{"listOCMShares", listMsg, "", func(s DriveItemPermissionsService) error {
			_, err := s.listOCMShares(ctx, nil, driveItemsByResourceID{})
			return err
		}},
		{"listPublicShares", listMsg, "", func(s DriveItemPermissionsService) error {
			_, err := s.listPublicShares(ctx, nil, driveItemsByResourceID{})
			return err
		}},
		{"updatePublicLinkPermission", listMsg, permissionID, func(s DriveItemPermissionsService) error {
			_, err := s.updatePublicLinkPermission(ctx, permissionID, itemID, &libregraph.Permission{})
			return err
		}},
		{"getCS3PublicShareByID", lookupMsg, permissionID, func(s DriveItemPermissionsService) error {
			_, err := s.getCS3PublicShareByID(ctx, permissionID)
			return err
		}},
		{"getCS3UserShareByID", lookupMsg, permissionID, func(s DriveItemPermissionsService) error {
			_, err := s.getCS3UserShareByID(ctx, permissionID)
			return err
		}},
		{"getCS3OCMShareByID", lookupMsg, permissionID, func(s DriveItemPermissionsService) error {
			_, err := s.getCS3OCMShareByID(ctx, permissionID)
			return err
		}},
		{"getPermissionByID", lookupMsg, permissionID, func(s DriveItemPermissionsService) error {
			_, _, err := s.getPermissionByID(ctx, permissionID, itemID)
			return err
		}},
		{"getOCMPermissionByID", lookupMsg, permissionID, func(s DriveItemPermissionsService) error {
			_, _, err := s.getOCMPermissionByID(ctx, permissionID, itemID)
			return err
		}},
		{"removeUserShare", lookupMsg, permissionID, func(s DriveItemPermissionsService) error {
			return s.removeUserShare(ctx, permissionID)
		}},
		{"removePublicShare", lookupMsg, permissionID, func(s DriveItemPermissionsService) error {
			return s.removePublicShare(ctx, permissionID)
		}},
		{"removeOCMPermission", lookupMsg, permissionID, func(s DriveItemPermissionsService) error {
			return s.removeOCMPermission(ctx, permissionID)
		}},
		{"updateUserShare", lookupMsg, permissionID, func(s DriveItemPermissionsService) error {
			_, err := s.updateUserShare(ctx, permissionID, itemID, &libregraph.Permission{})
			return err
		}},
		{"updateOCMPermission", lookupMsg, permissionID, func(s DriveItemPermissionsService) error {
			_, err := s.updateOCMPermission(ctx, permissionID, itemID, &libregraph.Permission{})
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			s := DriveItemPermissionsService{BaseGraphService: BaseGraphService{
				logger:          &log.Logger{Logger: zerolog.New(&buf)},
				gatewaySelector: failingGatewaySelector{err: errors.New("no gateway available")},
			}}

			require.Error(t, tc.call(s))

			var entry map[string]any
			require.NoError(t, json.Unmarshal(buf.Bytes(), &entry), "exactly one log entry")
			assert.Equal(t, "error", entry["level"])
			assert.Equal(t, tc.message, entry["message"])
			assert.Equal(t, "no gateway available", entry["error"])
			assert.Equal(t, requestID, entry[log.RequestIDString])
			if tc.permissionID != "" {
				assert.Equal(t, tc.permissionID, entry[_logKeyPermissionID])
			} else {
				assert.NotContains(t, entry, _logKeyPermissionID)
			}
		})
	}
}

func TestLogLevelForError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want zerolog.Level
	}{
		// expected client errors, rendered as 4xx by errorcode.Error.Render
		{"item not found", errorcode.New(errorcode.ItemNotFound, "x"), zerolog.DebugLevel},
		{"access denied", errorcode.New(errorcode.AccessDenied, "x"), zerolog.DebugLevel},
		{"invalid request", errorcode.New(errorcode.InvalidRequest, "x"), zerolog.DebugLevel},
		{"name already exists", errorcode.New(errorcode.NameAlreadyExists, "x"), zerolog.DebugLevel},
		{"invalid range", errorcode.New(errorcode.InvalidRange, "x"), zerolog.DebugLevel},
		{"item is locked", errorcode.New(errorcode.ItemIsLocked, "x"), zerolog.DebugLevel},
		{"precondition failed", errorcode.New(errorcode.PreconditionFailed, "x"), zerolog.DebugLevel},
		{"not allowed", errorcode.New(errorcode.NotAllowed, "x"), zerolog.DebugLevel},
		{"wrapped item not found", fmt.Errorf("lookup: %w", errorcode.New(errorcode.ItemNotFound, "x")), zerolog.DebugLevel},
		{"wrapped access denied", fmt.Errorf("lookup: %w", errorcode.New(errorcode.AccessDenied, "x")), zerolog.DebugLevel},

		// unexpected server-side failures
		{"general exception", errorcode.New(errorcode.GeneralException, "x"), zerolog.ErrorLevel},
		{"service not available", errorcode.New(errorcode.ServiceNotAvailable, "x"), zerolog.ErrorLevel},
		{"unauthenticated", errorcode.New(errorcode.Unauthenticated, "x"), zerolog.ErrorLevel},
		{"quota limit reached", errorcode.New(errorcode.QuotaLimitReached, "x"), zerolog.ErrorLevel},
		{"not supported", errorcode.New(errorcode.NotSupported, "x"), zerolog.ErrorLevel},
		{"wrapped general exception", fmt.Errorf("stat: %w", errorcode.New(errorcode.GeneralException, "x")), zerolog.ErrorLevel},
		{"plain error", errors.New("connection refused"), zerolog.ErrorLevel},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, logLevelForError(tc.err))
		})
	}
}

func TestErrorCodeForError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"graph error", errorcode.New(errorcode.ItemNotFound, "private-file.txt"), "itemNotFound"},
		{"wrapped graph error", fmt.Errorf("stat: %w", errorcode.New(errorcode.ServiceNotAvailable, "x")), "serviceNotAvailable"},
		// errorcode.RenderError renders untyped errors as generalException
		{"plain error", errors.New("connection refused"), "generalException"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, errorCodeForError(tc.err))
		})
	}
}

// TestLogLevelForErrorMatchesRenderedStatus keeps logLevelForError in sync with errorcode.Error.Render:
// every code rendered as a 4xx is an expected client error, everything else is unexpected.
func TestLogLevelForErrorMatchesRenderedStatus(t *testing.T) {
	// ItemIsLocked is the last declared error code
	for code := errorcode.AccessDenied; code <= errorcode.ItemIsLocked; code++ {
		t.Run(code.String(), func(t *testing.T) {
			rr := httptest.NewRecorder()
			errorcode.New(code, "x").Render(rr, httptest.NewRequest(http.MethodGet, "/", nil))

			want := zerolog.ErrorLevel
			if rr.Code >= http.StatusBadRequest && rr.Code < http.StatusInternalServerError {
				want = zerolog.DebugLevel
			}
			assert.Equal(t, want, logLevelForError(errorcode.New(code, "x")), "rendered status %d", rr.Code)
		})
	}
}
