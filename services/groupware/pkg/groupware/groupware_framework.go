package groupware

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/gorilla/websocket"
	"github.com/miekg/dns"
	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"

	"github.com/prometheus/client_golang/prometheus"

	cmap "github.com/orcaman/concurrent-map"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"

	"github.com/opencloud-eu/opencloud/services/groupware/pkg/config"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/metrics"
)

const (
	logUsername              = "username"
	logUserId                = "user-id"
	logSessionState          = "session-state"
	logAccountId             = "account-id"
	logBlobAccountId         = "blob-account-id" // if the blob accountId is needed as well
	logErrorId               = "error-id"
	logErrorCode             = "code"
	logErrorStatus           = "status"
	logErrorSourceHeader     = "source-header"
	logErrorSourceParameter  = "source-parameter"
	logErrorSourcePointer    = "source-pointer"
	logInvalidQueryParameter = "error-query-param"
	logInvalidPathParameter  = "error-path-param"
	logFolderId              = "folder-id"
	logIdentityId            = "identity-id"
	logQuery                 = "query"
	logEmailId               = "email-id"
	logJobDescription        = "job"
	logJobId                 = "job-id"
	logStreamId              = "stream-id"
	logPath                  = "path"
	logMethod                = "method"
)

// Minimalistic representation of a user, containing only the attributes that are
// necessary for the Groupware implementation.
type user interface {
	GetUsername() string
	GetId() string
}

// Provides a User that is associated with a request.
type userProvider interface {
	// Provide the user for JMAP operations.
	GetUser(req *http.Request, ctx context.Context, logger *log.Logger) (user, error)
}

// Background job that needs to be executed asynchronously by the Groupware.
type Job struct {
	// An identifier for the job, to use in logs for correlation.
	id uint64
	// A human readable description of the job, to use in logs.
	description string
	// The logger to use for the job.
	logger *log.Logger
	// The function that performs the job.
	job func(uint64, *log.Logger)
}

type Groupware struct {
	mux       *chi.Mux
	metrics   *metrics.Metrics
	sseServer *sse.Server
	// A map of all the SSE streams that have been created, in order to be able to iterate over them as,
	// unfortunately, the sse implementation does not provide such a function.
	// Key: the stream ID, which is the username
	// Value: the timestamp of the creation of the stream
	streams             cmap.ConcurrentMap
	logger              *log.Logger
	defaultEmailLimit   uint
	defaultContactLimit uint
	maxBodyValueBytes   uint
	sanitize            bool
	// Caches successful and failed Sessions by the username.
	sessionCache sessionCache
	jmap         *jmap.Client
	userProvider userProvider
	// SSE events that need to be pushed to clients.
	eventChannel chan Event
	// Background jobs that need to be executed.
	jobsChannel chan Job
	// A threadsafe counter to generate the job IDs.
	jobCounter atomic.Uint64
}

// An error during the Groupware initialization.
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

// SSE Event.
type Event struct {
	// The type of event, will be sent as the "type" attribute.
	Type string
	// The ID of the stream to push the event to, typically the username.
	Stream string
	// The payload of the event, will be serialized as JSON.
	Body any
}

// A jmap.HttpJmapApiClientEventListener implementation that records those JMAP
// events as metric increments.
type groupwareHttpJmapApiClientMetricsRecorder struct {
	m *metrics.Metrics
}

var _ jmap.HttpJmapApiClientEventListener = groupwareHttpJmapApiClientMetricsRecorder{}

