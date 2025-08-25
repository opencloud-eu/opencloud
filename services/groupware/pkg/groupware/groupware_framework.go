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
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"

	"github.com/jellydator/ttlcache/v3"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/config"
)

const (
	logUsername              = "username" // this should match jmap.logUsername to avoid having the field twice in the logs under different keys
	logUserId                = "user-id"
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
	logEmailId               = "email-id"
)

type User interface {
	GetUsername() string
	GetId() string
}

type UserProvider interface {
	// Provide the user for JMAP operations.
	GetUser(req *http.Request, ctx context.Context, logger *log.Logger) (User, error)
}

type Job struct {
	id          uint64
	description string
	logger      *log.Logger
	job         func(uint64, *log.Logger)
}

type Groupware struct {
	mux               *chi.Mux
	sseServer         *sse.Server
	streams           map[string]time.Time
	streamsLock       sync.Mutex
	logger            *log.Logger
	defaultEmailLimit uint
	maxBodyValueBytes uint
	sessionCache      *ttlcache.Cache[string, cachedSession]
	jmap              *jmap.Client
	userProvider      UserProvider
	eventChannel      chan Event
	jobsChannel       chan Job
	jobCounter        atomic.Uint64
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

func (l GroupwareSessionEventListener) OnSessionOutdated(session *jmap.Session, newSessionState jmap.SessionState) {
	// it's enough to remove the session from the cache, as it will be fetched on-demand
	// the next time an operation is performed on behalf of the user
	l.sessionCache.Delete(session.Username)

	l.logger.Trace().Msgf("removed outdated session for user '%v': state %v -> %v", session.Username, session.State, newSessionState)
}

var _ jmap.SessionEventListener = GroupwareSessionEventListener{}

type Event struct {
	Type   string
	Stream string
	Body   any
}

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

	keepStreamsAlive := true // TODO configuration

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.ResponseHeaderTimeout = responseHeaderTimeout
	tlsConfig := &tls.Config{InsecureSkipVerify: true} // TODO make configurable
	tr.TLSClientConfig = tlsConfig
	c := *http.DefaultClient
	c.Transport = tr

	userProvider := NewRevaContextUsernameProvider()

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
				reason = fmt.Sprintf("expired after %v", item.TTL())
			case ttlcache.EvictionReasonMaxCostExceeded:
				reason = "max cost exceeded"
			}
			if reason == "" {
				reason = fmt.Sprintf("unknown (%v)", r)
			}
			spentInCache := time.Since(item.Value().Since())

			logger.Trace().Msgf("session cache eviction of user '%v' after %v: %v", item.Key(), spentInCache, reason)
		})
	}

	sessionEventListener := GroupwareSessionEventListener{sessionCache: sessionCache, logger: logger}
	jmapClient.AddSessionEventListener(&sessionEventListener)

	eventChannel := make(chan Event, 100) // TODO make channel queue buffering size configurable

	sseServer := sse.New()
	sseServer.EventTTL = time.Duration(5) * time.Minute // TODO configuration setting

	workerQueueSize := 100 // TODO configuration setting
	workerPoolSize := 10   // TODO configuration setting
	jobsChannel := make(chan Job, workerQueueSize)

	g := &Groupware{
		mux:               mux,
		sseServer:         sseServer,
		streams:           map[string]time.Time{},
		streamsLock:       sync.Mutex{},
		logger:            logger,
		sessionCache:      sessionCache,
		userProvider:      userProvider,
		jmap:              &jmapClient,
		defaultEmailLimit: defaultEmailLimit,
		maxBodyValueBytes: maxBodyValueBytes,
		eventChannel:      eventChannel,
		jobsChannel:       jobsChannel,
		jobCounter:        atomic.Uint64{},
	}

	for w := 1; w <= workerPoolSize; w++ {
		go g.worker(jobsChannel)
	}

	if keepStreamsAlive {
		ticker := time.NewTicker(time.Duration(30) * time.Second) // TODO configuration
		//defer ticker.Stop()
		go func() {
			for range ticker.C {
				g.keepStreamsAlive()
			}
		}()
	}

	go g.listenForEvents()

	return g, nil
}

func (g *Groupware) worker(jobs <-chan Job) {
	for job := range jobs {
		before := time.Now()
		logger := log.From(job.logger.With().Str("job", job.description).Uint64("job-id", job.id))
		job.job(job.id, logger)
		logger.Trace().Msgf("finished job %d [%s] in %v", job.id, job.description, time.Since(before)) // TODO remove
	}
}

func (g *Groupware) job(logger *log.Logger, description string, f func(uint64, *log.Logger)) uint64 {
	id := g.jobCounter.Add(1)
	before := time.Now()
	g.jobsChannel <- Job{id: id, description: description, logger: logger, job: f}
	g.logger.Trace().Msgf("pushed job %d [%s] in %v", id, description, time.Since(before)) // TODO remove
	return id
}

