package groupware

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"

	"github.com/jellydator/ttlcache/v3"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/config"
)

const (
	logUsername              = "username" // this should match jmap.logUsername to avoid having the field twice in the logs under different keys
	logErrorId               = "error-id"
	logErrorCode             = "code"
	logErrorStatus           = "status"
	logErrorSourceHeader     = "source-header"
	logErrorSourceParameter  = "source-parameter"
	logErrorSourcePointer    = "source-pointer"
	logInvalidQueryParameter = "error-query-param"
	logInvalidPathParameter  = "error-path-param"
	logFolderId              = "folder-id"
	logQuery                 = "query"
)

type Groupware struct {
	mux               *chi.Mux
	logger            *log.Logger
	defaultEmailLimit int
	maxBodyValueBytes int
	sessionCache      *ttlcache.Cache[string, cachedSession]
	jmap              *jmap.Client
	usernameProvider  UsernameProvider
}

type GroupwareInitializationError struct {
	Message string
	Err     error
}

func (e GroupwareInitializationError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("failed to create Groupware: %s: %v", e.Message, e.Err.Error())
	} else {
		return fmt.Sprintf("failed to create Groupware: %v", e.Err.Error())
	}
}
func (e GroupwareInitializationError) Unwrap() error {
	return e.Err
}

type GroupwareSessionEventListener struct {
	logger       *log.Logger
	sessionCache *ttlcache.Cache[string, cachedSession]
}

func (l GroupwareSessionEventListener) OnSessionOutdated(session *jmap.Session, newSessionState string) {
	// it's enough to remove the session from the cache, as it will be fetched on-demand
	// the next time an operation is performed on behalf of the user
	l.sessionCache.Delete(session.Username)

	l.logger.Trace().Msgf("removed outdated session for user '%v': state %s -> %s", session.Username, session.State, newSessionState)
}

var _ jmap.SessionEventListener = GroupwareSessionEventListener{}

func NewGroupware(config *config.Config, logger *log.Logger, mux *chi.Mux) (*Groupware, error) {
	baseUrl, err := url.Parse(config.Mail.BaseUrl)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to parse configured Mail.Baseurl '%v'", config.Mail.BaseUrl)
		return nil, GroupwareInitializationError{Message: fmt.Sprintf("failed to parse configured Mail.BaseUrl '%s'", config.Mail.BaseUrl), Err: err}
	}

	masterUsername := config.Mail.Master.Username
	if masterUsername == "" {
		logger.Error().Msg("failed to parse empty Mail.Master.Username")
		return nil, GroupwareInitializationError{Message: "Mail.Master.Username is empty"}
	}
	masterPassword := config.Mail.Master.Password
	if masterPassword == "" {
		logger.Error().Msg("failed to parse empty Mail.Master.Password")
		return nil, GroupwareInitializationError{Message: "Mail.Master.Password is empty"}
	}

	defaultEmailLimit := max(config.Mail.DefaultEmailLimit, 0)
	maxBodyValueBytes := max(config.Mail.MaxBodyValueBytes, 0)
	responseHeaderTimeout := max(config.Mail.ResponseHeaderTimeout, 0)
	sessionCacheMaxCapacity := uint64(max(config.Mail.SessionCache.MaxCapacity, 0))
	sessionCacheTtl := max(config.Mail.SessionCache.Ttl, 0)
	sessionFailureCacheTtl := max(config.Mail.SessionCache.FailureTtl, 0)

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.ResponseHeaderTimeout = responseHeaderTimeout
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	tr.TLSClientConfig = tlsConfig
	c := *http.DefaultClient
	c.Transport = tr

	usernameProvider := NewRevaContextUsernameProvider()

	api := jmap.NewHttpJmapApiClient(
		*baseUrl,
		&c,
		masterUsername,
		masterPassword,
	)

	jmapClient := jmap.NewClient(api, api, api)

	var sessionCache *ttlcache.Cache[string, cachedSession]
	{
		sessionLoader := &sessionCacheLoader{
			logger:     logger,
			jmapClient: &jmapClient,
			errorTtl:   sessionFailureCacheTtl,
		}

		sessionCache = ttlcache.New(
			ttlcache.WithCapacity[string, cachedSession](sessionCacheMaxCapacity),
			ttlcache.WithTTL[string, cachedSession](sessionCacheTtl),
			ttlcache.WithDisableTouchOnHit[string, cachedSession](),
			ttlcache.WithLoader(sessionLoader),
		)
		go sessionCache.Start()
	}

	if logger.Trace().Enabled() {
		sessionCache.OnEviction(func(c context.Context, r ttlcache.EvictionReason, item *ttlcache.Item[string, cachedSession]) {
			reason := ""
			switch r {
			case ttlcache.EvictionReasonDeleted:
				reason = "deleted"
			case ttlcache.EvictionReasonCapacityReached:
				reason = "capacity reached"
			case ttlcache.EvictionReasonExpired:
				reason = fmt.Sprintf("expired after %vms", item.TTL().Milliseconds())
			case ttlcache.EvictionReasonMaxCostExceeded:
				reason = "max cost exceeded"
			}
			if reason == "" {
				reason = fmt.Sprintf("unknown (%v)", r)
			}

			logger.Trace().Msgf("session cache eviction of user '%v': %v", item.Key(), reason)
		})
	}

	sessionEventListener := GroupwareSessionEventListener{sessionCache: sessionCache, logger: logger}
	jmapClient.AddSessionEventListener(&sessionEventListener)

	return &Groupware{
		mux:               mux,
		logger:            logger,
		sessionCache:      sessionCache,
		usernameProvider:  usernameProvider,
		jmap:              &jmapClient,
		defaultEmailLimit: defaultEmailLimit,
		maxBodyValueBytes: maxBodyValueBytes,
	}, nil
}

