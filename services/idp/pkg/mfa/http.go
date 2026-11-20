package mfa

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/libregraph/lico/identifier"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

const (
	// ResultHeader is the response header of the logon endpoint that tells the login page
	// why a logon with a correct password was rejected. Its value is a LoginResult.
	ResultHeader = "OpenCloud-MFA"

	// xsrfHeader has to be sent by the login page, like for the other identifier endpoints.
	xsrfHeader = "Kopano-Konnect-XSRF"

	// logonCodeParam is the index of the one-time code in the params of a logon request:
	// [username, password, mode, code]
	logonCodeParam = 3

	maxBodySize = 1 << 16
)

// Handler provides the HTTP API of the two-factor authentication.
type Handler struct {
	svc *Service
	// currentUser returns the logged in user of a request or nil.
	currentUser func(r *http.Request) *User
	logger      log.Logger
}

// NewHandler creates a new Handler. The identifier is used to find out who is logged in.
func NewHandler(svc *Service, identifier *identifier.Identifier, logger log.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger,
		currentUser: func(r *http.Request) *User {
			identified, err := identifier.GetUserFromLogonCookie(r.Context(), r, 0, true)
			if err != nil {
				logger.Debug().Err(err).Msg("could not read logon cookie")
			}
			if identified == nil || identified.Subject() == "" {
				return nil
			}
			return &User{ID: identified.Subject(), Username: identified.Username()}
		},
	}
}

// LogonMiddleware passes the one-time code of logon requests on logonPath to the wrapped
// identifier backend (see Service.WrapBackend) and reports the result of the second
// factor check in the ResultHeader of the response.
func (h *Handler) LogonMiddleware(logonPath string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != logonPath {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodySize))
			if err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			state := &loginState{}
			var logon struct {
				Params []string `json:"params"`
			}
			// invalid requests are rejected by the identifier
			if err := json.Unmarshal(body, &logon); err == nil && len(logon.Params) > logonCodeParam {
				state.code = logon.Params[logonCodeParam]
			}

			next.ServeHTTP(
				&logonResponseWriter{ResponseWriter: w, state: state},
				r.WithContext(withLoginState(r.Context(), state)),
			)
		})
	}
}

// logonResponseWriter adds the ResultHeader to the response of a rejected logon.
type logonResponseWriter struct {
	http.ResponseWriter
	state       *loginState
	wroteHeader bool
}

func (w *logonResponseWriter) WriteHeader(statusCode int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		if !w.state.result.Passed() {
			w.Header().Set(ResultHeader, string(w.state.result))
		}
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *logonResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// Unwrap supports http.ResponseController.
func (w *logonResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// Routes returns the API users manage their second factor with. All endpoints require a
// logged in user. It has to be mounted below the path of the logon cookie.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(h.secure)
	r.Get("/status", h.authenticated(h.status))
	r.Post("/setup", h.authenticated(h.setup))
	r.Post("/enable", h.authenticated(h.enable))
	r.Post("/disable", h.authenticated(h.disable))
	r.Post("/recovery-codes", h.authenticated(h.recoveryCodes))
	return r
}

// secure protects against cross site request forgery the same way the identifier does
// for its endpoints: a custom header and a matching Origin or Referer are required.
func (h *Handler) secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		requiredHost := r.Host
		if host, port, err := net.SplitHostPort(requiredHost); err == nil && port == "443" {
			requiredHost = host
		}

		source := r.Header.Get("Origin")
		if source == "" {
			source = r.Header.Get("Referer")
		}
		sourceURL, err := url.Parse(source)
		if r.Header.Get(xsrfHeader) != "1" || source == "" || err != nil || sourceURL.Host != requiredHost {
			h.logger.Warn().Str("host", requiredHost).Str("origin", r.Header.Get("Origin")).Str("referer", r.Referer()).
				Msg("rejecting two-factor authentication request")
			writeError(w, http.StatusBadRequest, "bad_request")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type userHandler func(w http.ResponseWriter, r *http.Request, user User)

// authenticated resolves the logged in user from the logon cookie of the identifier.
func (h *Handler) authenticated(next userHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := h.currentUser(r)
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r, *user)
	}
}

type statusResponse struct {
	Enabled           bool       `json:"enabled"`
	EnabledAt         *time.Time `json:"enabledAt,omitempty"`
	RecoveryCodesLeft int        `json:"recoveryCodesLeft"`
	Username          string     `json:"username"`
}

type secretRequest struct {
	Password string `json:"password"`
	Code     string `json:"code"`
}

type setupResponse struct {
	Secret string `json:"secret"`
	URI    string `json:"uri"`
}

type recoveryCodesResponse struct {
	RecoveryCodes []string `json:"recoveryCodes"`
}

func (h *Handler) status(w http.ResponseWriter, _ *http.Request, user User) {
	status, err := h.svc.Status(user)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	res := statusResponse{
		Enabled:           status.Enabled,
		RecoveryCodesLeft: status.RecoveryCodesLeft,
		Username:          user.Username,
	}
	if status.Enabled {
		res.EnabledAt = &status.EnabledAt
	}
	writeJSON(w, http.StatusOK, res)
}

// setup starts the setup. The password is required again, so a session somebody left
// open cannot be used to lock the user out with a foreign authenticator app.
func (h *Handler) setup(w http.ResponseWriter, r *http.Request, user User) {
	req, ok := readRequest(w, r)
	if !ok {
		return
	}

	valid, err := h.svc.verifyPassword(r.Context(), user, req.Password)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if !valid {
		writeError(w, http.StatusForbidden, "invalid_password")
		return
	}

	secret, uri, err := h.svc.Setup(user)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, setupResponse{Secret: secret, URI: uri})
}

func (h *Handler) enable(w http.ResponseWriter, r *http.Request, user User) {
	req, ok := readRequest(w, r)
	if !ok {
		return
	}

	codes, err := h.svc.Enable(user, req.Code)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, recoveryCodesResponse{RecoveryCodes: codes})
}

func (h *Handler) disable(w http.ResponseWriter, r *http.Request, user User) {
	req, ok := readRequest(w, r)
	if !ok {
		return
	}

	if err := h.svc.Disable(user, req.Code); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) recoveryCodes(w http.ResponseWriter, r *http.Request, user User) {
	req, ok := readRequest(w, r)
	if !ok {
		return
	}

	codes, err := h.svc.RegenerateRecoveryCodes(user, req.Code)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, recoveryCodesResponse{RecoveryCodes: codes})
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidCode):
		writeError(w, http.StatusForbidden, "invalid_code")
	case errors.Is(err, ErrLocked):
		writeError(w, http.StatusTooManyRequests, "locked")
	case errors.Is(err, ErrAlreadyEnabled):
		writeError(w, http.StatusConflict, "already_enabled")
	case errors.Is(err, ErrNotEnabled):
		writeError(w, http.StatusConflict, "not_enabled")
	case errors.Is(err, ErrNoPendingSetup):
		writeError(w, http.StatusConflict, "no_pending_setup")
	default:
		h.logger.Error().Err(err).Msg("two-factor authentication request failed")
		writeError(w, http.StatusInternalServerError, "internal_error")
	}
}

func readRequest(w http.ResponseWriter, r *http.Request) (secretRequest, bool) {
	var req secretRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodySize)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request")
		return req, false
	}
	return req, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}