func (g *Groupware) listenForEvents() {
	for ev := range g.eventChannel {
		data, err := json.Marshal(ev.Body)
		if err == nil {
			published := g.sseServer.TryPublish(ev.Stream, &sse.Event{
				Event: []byte(ev.Type),
				Data:  data,
			})
			if !published && g.logger.Debug().Enabled() {
				g.logger.Debug().Str("stream", log.SafeString(ev.Stream)).Msgf("dropped SSE event") // TODO more details
			}
		} else {
			g.logger.Error().Err(err).Msgf("failed to serialize %T body to JSON", ev)
		}
	}
}

func (g *Groupware) push(user User, typ string, body any) {
	g.eventChannel <- Event{Type: typ, Stream: user.GetUsername(), Body: body}
}

func (g *Groupware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

func (g *Groupware) addStream(stream string) bool {
	g.streamsLock.Lock()
	defer g.streamsLock.Unlock()
	_, ok := g.streams[stream]
	if ok {
		return false
	}
	g.streams[stream] = time.Now()
	return true
}

func (g *Groupware) keepStreamsAlive() {
	event := &sse.Event{Comment: []byte("keepalive")}
	g.streamsLock.Lock()
	defer g.streamsLock.Unlock()
	for stream := range g.streams {
		g.sseServer.Publish(stream, event)
	}
}

func (g *Groupware) ServeSSE(w http.ResponseWriter, r *http.Request) {
	g.withSession(w, r, func(req Request) Response {
		stream := req.GetUser().GetUsername()

		if g.addStream(stream) {
			str := g.sseServer.CreateStream(stream)
			if g.logger.Trace().Enabled() {
				g.logger.Trace().Msgf("created stream '%v'", log.SafeString(str.ID))
			}
		}

		q := r.URL.Query()
		q.Set("stream", stream)
		r.URL.RawQuery = q.Encode()

		g.sseServer.ServeHTTP(w, r)
		return Response{}
	})
}

// Provide a JMAP Session for the
func (g *Groupware) session(user User, _ *http.Request, _ context.Context, _ *log.Logger) (jmap.Session, bool, error) {
	item := g.sessionCache.Get(user.GetUsername())
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
	g       *Groupware
	user    User
	r       *http.Request
	ctx     context.Context
	logger  *log.Logger
	session *jmap.Session
}

type Response struct {
	body         any
	status       int
	err          *Error
	etag         jmap.State
	sessionState jmap.SessionState
}

func errorResponse(err *Error) Response {
	return Response{
		body:         nil,
		err:          err,
		etag:         "",
		sessionState: "",
	}
}

func response(body any, sessionState jmap.SessionState) Response {
	return Response{
		body:         body,
		err:          nil,
		etag:         jmap.State(sessionState),
		sessionState: sessionState,
	}
}

func etagResponse(body any, sessionState jmap.SessionState, etag jmap.State) Response {
	return Response{
		body:         body,
		err:          nil,
		etag:         etag,
		sessionState: sessionState,
	}
}

func etagOnlyResponse(body any, etag jmap.State) Response {
	return Response{
		body:         body,
		err:          nil,
		etag:         etag,
		sessionState: "",
	}
}

func noContentResponse(sessionState jmap.SessionState) Response {
	return Response{
		body:         nil,
		status:       http.StatusNoContent,
		err:          nil,
		etag:         jmap.State(sessionState),
		sessionState: sessionState,
	}
}

func acceptedResponse(sessionState jmap.SessionState) Response {
	return Response{
		body:         nil,
		status:       http.StatusAccepted,
		err:          nil,
		etag:         jmap.State(sessionState),
		sessionState: sessionState,
	}
}

func timeoutResponse(sessionState jmap.SessionState) Response {
	return Response{
		body:         nil,
		status:       http.StatusRequestTimeout,
		err:          nil,
		etag:         "",
		sessionState: sessionState,
	}
}

func notFoundResponse(sessionState jmap.SessionState) Response {
	return Response{
		body:         nil,
		status:       http.StatusNotFound,
		err:          nil,
		etag:         jmap.State(sessionState),
		sessionState: sessionState,
	}
}

func (r Request) push(typ string, event any) {
	r.g.push(r.user, typ, event)
}

func (r Request) GetUser() User {
	return r.user
}

func (r Request) GetRequestId() string {
	return chimiddleware.GetReqID(r.ctx)
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
		// don't include the original error, as it leaks too much about our implementation, e.g.:
		// strconv.ParseInt: parsing \"a\": invalid syntax
		msg := fmt.Sprintf("Invalid numeric value for query parameter '%v': '%s'", param, log.SafeString(str))
		return defaultValue, true, apiError(errorId, ErrorInvalidRequestParameter,
			withDetail(msg),
			withSource(&ErrorSource{Parameter: param}),
		)
	}
	return int(value), true, nil
}

