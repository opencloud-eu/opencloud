package groupware

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

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

const (
	logMaxStrLength = 512
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
	sessionCache *ttlcache.Cache[string, cachedSession]
}

func (l GroupwareSessionEventListener) OnSessionOutdated(session *jmap.Session) {
	// it's enough to remove the session from the cache, as it will be fetched on-demand
	// the next time an operation is performed on behalf of the user
	l.sessionCache.Delete(session.Username)
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

	jmapClient := jmap.NewClient(api, api)

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

	sessionEventListener := GroupwareSessionEventListener{sessionCache: sessionCache}
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
func (g Groupware) session(username string, req *http.Request, ctx context.Context, logger *log.Logger) (jmap.Session, bool, error) {
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
			withDetail(fmt.Sprintf("The account '%v' does not exist", logstr(accountId))),
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
		msg := fmt.Sprintf("Invalid value for query parameter '%v': '%s': %s", param, logstr(str), err.Error())
		return defaultValue, true, apiError(errorId, ErrorInvalidRequestParameter,
			withDetail(msg),
			withSource(&ErrorSource{Parameter: param}),
		)
	}
	return int(value), true, nil
}

// Safely caps a string to a given size to avoid log bombing.
// Use this function to wrap strings that are user input (HTTP headers, path parameters, URI parameters, HTTP body, ...).
func logstr(text string) string {
	runes := []rune(text)

	if len(runes) <= logMaxStrLength {
		return text
	} else {
		return string(runes[0:logMaxStrLength-1]) + `\u2026` // hellip
	}
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
			l.Str(logErrorSourceHeader, logstr(error.Source.Header))
		}
		if error.Source.Parameter != "" {
			l.Str(logErrorSourceParameter, logstr(error.Source.Parameter))
		}
		if error.Source.Pointer != "" {
			l.Str(logErrorSourcePointer, logstr(error.Source.Pointer))
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

func (g Groupware) respond(w http.ResponseWriter, r *http.Request, handler func(r Request) (any, string, *Error)) {
	ctx := r.Context()
	logger := g.logger.SubloggerWithRequestID(ctx)

	username, ok, err := g.usernameProvider.GetUsername(r, ctx, &logger)
	if err != nil {
		g.serveError(w, r, apiError(errorId(r, ctx), ErrorInvalidAuthentication))
		return
	}
	if !ok {
		g.serveError(w, r, apiError(errorId(r, ctx), ErrorMissingAuthentication))
		return
	}

	logger = log.Logger{Logger: logger.With().Str(logUsername, logstr(username)).Logger()}

	session, ok, err := g.session(username, r, ctx, &logger)
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
	logger = session.DecorateLogger(logger)

	req := Request{
		r:       r,
		ctx:     ctx,
		logger:  &logger,
		session: &session,
	}

	response, state, apierr := handler(req)
	if apierr != nil {
		g.log(apierr)
		w.Header().Add("Content-Type", ContentTypeJsonApi)
		render.Status(r, apierr.NumStatus)
		w.WriteHeader(apierr.NumStatus)
		render.Render(w, r, errorResponses(*apierr))
		return
	}

	if state != "" {
		w.Header().Add("ETag", state)
	}
	if response == nil {
		render.Status(r, http.StatusNotFound)
		w.WriteHeader(http.StatusNotFound)
	} else {
		render.Status(r, http.StatusOK)
		render.JSON(w, r, response)
	}
}

/*
func (g Groupware) withSession(w http.ResponseWriter, r *http.Request, handler func(r Request) (any, string, error)) (any, string, error) {
	ctx := r.Context()
	logger := g.logger.SubloggerWithRequestID(ctx)
	session, ok, err := g.session(r, ctx, &logger)
	if err != nil {
		logger.Error().Err(err).Interface(logQuery, r.URL.Query()).Msg("failed to determine JMAP session")
		return nil, "", err
	}
	if !ok {
		// no session = authentication failed
		logger.Warn().Err(err).Interface(logQuery, r.URL.Query()).Msg("could not authenticate")
		return nil, "", err
	}
	logger = session.DecorateLogger(logger)

	req := Request{
		r:       r,
		ctx:     ctx,
		logger:  &logger,
		session: &session,
	}

	response, state, err := handler(req)
	if err != nil {
		logger.Error().Err(err).Interface(logQuery, r.URL.Query()).Msg(err.Error())
	}
	return response, state, err
}
*/

func (g Groupware) NotFound(w http.ResponseWriter, r *http.Request) {
	level := g.logger.Debug()
	if level.Enabled() {
		path := logstr(r.URL.Path)
		level.Str("path", path).Int(logErrorStatus, http.StatusNotFound).Msgf("unmatched path: '%v'", path)
	}
	render.Status(r, http.StatusNotFound)
	w.WriteHeader(http.StatusNotFound)
}