func (g Groupware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

// Provide a JMAP Session for the
func (g Groupware) session(username string, _ *http.Request, _ context.Context, _ *log.Logger) (jmap.Session, bool, error) {
	item := g.sessionCache.Get(username)
	if item != nil {
		value := item.Value()
		if value != nil {
			if value.Success() {
				return value.Get(), true, nil
			} else {
				return jmap.Session{}, false, value.Error()
			}
		}
	}
	return jmap.Session{}, false, nil
}

// using a wrapper class for requests, to group multiple parameters, really to avoid crowding the
// API of handlers but also to make it easier to expand it in the future without having to modify
// the parameter list of every single handler function
type Request struct {
	r       *http.Request
	ctx     context.Context
	logger  *log.Logger
	session *jmap.Session
}

type Response struct {
	body         any
	err          *Error
	etag         string
	sessionState string
}

func errorResponse(err *Error) Response {
	return Response{
		body:         nil,
		err:          err,
		etag:         "",
		sessionState: "",
	}
}

func response(body any, sessionStatus string) Response {
	return Response{
		body:         body,
		err:          nil,
		etag:         sessionStatus,
		sessionState: sessionStatus,
	}
}

func etagResponse(body any, sessionState string, etag string) Response {
	return Response{
		body:         body,
		err:          nil,
		etag:         etag,
		sessionState: sessionState,
	}
}

func etagOnlyResponse(body any, etag string) Response {
	return Response{
		body:         body,
		err:          nil,
		etag:         etag,
		sessionState: "",
	}
}

func noContentResponse(sessionStatus string) Response {
	return Response{
		body:         "",
		err:          nil,
		etag:         sessionStatus,
		sessionState: sessionStatus,
	}
}

func notFoundResponse(sessionStatus string) Response {
	return Response{
		body:         nil,
		err:          nil,
		etag:         sessionStatus,
		sessionState: sessionStatus,
	}
}

func (r Request) GetAccountId() string {
	accountId := chi.URLParam(r.r, UriParamAccount)
	return r.session.MailAccountId(accountId)
}

func (r Request) GetAccount() (jmap.SessionAccount, *Error) {
	accountId := r.GetAccountId()

	account, ok := r.session.Accounts[accountId]
	if !ok {
		errorId := r.errorId()
		r.logger.Debug().Msgf("failed to find account '%v'", accountId)
		return jmap.SessionAccount{}, apiError(errorId, ErrorNonExistingAccount,
			withDetail(fmt.Sprintf("The account '%v' does not exist", log.SafeString(accountId))),
			withSource(&ErrorSource{Parameter: UriParamAccount}),
		)
	}
	return account, nil
}

func (r Request) parseNumericParam(param string, defaultValue int) (int, bool, *Error) {
	q := r.r.URL.Query()
	if !q.Has(param) {
		return defaultValue, false, nil
	}

	str := q.Get(param)
	if str == "" {
		return defaultValue, false, nil
	}

	value, err := strconv.ParseInt(str, 10, 0)
	if err != nil {
		errorId := r.errorId()
		msg := fmt.Sprintf("Invalid value for query parameter '%v': '%s': %s", param, log.SafeString(str), err.Error())
		return defaultValue, true, apiError(errorId, ErrorInvalidRequestParameter,
			withDetail(msg),
			withSource(&ErrorSource{Parameter: param}),
		)
	}
	return int(value), true, nil
}

func (r Request) parseDateParam(param string) (time.Time, bool, *Error) {
	q := r.r.URL.Query()
	if !q.Has(param) {
		return time.Time{}, false, nil
	}

	str := q.Get(param)
	if str == "" {
		return time.Time{}, false, nil
	}

	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		errorId := r.errorId()
		msg := fmt.Sprintf("Invalid RFC3339 value for query parameter '%v': '%s': %s", param, log.SafeString(str), err.Error())
		return time.Time{}, true, apiError(errorId, ErrorInvalidRequestParameter,
			withDetail(msg),
			withSource(&ErrorSource{Parameter: param}),
		)
	}
	return t, true, nil
}