func (r Request) parseUNumericParam(param string, defaultValue uint) (uint, bool, *Error) {
	q := r.r.URL.Query()
	if !q.Has(param) {
		return defaultValue, false, nil
	}

	str := q.Get(param)
	if str == "" {
		return defaultValue, false, nil
	}

	value, err := strconv.ParseUint(str, 10, 0)
	if err != nil {
		errorId := r.errorId()
		// don't include the original error, as it leaks too much about our implementation, e.g.:
		// strconv.ParseInt: parsing \"a\": invalid syntax
		msg := fmt.Sprintf("Invalid numeric value for query parameter '%v': '%s'", param, log.SafeString(str))
		return defaultValue, true, apiError(errorId, ErrorInvalidRequestParameter,
			withDetail(msg),
			withSource(&ErrorSource{Parameter: param}),
		)
	}
	return uint(value), true, nil
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

func (g *Groupware) log(error *Error) {
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

func (g *Groupware) serveError(w http.ResponseWriter, r *http.Request, error *Error) {
	if error == nil {
		return
	}
	g.log(error)
	w.Header().Add("Content-Type", ContentTypeJsonApi)
	render.Status(r, error.NumStatus)
	w.WriteHeader(error.NumStatus)
	render.Render(w, r, errorResponses(*error))
}

func (g *Groupware) withSession(w http.ResponseWriter, r *http.Request, handler func(r Request) Response) (Response, bool) {
	ctx := r.Context()
	sl := g.logger.SubloggerWithRequestID(ctx)
	logger := &sl

	user, err := g.userProvider.GetUser(r, ctx, logger)
	if err != nil {
		g.serveError(w, r, apiError(errorId(r, ctx), ErrorInvalidAuthentication))
		return Response{}, false
	}
	if user == nil {
		g.serveError(w, r, apiError(errorId(r, ctx), ErrorMissingAuthentication))
		return Response{}, false
	}

	logger = log.From(logger.With().Str(logUsername, log.SafeString(user.GetUsername())).Str(logUserId, log.SafeString(user.GetId())))

	session, ok, err := g.session(user, r, ctx, logger)
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
		g:       g,
		user:    user,
		r:       r,
		ctx:     ctx,
		logger:  decoratedLogger,
		session: &session,
	}

	response := handler(req)
	return response, true
}

func (g *Groupware) sendResponse(w http.ResponseWriter, r *http.Request, response Response) {
	if response.err != nil {
		g.log(response.err)
		w.Header().Add("Content-Type", ContentTypeJsonApi)
		render.Status(r, response.err.NumStatus)
		render.Render(w, r, errorResponses(*response.err))
		return
	}

	etag := ""
	sessionState := ""

	if response.etag != "" {
		etag = string(response.etag)
	}

	if response.sessionState != "" {
		sessionState = string(response.sessionState)
		if etag == "" {
			etag = sessionState
		}
	}

	if sessionState != "" {
		w.Header().Add("Session-State", string(sessionState))
	}

	notModified := false
	if etag != "" {
		challenge := r.Header.Get("if-none-match")
		quotedEtag := "\"" + etag + "\""
		notModified = challenge != "" && (challenge == etag || challenge == quotedEtag)
		w.Header().Add("ETag", quotedEtag)
	}

	if notModified {
		w.WriteHeader(http.StatusNotModified)
	} else {
		switch response.body {
		case nil, "":
			w.WriteHeader(response.status)
		default:
			render.Status(r, http.StatusOK)
			render.JSON(w, r, response.body)
		}
	}
}

func (g *Groupware) respond(w http.ResponseWriter, r *http.Request, handler func(r Request) Response) {
	response, ok := g.withSession(w, r, handler)
	if !ok {
		return
	}
	g.sendResponse(w, r, response)
}

func (g *Groupware) stream(w http.ResponseWriter, r *http.Request, handler func(r Request, w http.ResponseWriter) *Error) {
	ctx := r.Context()
	sl := g.logger.SubloggerWithRequestID(ctx)
	logger := &sl

	user, err := g.userProvider.GetUser(r, ctx, logger)
	if err != nil {
		g.serveError(w, r, apiError(errorId(r, ctx), ErrorInvalidAuthentication))
		return
	}
	if user == nil {
		g.serveError(w, r, apiError(errorId(r, ctx), ErrorMissingAuthentication))
		return
	}

	logger = log.From(logger.With().Str(logUsername, log.SafeString(user.GetUsername())).Str(logUserId, log.SafeString(user.GetId())))

	session, ok, err := g.session(user, r, ctx, logger)
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

func (g *Groupware) NotFound(w http.ResponseWriter, r *http.Request) {
	level := g.logger.Debug()
	if level.Enabled() {
		path := log.SafeString(r.URL.Path)
		method := log.SafeString(r.Method)
		level.Str("path", path).Str("method", method).Int(logErrorStatus, http.StatusNotFound).Msgf("unmatched path: '%v'", path)
	}
	w.WriteHeader(http.StatusNotFound)
}

func (g *Groupware) MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	level := g.logger.Debug()
	if level.Enabled() {
		path := log.SafeString(r.URL.Path)
		method := log.SafeString(r.Method)
		level.Str("path", path).Str("method", method).Int(logErrorStatus, http.StatusNotFound).Msgf("method not allowed: '%v'", method)
	}
	w.WriteHeader(http.StatusNotFound)
}
