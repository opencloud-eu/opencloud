package service_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	revaContext "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/sse/pkg/config"
	"github.com/opencloud-eu/opencloud/services/sse/pkg/service"
)

func TestNewSSEHandler(t *testing.T) {
	eventChan := make(chan events.Event)
	defer close(eventChan)

	t.Run("initialization", func(t *testing.T) {
		_, err := service.NewSSEHandler(context.Background(), &config.Config{}, log.NopLogger(), eventChan)
		assert.NoError(t, err)
	})
}

func TestSSEHandler_ServeHTTP(t *testing.T) {
	eventChan := make(chan events.Event)
	defer close(eventChan)

	handler, _ := service.NewSSEHandler(context.Background(), &config.Config{}, log.NopLogger(), eventChan)

	t.Run("fails without user topic", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("handles sse events", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := revaContext.ContextSetUser(r.Context(), &userv1beta1.User{
				Id: &userv1beta1.UserId{
					OpaqueId: "user_1",
				},
			})

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

			handler.ServeHTTP(w, r.WithContext(ctx))
		}))
		defer ts.Close()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() {
			_ = resp.Body.Close()
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		reader := bufio.NewReader(resp.Body)

		eventChan <- events.Event{
			Event: events.SendSSE{
				UserIDs: []string{"user_1"},
				Type:    "whatever",
				Message: []byte("u1_m1"),
			},
		}

		eventChan <- events.Event{
			Event: events.SendSSE{
				UserIDs: []string{"user_1"},
				Type:    "whatever",
				Message: []byte("u1_m2"),
			},
		}

		eventChan <- events.Event{
			Event: events.SendSSE{
				UserIDs: []string{"user_2"},
				Type:    "whatever",
				Message: []byte("u2_m1"),
			},
		}

		eventChan <- events.Event{
			Event: events.SendSSE{
				UserIDs: []string{"all"},
				Type:    "whatever",
				Message: []byte("all_m1"),
			},
		}

		var messages []string
		for len(messages) < 3 {
			line, err := reader.ReadString('\n')
			require.NoError(t, err)

			if line == "\n" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			messages = append(messages, strings.TrimSpace(strings.TrimPrefix(line, "data: ")))
		}

		assert.Equal(t, []string{"u1_m1", "u1_m2", "all_m1"}, messages)
	})
}
