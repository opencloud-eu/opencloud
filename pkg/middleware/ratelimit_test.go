package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitPerAuthenticatedUser(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	// do sends a request as the given user, or without one if user is empty
	do := func(h http.Handler, user, path string) *httptest.ResponseRecorder {
		ctx := context.Background()
		if user != "" {
			ctx = revactx.ContextSetUser(ctx, &userv1beta1.User{Id: &userv1beta1.UserId{OpaqueId: user}})
		}
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, path, nil).WithContext(ctx))
		return rr
	}

	t.Run("limits each user on their own", func(t *testing.T) {
		h := RateLimitPerAuthenticatedUser(2, time.Minute, nil)(ok)
		assert.Equal(t, http.StatusOK, do(h, "alice", "/a").Code)
		assert.Equal(t, http.StatusOK, do(h, "alice", "/a").Code)
		rr := do(h, "alice", "/a")
		assert.Equal(t, http.StatusTooManyRequests, rr.Code)
		assert.NotEmpty(t, rr.Header().Get("Retry-After"))
		assert.Equal(t, http.StatusOK, do(h, "bob", "/a").Code)
	})

	t.Run("counts by user, not by URL", func(t *testing.T) {
		h := RateLimitPerAuthenticatedUser(2, time.Minute, nil)(ok)
		assert.Equal(t, http.StatusOK, do(h, "alice", "/a").Code)
		assert.Equal(t, http.StatusOK, do(h, "alice", "/a/").Code)
		assert.Equal(t, http.StatusTooManyRequests, do(h, "alice", "/b?x=1").Code)
	})

	t.Run("rejects requests without an authenticated user", func(t *testing.T) {
		h := RateLimitPerAuthenticatedUser(2, time.Minute, nil)(ok)
		assert.Equal(t, http.StatusUnauthorized, do(h, "", "/a").Code)
	})

	t.Run("a limit of 0 disables it", func(t *testing.T) {
		h := RateLimitPerAuthenticatedUser(0, time.Minute, nil)(ok)
		for range 5 {
			assert.Equal(t, http.StatusOK, do(h, "alice", "/a").Code)
		}
	})

	t.Run("answers beyond the limit with the given handler", func(t *testing.T) {
		limitReached := func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"slow down"}`))
		}
		h := RateLimitPerAuthenticatedUser(1, time.Minute, limitReached)(ok)
		assert.Equal(t, http.StatusOK, do(h, "alice", "/a").Code)
		rr := do(h, "alice", "/a")
		assert.Equal(t, http.StatusTooManyRequests, rr.Code)
		assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"error":"slow down"}`, rr.Body.String())
		// set before the handler is called
		assert.NotEmpty(t, rr.Header().Get("Retry-After"))
	})
}