func (r Request) parseBoolParam(param string, defaultValue bool) (bool, bool, *Error) {
	q := r.r.URL.Query()
	if !q.Has(param) {
		return defaultValue, false, nil
	}

	str := q.Get(param)
	if str == "" {
		return defaultValue, false, nil
	}

	b, err := strconv.ParseBool(str)
	if err != nil {
		errorId := r.errorId()
		msg := fmt.Sprintf("Invalid boolean value for query parameter '%v': '%s': %s", param, log.SafeString(str), err.Error())
		return defaultValue, true, apiError(errorId, ErrorInvalidRequestParameter,
			withDetail(msg),
			withSource(&ErrorSource{Parameter: param}),
		)
	}
	return b, true, nil
}

func (r Request) body(target any) *Error {
	body := r.r.Body
	defer func(b io.ReadCloser) {
		err := b.Close()
		if err != nil {
			r.logger.Error().Err(err).Msg("failed to close request body")
		}
	}(body)

	err := json.NewDecoder(body).Decode(target)
	if err != nil {
		// TODO(pbleser-oc) error handling when failing to decode body
	}
	return nil
}

func (g Groupware) log(error *Error) {
	var level *zerolog.Event
	if error.NumStatus < 300 {
		// shouldn't land here, but just in case: 1xx and 2xx are "OK" and should be logged as debug
		level = g.logger.Debug()
	} else if error.NumStatus == http.StatusUnauthorized || error.NumStatus == http.StatusForbidden {
		// security related errors are logged as warnings
		level = g.logger.Warn()
	} else if error.NumStatus >= 500 {
		// internal errors are potentially cause for concerned: bugs or third party systems malfunctioning, log as errors
		level = g.logger.Error()
	} else {
		// everything else should be 4xx which indicates mistakes from the client, log as debug
		level = g.logger.Debug()
	}
	if !level.Enabled() {
		return
	}
	l := level.Str(logErrorCode, error.Code).Str(logErrorId, error.Id).Int(logErrorStatus, error.NumStatus)
	if error.Source != nil {
		if error.Source.Header != "" {
			l.Str(logErrorSourceHeader, log.SafeString(error.Source.Header))
		}
		if error.Source.Parameter != "" {
			l.Str(logErrorSourceParameter, log.SafeString(error.Source.Parameter))
		}
		if error.Source.Pointer != "" {
			l.Str(logErrorSourcePointer, log.SafeString(error.Source.Pointer))
		}
	}
	l.Msg(error.Title)
}

func (g Groupware) serveError(w http.ResponseWriter, r *http.Request, error *Error) {
	if error == nil {
		return
	}
	g.log(error)
	w.Header().Add("Content-Type", ContentTypeJsonApi)
	render.Status(r, error.NumStatus)
	w.WriteHeader(error.NumStatus)
	render.Render(w, r, errorResponses(*error))
}

