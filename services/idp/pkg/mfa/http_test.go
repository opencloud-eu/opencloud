package mfa

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

const testLogonPath = "/signin/v1/identifier/_/logon"

// fakeIdentifier behaves like the logon handler of the identifier: it asks the backend
// and answers with 200 or 204.
func fakeIdentifier(t *testing.T, svc *Service) http.Handler {
	wrapped := svc.WrapBackend(&fakeBackend{})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var logon struct {
			Params []string `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&logon), "the body must still be readable")
		ok, _, _, _, err := wrapped.Logon(r.Context(), "", logon.Params[0], logon.Params[1])
		switch {
		case err != nil:
			w.WriteHeader(http.StatusInternalServerError)
		case ok:
			_, _ = w.Write([]byte(`{"success":true}`))
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	})
}

func TestLogonMiddleware(t *testing.T) {
	svc, c, _ := newTestService(t)
	secret, _ := enroll(t, svc, c, alan)
	h := &Handler{svc: svc, logger: log.NopLogger()}
	server := h.LogonMiddleware(testLogonPath)(fakeIdentifier(t, svc))

	logon := func(params ...string) *http.Response {
		body, err := json.Marshal(map[string]any{"params": params, "state": "abc"})
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, testLogonPath, strings.NewReader(string(body))))
		return rec.Result()
	}

	t.Run("user without second factor", func(t *testing.T) {
		res := logon("mary", "secret", "1")
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Empty(t, res.Header.Get(ResultHeader))
	})

	t.Run("wrong password", func(t *testing.T) {
		res := logon("alan", "wrong", "1", codeFor(t, secret, c.now()))
		assert.Equal(t, http.StatusNoContent, res.StatusCode)
		assert.Empty(t, res.Header.Get(ResultHeader), "must not reveal that the user has a second factor")
	})

	t.Run("code required", func(t *testing.T) {
		res := logon("alan", "secret", "1")
		assert.Equal(t, http.StatusNoContent, res.StatusCode)
		assert.Equal(t, "required", res.Header.Get(ResultHeader))
	})

	t.Run("wrong code", func(t *testing.T) {
		res := logon("alan", "secret", "1", "000000")
		assert.Equal(t, http.StatusNoContent, res.StatusCode)
		assert.Equal(t, "invalid", res.Header.Get(ResultHeader))
	})

	t.Run("correct code", func(t *testing.T) {
		res := logon("alan", "secret", "1", codeFor(t, secret, c.now()))
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Empty(t, res.Header.Get(ResultHeader))
	})

	t.Run("locked", func(t *testing.T) {
		for i := 0; i < maxFailedAttempts; i++ {
			logon("alan", "secret", "1", "000000")
		}
		c.advance(totpPeriod * time.Second)
		res := logon("alan", "secret", "1", codeFor(t, secret, c.now()))
		assert.Equal(t, http.StatusNoContent, res.StatusCode)
		assert.Equal(t, "locked", res.Header.Get(ResultHeader))
	})

	t.Run("other requests are passed through untouched", func(t *testing.T) {
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			assert.Nil(t, loginStateFromContext(r.Context()))
		})
		h.LogonMiddleware(testLogonPath)(next).ServeHTTP(httptest.NewRecorder(),
			httptest.NewRequest(http.MethodPost, "/signin/v1/identifier/_/hello", strings.NewReader(`{}`)))
		assert.True(t, called)
	})

	t.Run("too large requests are rejected", func(t *testing.T) {
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, testLogonPath, strings.NewReader(strings.Repeat("x", maxBodySize+1))))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// api is a client for the routes of a Handler with alan logged in.
type api struct {
	t        *testing.T
	handler  http.Handler
	loggedIn *User
	headers  map[string]string
}

func newAPI(t *testing.T, svc *Service) *api {
	a := &api{t: t, loggedIn: &alan, headers: map[string]string{
		xsrfHeader: "1",
		"Origin":   "https://cloud.example.org",
	}}
	h := &Handler{svc: svc, logger: log.NopLogger(), currentUser: func(*http.Request) *User { return a.loggedIn }}
	a.handler = h.Routes()
	return a
}

func (a *api) do(method, path, body string, out any) int {
	a.t.Helper()
	req := httptest.NewRequest(method, "https://cloud.example.org"+path, strings.NewReader(body))
	for k, v := range a.headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)

	raw, err := io.ReadAll(rec.Body)
	require.NoError(a.t, err)
	if out != nil {
		require.NoError(a.t, json.Unmarshal(raw, out), string(raw))
	}
	assert.Equal(a.t, "no-store", rec.Header().Get("Cache-Control"))
	return rec.Code
}

func TestAPI(t *testing.T) {
	svc, c, _ := newTestService(t)
	svc.WrapBackend(&fakeBackend{})
	a := newAPI(t, svc)

	var status statusResponse
	require.Equal(t, http.StatusOK, a.do(http.MethodGet, "/status", "", &status))
	assert.False(t, status.Enabled)
	assert.Equal(t, "alan", status.Username)

	var errRes map[string]string

	t.Run("setup requires the password", func(t *testing.T) {
		for _, body := range []string{`{}`, `{"password":"wrong"}`} {
			assert.Equal(t, http.StatusForbidden, a.do(http.MethodPost, "/setup", body, &errRes))
			assert.Equal(t, "invalid_password", errRes["error"])
		}
		assert.Equal(t, http.StatusBadRequest, a.do(http.MethodPost, "/setup", `not json`, &errRes))
	})

	t.Run("enable requires a setup", func(t *testing.T) {
		assert.Equal(t, http.StatusConflict, a.do(http.MethodPost, "/enable", `{"code":"123456"}`, &errRes))
		assert.Equal(t, "no_pending_setup", errRes["error"])
	})

	var setup setupResponse
	require.Equal(t, http.StatusOK, a.do(http.MethodPost, "/setup", `{"password":"secret"}`, &setup))
	assert.Len(t, setup.Secret, 32)
	assert.True(t, strings.HasPrefix(setup.URI, "otpauth://totp/OpenCloud:alan?secret="+setup.Secret))

	t.Run("enable requires the right code", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, a.do(http.MethodPost, "/enable", `{"code":"000000"}`, &errRes))
		assert.Equal(t, "invalid_code", errRes["error"])
	})

	var codes recoveryCodesResponse
	require.Equal(t, http.StatusOK, a.do(http.MethodPost, "/enable", `{"code":"`+codeFor(t, setup.Secret, c.now())+`"}`, &codes))
	assert.Len(t, codes.RecoveryCodes, recoveryCodeCount)
	c.advance(totpPeriod * time.Second)

	require.Equal(t, http.StatusOK, a.do(http.MethodGet, "/status", "", &status))
	assert.True(t, status.Enabled)
	assert.Equal(t, recoveryCodeCount, status.RecoveryCodesLeft)
	require.NotNil(t, status.EnabledAt)

	t.Run("setup is rejected when enabled", func(t *testing.T) {
		assert.Equal(t, http.StatusConflict, a.do(http.MethodPost, "/setup", `{"password":"secret"}`, &errRes))
		assert.Equal(t, "already_enabled", errRes["error"])
	})

	t.Run("recovery codes can be regenerated with a code", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, a.do(http.MethodPost, "/recovery-codes", `{"code":"000000"}`, &errRes))

		var fresh recoveryCodesResponse
		require.Equal(t, http.StatusOK, a.do(http.MethodPost, "/recovery-codes", `{"code":"`+codeFor(t, setup.Secret, c.now())+`"}`, &fresh))
		assert.Len(t, fresh.RecoveryCodes, recoveryCodeCount)
		assert.NotEqual(t, codes.RecoveryCodes, fresh.RecoveryCodes)
		codes = fresh
		c.advance(totpPeriod * time.Second)
	})

	t.Run("disable requires a code", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, a.do(http.MethodPost, "/disable", `{}`, &errRes))
		assert.Equal(t, "invalid_code", errRes["error"])
		// the password does not help
		assert.Equal(t, http.StatusForbidden, a.do(http.MethodPost, "/disable", `{"password":"secret"}`, &errRes))

		require.Equal(t, http.StatusOK, a.do(http.MethodGet, "/status", "", &status))
		assert.True(t, status.Enabled)
	})

	t.Run("too many wrong codes lock", func(t *testing.T) {
		for i := 0; i < maxFailedAttempts; i++ {
			a.do(http.MethodPost, "/disable", `{"code":"000000"}`, nil)
		}
		assert.Equal(t, http.StatusTooManyRequests, a.do(http.MethodPost, "/disable", `{"code":"`+codes.RecoveryCodes[0]+`"}`, &errRes))
		assert.Equal(t, "locked", errRes["error"])
		c.advance(lockDuration + time.Second)
	})

	require.Equal(t, http.StatusNoContent, a.do(http.MethodPost, "/disable", `{"code":"`+codes.RecoveryCodes[0]+`"}`, nil))
	require.Equal(t, http.StatusOK, a.do(http.MethodGet, "/status", "", &status))
	assert.False(t, status.Enabled)

	t.Run("disable when not enabled", func(t *testing.T) {
		assert.Equal(t, http.StatusConflict, a.do(http.MethodPost, "/disable", `{"code":"123456"}`, &errRes))
		assert.Equal(t, "not_enabled", errRes["error"])
	})
}

func TestAPIRequiresLogin(t *testing.T) {
	svc, _, _ := newTestService(t)
	svc.WrapBackend(&fakeBackend{})
	a := newAPI(t, svc)
	a.loggedIn = nil

	for path, method := range map[string]string{
		"/status":         http.MethodGet,
		"/setup":          http.MethodPost,
		"/enable":         http.MethodPost,
		"/disable":        http.MethodPost,
		"/recovery-codes": http.MethodPost,
	} {
		assert.Equal(t, http.StatusUnauthorized, a.do(method, path, `{"password":"secret","code":"123456"}`, nil), path)
	}
}

func TestAPIRejectsCrossSiteRequests(t *testing.T) {
	svc, _, _ := newTestService(t)
	svc.WrapBackend(&fakeBackend{})

	for name, headers := range map[string]map[string]string{
		"no xsrf header":       {"Origin": "https://cloud.example.org"},
		"wrong xsrf header":    {xsrfHeader: "0", "Origin": "https://cloud.example.org"},
		"no origin or referer": {xsrfHeader: "1"},
		"foreign origin":       {xsrfHeader: "1", "Origin": "https://evil.example.com"},
		"foreign referer":      {xsrfHeader: "1", "Referer": "https://evil.example.com/page"},
		"origin wins":          {xsrfHeader: "1", "Origin": "https://evil.example.com", "Referer": "https://cloud.example.org/"},
		"lookalike host":       {xsrfHeader: "1", "Origin": "https://cloud.example.org.evil.example.com"},
	} {
		t.Run(name, func(t *testing.T) {
			a := newAPI(t, svc)
			a.headers = headers
			assert.Equal(t, http.StatusBadRequest, a.do(http.MethodGet, "/status", "", nil))
			assert.Equal(t, http.StatusBadRequest, a.do(http.MethodPost, "/setup", `{"password":"secret"}`, nil))
		})
	}

	t.Run("a matching referer is sufficient", func(t *testing.T) {
		a := newAPI(t, svc)
		a.headers = map[string]string{xsrfHeader: "1", "Referer": "https://cloud.example.org/signin/v1/mfa"}
		assert.Equal(t, http.StatusOK, a.do(http.MethodGet, "/status", "", nil))
	})
}