func (r groupwareHttpJmapApiClientMetricsRecorder) OnSuccessfulRequest(endpoint string, status int) {
	r.m.SuccessfulRequestPerEndpointCounter.With(metrics.Endpoint(endpoint)).Inc()
}
func (r groupwareHttpJmapApiClientMetricsRecorder) OnFailedRequest(endpoint string, err error) {
	r.m.FailedRequestPerEndpointCounter.With(metrics.Endpoint(endpoint)).Inc()
}
func (r groupwareHttpJmapApiClientMetricsRecorder) OnFailedRequestWithStatus(endpoint string, status int) {
	r.m.FailedRequestStatusPerEndpointCounter.With(metrics.EndpointAndStatus(endpoint, status)).Inc()
}
func (r groupwareHttpJmapApiClientMetricsRecorder) OnResponseBodyReadingError(endpoint string, err error) {
	r.m.ResponseBodyReadingErrorPerEndpointCounter.With(metrics.Endpoint(endpoint)).Inc()
}
func (r groupwareHttpJmapApiClientMetricsRecorder) OnResponseBodyUnmarshallingError(endpoint string, err error) {
	r.m.ResponseBodyUnmarshallingErrorPerEndpointCounter.With(metrics.Endpoint(endpoint)).Inc()
}

func NewGroupware(config *config.Config, logger *log.Logger, mux *chi.Mux, prometheusRegistry prometheus.Registerer) (*Groupware, error) {
	baseUrl, err := url.Parse(config.Mail.BaseUrl)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to parse configured Mail.Baseurl '%v'", config.Mail.BaseUrl)
		return nil, GroupwareInitializationError{Message: fmt.Sprintf("failed to parse configured Mail.BaseUrl '%s'", config.Mail.BaseUrl), Err: err}
	}

	sessionUrl := baseUrl.JoinPath(".well-known", "jmap")

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
	defaultContactLimit := max(config.Mail.DefaultContactLimit, 0)
	responseHeaderTimeout := max(config.Mail.ResponseHeaderTimeout, 0)
	sessionCacheMaxCapacity := uint64(max(config.Mail.SessionCache.MaxCapacity, 0))
	sessionCacheTtl := max(config.Mail.SessionCache.Ttl, 0)
	sessionFailureCacheTtl := max(config.Mail.SessionCache.FailureTtl, 0)
	wsHandshakeTimeout := config.Mail.PushHandshakeTimeout

	eventChannelSize := 100 // TODO make channel queue buffering size configurable
	workerQueueSize := 100  // TODO configuration setting
	workerPoolSize := 10    // TODO configuration setting

	keepStreamsAliveInterval := time.Duration(30) * time.Second // TODO configuration, make it 0 to disable keepalive
	sseEventTtl := time.Duration(5) * time.Minute               // TODO configuration setting

	useDnsForSessionResolution := false // TODO configuration setting, although still experimental, needs proper unit tests first

	insecureTls := true // TODO make configurable

	sanitize := true // TODO make configurable

	m := metrics.New(prometheusRegistry, logger)

	// TODO add timeouts and other meaningful configuration settings for the HTTP client
	httpTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpTransport.ResponseHeaderTimeout = responseHeaderTimeout
	if insecureTls {
		tlsConfig := &tls.Config{InsecureSkipVerify: true}
		httpTransport.TLSClientConfig = tlsConfig
	}
	httpClient := *http.DefaultClient
	httpClient.Transport = httpTransport

	userProvider := newRevaContextUsernameProvider()

	jmapMetricsAdapter := groupwareHttpJmapApiClientMetricsRecorder{m: m}

	api := jmap.NewHttpJmapClient(
		&httpClient,
		masterUsername,
		masterPassword,
		jmapMetricsAdapter,
	)

	wsDialer := &websocket.Dialer{
		HandshakeTimeout: wsHandshakeTimeout,
	}
	if insecureTls {
		wsDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	wsf, err := jmap.NewHttpWsClientFactory(wsDialer, masterUsername, masterPassword, logger)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create websocket client")
		return nil, GroupwareInitializationError{Message: "failed to create websocket client", Err: err}
	}

	// api implements all three interfaces:
	jmapClient := jmap.NewClient(api, api, api, wsf)

	sessionCacheBuilder := newSessionCacheBuilder(
		sessionUrl,
		logger,
		jmapClient.FetchSession,
		prometheusRegistry,
		m,
		sessionCacheMaxCapacity,
		sessionCacheTtl,
		sessionFailureCacheTtl,
	)
	if useDnsForSessionResolution {
		conf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
		if err != nil {
			return nil, GroupwareInitializationError{Message: "failed to parse DNS client configuration from /etc/resolv.conf", Err: err}
		}

		var dnsDomainGreenList []string = nil            // TODO domain greenlist from configuration
		var dnsDomainRedList []string = nil              // TODO domain redlist from configuration
		dnsDialTimeout := time.Duration(2) * time.Second // TODO DNS server connection timeout configuration
		dnsReadTimeout := time.Duration(2) * time.Second // TODO DNS server response reading timeout configuration
		defaultDomain := "example.com"                   // TODO default domain when the username is not an email address configuration

		sessionCacheBuilder = sessionCacheBuilder.withDnsAutoDiscovery(
			defaultDomain,
			conf,
			dnsDialTimeout,
			dnsReadTimeout,
			dnsDomainGreenList,
			dnsDomainRedList,
		)
	}

	sessionCache, err := sessionCacheBuilder.build()

	if err != nil {
		// assuming that the error was logged in great detail upstream
		return nil, GroupwareInitializationError{Message: "failed to initialize the session cache", Err: err}
	}
	jmapClient.AddSessionEventListener(sessionCache)

	// A channel to process SSE Events with a single worker.
	eventChannel := make(chan Event, eventChannelSize)
	{
		eventBufferSizeMetric, err := prometheus.NewConstMetric(m.EventBufferSizeDesc, prometheus.GaugeValue, float64(eventChannelSize))
		if err != nil {
			logger.Warn().Err(err).Msgf("failed to create metric %v", m.EventBufferSizeDesc.String())
		} else {
			prometheusRegistry.Register(metrics.ConstMetricCollector{Metric: eventBufferSizeMetric})
		}
		prometheusRegistry.Register(prometheus.NewGaugeFunc(m.EventBufferQueuedOpts, func() float64 {
			return float64(len(eventChannel))
		}))
	}

	sseServer := sse.New()
	sseServer.EventTTL = sseEventTtl
	{
		var sseSubscribers atomic.Int32
		sseServer.OnSubscribe = func(streamID string, sub *sse.Subscriber) {
			sseSubscribers.Add(1)
		}
		sseServer.OnUnsubscribe = func(streamID string, sub *sse.Subscriber) {
			sseSubscribers.Add(-1)
		}
		prometheusRegistry.Register(prometheus.NewGaugeFunc(m.SSESubscribersOpts, func() float64 {
			return float64(sseSubscribers.Load())
		}))
	}

	jobsChannel := make(chan Job, workerQueueSize)
	{
		totalWorkerBufferMetric, err := prometheus.NewConstMetric(m.WorkersBufferSizeDesc, prometheus.GaugeValue, float64(workerQueueSize))
		if err != nil {
			logger.Warn().Err(err).Msgf("failed to create metric %v", m.WorkersBufferSizeDesc.String())
		} else {
			prometheusRegistry.Register(metrics.ConstMetricCollector{Metric: totalWorkerBufferMetric})
		}

		prometheusRegistry.Register(prometheus.NewGaugeFunc(m.WorkersBufferQueuedOpts, func() float64 {
			return float64(len(jobsChannel))
		}))
	}

	var busyWorkers atomic.Int32
	{
		totalWorkersMetric, err := prometheus.NewConstMetric(m.TotalWorkersDesc, prometheus.GaugeValue, float64(workerPoolSize))
		if err != nil {
			logger.Warn().Err(err).Msgf("failed to create metric %v", m.TotalWorkersDesc.String())
		} else {
			prometheusRegistry.Register(metrics.ConstMetricCollector{Metric: totalWorkersMetric})
		}

		prometheusRegistry.Register(prometheus.NewGaugeFunc(m.BusyWorkersOpts, func() float64 {
			return float64(busyWorkers.Load())
		}))
	}

	g := &Groupware{
		mux:                 mux,
		metrics:             m,
		sseServer:           sseServer,
		streams:             cmap.New(),
		logger:              logger,
		sessionCache:        sessionCache,
		userProvider:        userProvider,
		jmap:                &jmapClient,
		defaultEmailLimit:   defaultEmailLimit,
		defaultContactLimit: defaultContactLimit,
		maxBodyValueBytes:   maxBodyValueBytes,
		sanitize:            sanitize,
		eventChannel:        eventChannel,
		jobsChannel:         jobsChannel,
		jobCounter:          atomic.Uint64{},
	}

	for w := 1; w <= workerPoolSize; w++ {
		go g.worker(jobsChannel, &busyWorkers)
	}

	if keepStreamsAliveInterval != 0 {
		ticker := time.NewTicker(keepStreamsAliveInterval)
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

func (g *Groupware) worker(jobs <-chan Job, busy *atomic.Int32) {
	for job := range jobs {
		busy.Add(1)
		before := time.Now()
		logger := log.From(job.logger.With().Str(logJobDescription, job.description).Uint64(logJobId, job.id))
		job.job(job.id, logger)
		if logger.Trace().Enabled() {
			logger.Trace().Msgf("finished job %d [%s] in %v", job.id, job.description, time.Since(before))
		}
		busy.Add(-1)
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
				g.logger.Debug().Str(logStreamId, log.SafeString(ev.Stream)).Msgf("dropped SSE event") // TODO more details
			}
		} else {
			g.logger.Error().Err(err).Msgf("failed to serialize %T body to JSON", ev)
		}
	}
}