func (g Groupware) withSession(w http.ResponseWriter, r *http.Request, handler func(r Request) Response) (Response, bool) {
	ctx := r.Context()
	sl := g.logger.SubloggerWithRequestID(ctx)
	logger := &sl

	username, ok, err := g.usernameProvider.GetUsername(r, ctx, logger)
	if err != nil {
		g.serveError(w, r, apiError(errorId(r, ctx), ErrorInvalidAuthentication))
		return Response{}, false
	}
	if !ok {
		g.serveError(w, r, apiError(errorId(r, ctx), ErrorMissingAuthentication))
		return Response{}, false
	}

	logger = log.From(logger.With().Str(logUsername, log.SafeString(username)))

	session, ok, err := g.session(username, r, ctx, logger)
	if err != nil {
		logger.Error().Err(err).Interface(logQuery, r.URL.Query()).Msg("failed to determine JMAP session")
		render.Status(r, http.StatusInternalServerError)
		return Response{}, false
	}
	if !ok {
		// no session = authentication failed
		logger.Warn().Err(err).Interface(logQuery, r.URL.Query()).Msg("could not authenticate")
		render.Status(r, http.StatusForbidden)
		return Response{}, false
	}
	decoratedLogger := session.DecorateLogger(*logger)

	req := Request{
		r:       r,
		ctx:     ctx,
		logger:  decoratedLogger,
		session: &session,
	}

	response := handler(req)
	return response, true
}

func (g Groupware) sendResponse(w http.ResponseWriter, r *http.Request, response Response) {
	if response.err != nil {
		g.log(response.err)
		w.Header().Add("Content-Type", ContentTypeJsonApi)
		render.Status(r, response.err.NumStatus)
		render.Render(w, r, errorResponses(*response.err))
		return
	}

	if response.etag != "" {
		w.Header().Add("ETag", response.etag)
	}
	if response.sessionState != "" {
		if response.etag == "" {
			w.Header().Add("ETag", response.sessionState)
		}
		w.Header().Add("Session-State", response.sessionState)
	}

	switch response.body {
	case nil:
		w.WriteHeader(http.StatusNotFound)
	case "":
		w.WriteHeader(http.StatusNoContent)
	default:
		render.Status(r, http.StatusOK)
		render.JSON(w, r, response.body)
	}
}

func (g Groupware) respond(w http.ResponseWriter, r *http.Request, handler func(r Request) Response) {
	response, ok := g.withSession(w, r, handler)
	if !ok {
		return
	}
	g.sendResponse(w, r, response)
}

func (g Groupware) stream(w http.ResponseWriter, r *http.Request, handler func(r Request, w http.ResponseWriter) *Error) {
	ctx := r.Context()
	sl := g.logger.SubloggerWithRequestID(ctx)
	logger := &sl

	username, ok, err := g.usernameProvider.GetUsername(r, ctx, logger)
	if err != nil {
		g.serveError(w, r, apiError(errorId(r, ctx), ErrorInvalidAuthentication))
		return
	}
	if !ok {
		g.serveError(w, r, apiError(errorId(r, ctx), ErrorMissingAuthentication))
		return
	}

	logger = log.From(logger.With().Str(logUsername, log.SafeString(username)))

	session, ok, err := g.session(username, r, ctx, logger)
	if err != nil {
		logger.Error().Err(err).Interface(logQuery, r.URL.Query()).Msg("failed to determine JMAP session")
		render.Status(r, http.StatusInternalServerError)
		return
	}
	if !ok {
		// no session = authentication failed
		logger.Warn().Err(err).Interface(logQuery, r.URL.Query()).Msg("could not authenticate")
		render.Status(r, http.StatusForbidden)
		return
	}
	decoratedLogger := session.DecorateLogger(*logger)

	req := Request{
		r:       r,
		ctx:     ctx,
		logger:  decoratedLogger,
		session: &session,
	}

	apierr := handler(req, w)
	if apierr != nil {
		g.log(apierr)
		w.Header().Add("Content-Type", ContentTypeJsonApi)
		render.Status(r, apierr.NumStatus)
		w.WriteHeader(apierr.NumStatus)
		render.Render(w, r, errorResponses(*apierr))
	}
}

func (g Groupware) NotFound(w http.ResponseWriter, r *http.Request) {
	level := g.logger.Debug()
	if level.Enabled() {
		path := log.SafeString(r.URL.Path)
		level.Str("path", path).Int(logErrorStatus, http.StatusNotFound).Msgf("unmatched path: '%v'", path)
	}
	render.Status(r, http.StatusNotFound)
	w.WriteHeader(http.StatusNotFound)
}
