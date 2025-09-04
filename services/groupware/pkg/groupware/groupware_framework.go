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
	"github.com/miekg/dns"
	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/jellydator/ttlcache/v3"

	cmap "github.com/orcaman/concurrent-map"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"

	"github.com/opencloud-eu/opencloud/services/groupware/pkg/config"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/metrics"
)

const (
	logUsername              = "username" // this should match jmap.logUsername to avoid having the field twice in the logs under different keys
	logUserId                = "user-id"
	logAccountId             = "account-id"
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
	logJobDescription        = "job"
	logJobId                 = "job-id"
	logStreamId              = "stream-id"
	logPath                  = "path"
	logMethod                = "method"
)

// Minimalistic representation of a User, containing only the attributes that are
// necessary for the Groupware implementation.
type User interface {
	GetUsername() string
	GetId() string
}

// Provides a User that is associated with a request.
type UserProvider interface {
	// Provide the user for JMAP operations.
	GetUser(req *http.Request, ctx context.Context, logger *log.Logger) (User, error)
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
	streams           cmap.ConcurrentMap
	logger            *log.Logger
	defaultEmailLimit uint
	maxBodyValueBytes uint
	// Caches successful and failed Sessions by the username.
	sessionCache *ttlcache.Cache[sessionKey, cachedSession]
	jmap         *jmap.Client
	userProvider UserProvider
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

type groupwareHttpJmapApiClientMetricsRecorder struct {
	m *metrics.Metrics
}

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

var _ jmap.HttpJmapApiClientEventListener = groupwareHttpJmapApiClientMetricsRecorder{}

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
	responseHeaderTimeout := max(config.Mail.ResponseHeaderTimeout, 0)
	sessionCacheMaxCapacity := uint64(max(config.Mail.SessionCache.MaxCapacity, 0))
	sessionCacheTtl := max(config.Mail.SessionCache.Ttl, 0)
	sessionFailureCacheTtl := max(config.Mail.SessionCache.FailureTtl, 0)

	eventChannelSize := 100 // TODO make channel queue buffering size configurable
	workerQueueSize := 100  // TODO configuration setting
	workerPoolSize := 10    // TODO configuration setting

	keepStreamsAliveInterval := time.Duration(30) * time.Second // TODO configuration, make it 0 to disable keepalive
	sseEventTtl := time.Duration(5) * time.Minute               // TODO configuration setting

	useDnsForSessionResolution := false // TODO configuration setting, although still experimental, needs proper unit tests first

	insecureTls := true // TODO make configurable

	m := metrics.New(prometheusRegistry, logger)

	// TODO add timeouts and other meaningful configuration settings for the HTTP client
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.ResponseHeaderTimeout = responseHeaderTimeout
	if insecureTls {
		tlsConfig := &tls.Config{InsecureSkipVerify: true} // TODO make configurable
		tr.TLSClientConfig = tlsConfig
	}
	c := *http.DefaultClient
	c.Transport = tr

	userProvider := NewRevaContextUsernameProvider()

	jmapMetricsAdapter := groupwareHttpJmapApiClientMetricsRecorder{m: m}

	api := jmap.NewHttpJmapClient(
		&c,
		masterUsername,
		masterPassword,
		jmapMetricsAdapter,
	)

	jmapClient := jmap.NewClient(api, api, api)

	var sessionCache *ttlcache.Cache[sessionKey, cachedSession]
	{
		sessionUrlResolver := func(_ string) (*url.URL, *GroupwareError) {
			return sessionUrl, nil
		}
		if useDnsForSessionResolution {
			defaultSessionDomain := "example.com" // TODO default domain from configuration
			// TODO resolv.conf or other configuration
			conf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
			if err != nil {
				return nil, GroupwareInitializationError{Message: "failed to parse DNS client configuration from /etc/resolv.conf", Err: err}
			}

			var domainGreenList []string = nil // TODO domain greenlist from configuration
			var domainRedList []string = nil   // TODO domain redlist from configuration

			dialTimeout := time.Duration(2) * time.Second // TODO configuration
			readTimeout := time.Duration(2) * time.Second // TODO configuration

			dnsSessionUrlResolver, err := NewDnsSessionUrlResolver(
				sessionUrl,
				defaultSessionDomain,
				conf,
				domainGreenList,
				domainRedList,
				dialTimeout,
				readTimeout,
			)
			if err != nil {
				return nil, GroupwareInitializationError{Message: "failed to instantiate the DNS session URL resolver", Err: err}
			}
			sessionUrlResolver = dnsSessionUrlResolver.Resolve
		}

		sessionLoader := &sessionCacheLoader{
			logger:             logger,
			jmapClient:         &jmapClient,
			errorTtl:           sessionFailureCacheTtl,
			sessionUrlProvider: sessionUrlResolver,
		}

		sessionCache = ttlcache.New(
			ttlcache.WithCapacity[sessionKey, cachedSession](sessionCacheMaxCapacity),
			ttlcache.WithTTL[sessionKey, cachedSession](sessionCacheTtl),
			ttlcache.WithDisableTouchOnHit[sessionKey, cachedSession](),
			ttlcache.WithLoader(sessionLoader),
		)
		go sessionCache.Start()

		prometheusRegistry.Register(sessionCacheMetricsCollector{desc: m.SessionCacheDesc, supply: sessionCache.Metrics})
	}

	sessionCache.OnEviction(func(c context.Context, r ttlcache.EvictionReason, item *ttlcache.Item[sessionKey, cachedSession]) {
		if logger.Trace().Enabled() {
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
			tipe := "successful"
			if !item.Value().Success() {
				tipe = "failed"
			}
			logger.Trace().Msgf("%s session cache eviction of user '%v' after %v: %v", tipe, item.Key(), spentInCache, reason)
		}
	})

	sessionEventListener := sessionEventListener{
		sessionCache: sessionCache,
		logger:       logger,
		counter:      m.OutdatedSessionsCounter,
	}
	jmapClient.AddSessionEventListener(&sessionEventListener)

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
		mux:               mux,
		metrics:           m,
		sseServer:         sseServer,
		streams:           cmap.New(),
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

func (g *Groupware) push(user User, typ string, body any) {
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
func (g *Groupware) session(user User, _ *http.Request, _ context.Context, _ *log.Logger) (jmap.Session, bool, *GroupwareError) {
	item := g.sessionCache.Get(toSessionKey(user.GetUsername()))
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
		g.metrics.AuthenticationFailureCounter.Inc()
		g.serveError(w, r, apiError(errorId(r, ctx), ErrorInvalidAuthentication))
		return Response{}, false
	}
	if user == nil {
		g.metrics.AuthenticationFailureCounter.Inc()
		g.serveError(w, r, apiError(errorId(r, ctx), ErrorMissingAuthentication))
		return Response{}, false
	}

	logger = log.From(logger.With().Str(logUserId, log.SafeString(user.GetId())))

	session, ok, gwerr := g.session(user, r, ctx, logger)
	if gwerr != nil {
		g.metrics.SessionFailureCounter.Inc()
		errorId := errorId(r, ctx)
		logger.Error().Str("code", gwerr.Code).Str("error", gwerr.Title).Str("detail", gwerr.Detail).Str(logErrorId, errorId).Msg("failed to determine JMAP session")
		g.serveError(w, r, apiError(errorId, *gwerr))
		return Response{}, false
	}
	if !ok {
		// no session = authentication failed
		g.metrics.SessionFailureCounter.Inc()
		errorId := errorId(r, ctx)
		logger.Error().Str(logErrorId, errorId).Msg("could not authenticate, failed to find Session")
		gwerr = &ErrorInvalidAuthentication
		g.serveError(w, r, apiError(errorId, *gwerr))
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

	logger = log.From(logger.With().Str(logUserId, log.SafeString(user.GetId())))

	session, ok, gwerr := g.session(user, r, ctx, logger)
	if gwerr != nil {
		errorId := errorId(r, ctx)
		logger.Error().Str("code", gwerr.Code).Str("error", gwerr.Title).Str("detail", gwerr.Detail).Str(logErrorId, errorId).Msg("failed to determine JMAP session")
		g.serveError(w, r, apiError(errorId, *gwerr))
		return
	}
	if !ok {
		// no session = authentication failed
		errorId := errorId(r, ctx)
		logger.Error().Str(logErrorId, errorId).Msg("could not authenticate, failed to find Session")
		gwerr = &ErrorInvalidAuthentication
		g.serveError(w, r, apiError(errorId, *gwerr))
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
		level.Str(logPath, path).Str(logMethod, method).Int(logErrorStatus, http.StatusNotFound).Msgf("unmatched path: '%v'", path)
	}
	w.WriteHeader(http.StatusNotFound)
}

func (g *Groupware) MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	level := g.logger.Debug()
	if level.Enabled() {
		path := log.SafeString(r.URL.Path)
		method := log.SafeString(r.Method)
		level.Str(logPath, path).Str(logMethod, method).Int(logErrorStatus, http.StatusNotFound).Msgf("method not allowed: '%v'", method)
	}
	w.WriteHeader(http.StatusNotFound)
}