func (g *Groupware) push(user user, typ string, body any) {
	g.metrics.SSEEventsCounter.WithLabelValues(typ).Inc()
	g.eventChannel <- Event{Type: typ, Stream: user.GetUsername(), Body: body}
}

func (g *Groupware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

func (g *Groupware) addStream(stream string) bool {
	return g.streams.SetIfAbsent(stream, time.Now())
}

func (g *Groupware) keepStreamsAlive() {
	event := &sse.Event{Comment: []byte("keepalive")}
	g.streams.IterCb(func(stream string, created any) {
		g.sseServer.Publish(stream, event)
	})
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
func (g *Groupware) session(user user, _ *log.Logger) (jmap.Session, bool, *GroupwareError, time.Time) {
	s := g.sessionCache.Get(user.GetUsername())
	if s != nil {
		if s.Success() {
			return s.Get(), true, nil, s.Until()
		} else {
			return jmap.Session{}, false, s.Error(), s.Until()
		}
	}
	// not sure this should happen:
	return jmap.Session{}, false, nil, time.Time{}
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

func (g *Groupware) serveError(w http.ResponseWriter, r *http.Request, error *Error, retryAfter time.Time) {
	if error == nil {
		return
	}
	g.log(error)
	w.Header().Add("Content-Type", ContentTypeJsonApi)
	if !retryAfter.IsZero() {
		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Retry-After
		// either as an absolute timestamp:
		// w.Header().Add("Retry-After", retryAfter.UTC().Format(time.RFC1123))
		// or as a delay in seconds:
		w.Header().Add("Retry-After", fmt.Sprintf("%.0f", time.Until(retryAfter).Seconds()))
	}
	render.Status(r, error.NumStatus)
	w.WriteHeader(error.NumStatus)
	render.Render(w, r, errorResponses(*error))
}

func (g *Groupware) withSession(w http.ResponseWriter, r *http.Request, handler func(r Request) Response) (Response, time.Time, bool) {
	ctx := r.Context()
	sl := g.logger.SubloggerWithRequestID(ctx)
	logger := &sl

	user, err := g.userProvider.GetUser(r, ctx, logger)
	if err != nil {
		g.metrics.AuthenticationFailureCounter.Inc()
		g.serveError(w, r, apiError(errorId(r, ctx), ErrorInvalidAuthentication), time.Time{})
		return Response{}, time.Time{}, false
	}
	if user == nil {
		g.metrics.AuthenticationFailureCounter.Inc()
		g.serveError(w, r, apiError(errorId(r, ctx), ErrorMissingAuthentication), time.Time{})
		return Response{}, time.Time{}, false
	}

	logger = log.From(logger.With().Str(logUserId, log.SafeString(user.GetId())))

	session, ok, gwerr, retryAfter := g.session(user, logger)
	if gwerr != nil {
		g.metrics.SessionFailureCounter.Inc()
		errorId := errorId(r, ctx)
		logger.Error().Str("code", gwerr.Code).Str("error", gwerr.Title).Str("detail", gwerr.Detail).Str(logErrorId, errorId).Msg("failed to determine JMAP session")
		g.serveError(w, r, apiError(errorId, *gwerr), retryAfter)
		return Response{}, retryAfter, false
	}
	if !ok {
		// no session = authentication failed
		g.metrics.SessionFailureCounter.Inc()
		errorId := errorId(r, ctx)
		logger.Error().Str(logErrorId, errorId).Msg("could not authenticate, failed to find Session")
		gwerr = &ErrorInvalidAuthentication
		g.serveError(w, r, apiError(errorId, *gwerr), retryAfter)
		return Response{}, retryAfter, false
	}
	decoratedLogger := decorateLogger(logger, session)

	req := Request{
		g:       g,
		user:    user,
		r:       r,
		ctx:     ctx,
		logger:  decoratedLogger,
		session: &session,
	}

	response := handler(req)
	return response, time.Time{}, true
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

	if response.contentLanguage != "" {
		w.Header().Add("Content-Language", string(response.contentLanguage))
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
	response, _, ok := g.withSession(w, r, handler)
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
		g.serveError(w, r, apiError(errorId(r, ctx), ErrorInvalidAuthentication), time.Time{})
		return
	}
	if user == nil {
		g.serveError(w, r, apiError(errorId(r, ctx), ErrorMissingAuthentication), time.Time{})
		return
	}

	logger = log.From(logger.With().Str(logUserId, log.SafeString(user.GetId())))

	session, ok, gwerr, retryAfter := g.session(user, logger)
	if gwerr != nil {
		errorId := errorId(r, ctx)
		logger.Error().Str("code", gwerr.Code).Str("error", gwerr.Title).Str("detail", gwerr.Detail).Str(logErrorId, errorId).Msg("failed to determine JMAP session")
		g.serveError(w, r, apiError(errorId, *gwerr), retryAfter)
		return
	}
	if !ok {
		// no session = authentication failed
		errorId := errorId(r, ctx)
		logger.Error().Str(logErrorId, errorId).Msg("could not authenticate, failed to find Session")
		gwerr = &ErrorInvalidAuthentication
		g.serveError(w, r, apiError(errorId, *gwerr), retryAfter)
		return
	}

	decoratedLogger := decorateLogger(logger, session)

	req := Request{
		g:       g,
		user:    user,
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
		level.Str(logPath, path).Str(logMethod, method).Int(logErrorStatus, http.StatusNotFound).Msgf("unmatched path: '%v'", path)
	}
	w.Header().Add("Unmatched-Path", r.URL.Path) // TODO possibly remove this in production for security reasons?
	w.WriteHeader(http.StatusNotFound)
}

func (g *Groupware) MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	level := g.logger.Debug()
	if level.Enabled() {
		path := log.SafeString(r.URL.Path)
		method := log.SafeString(r.Method)
		level.Str(logPath, path).Str(logMethod, method).Int(logErrorStatus, http.StatusNotFound).Msgf("method not allowed: '%v'", method)
	}
	w.Header().Add("Unsupported-Method", r.Method) // TODO possibly remove this in production for security reasons?
	w.WriteHeader(http.StatusNotFound)
}
