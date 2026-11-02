package svc

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

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
