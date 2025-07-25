package groupware

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/jellydator/ttlcache/v3"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/config"
)

const (
	logFolderId = "folder-id"
	logQuery    = "query"
)

type cachedSession interface {
	Success() bool
	Get() jmap.Session
	Error() error
}

type succeededSession struct {
	session jmap.Session
}

func (s succeededSession) Success() bool {
	return true
}
func (s succeededSession) Get() jmap.Session {
	return s.session
}
func (s succeededSession) Error() error {
	return nil
}

var _ cachedSession = succeededSession{}

type failedSession struct {
	err error
}

func (s failedSession) Success() bool {
	return false
}
func (s failedSession) Get() jmap.Session {
	panic("this should never be called")
}
func (s failedSession) Error() error {
	return s.err
}

var _ cachedSession = failedSession{}

type sessionCacheLoader struct {
	logger     *log.Logger
	jmapClient jmap.Client
	errorTtl   time.Duration
}

func (l *sessionCacheLoader) Load(c *ttlcache.Cache[string, cachedSession], username string) *ttlcache.Item[string, cachedSession] {
	session, err := l.jmapClient.FetchSession(username, l.logger)
	if err != nil {
		l.logger.Warn().Str("username", username).Err(err).Msgf("failed to create session for '%v'", username)
		return c.Set(username, failedSession{err: err}, l.errorTtl)
	} else {
		l.logger.Debug().Str("username", username).Msgf("successfully created session for '%v'", username)
		return c.Set(username, succeededSession{session: session}, 0) // 0 = use the TTL configured on the Cache
	}
}

var _ ttlcache.Loader[string, cachedSession] = &sessionCacheLoader{}

type Groupware struct {
	mux               *chi.Mux
	logger            *log.Logger
	defaultEmailLimit int
	maxBodyValueBytes int
	sessionCache      *ttlcache.Cache[string, cachedSession]
	jmap              jmap.Client
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

	defaultEmailLimit := config.Mail.DefaultEmailLimit
	if defaultEmailLimit < 0 {
		defaultEmailLimit = 0
	}
	maxBodyValueBytes := config.Mail.MaxBodyValueBytes
	if maxBodyValueBytes < 0 {
		maxBodyValueBytes = 0
	}

	responseHeaderTimeout := config.Mail.ResponseHeaderTimeout
	if responseHeaderTimeout < 0 {
		responseHeaderTimeout = 0
	}
	sessionCacheTtl := config.Mail.SessionCacheTtl
	if sessionCacheTtl < 0 {
		sessionCacheTtl = 0
	}
	sessionFailureCacheTtl := config.Mail.SessionFailureCacheTtl
	if sessionFailureCacheTtl < 0 {
		sessionFailureCacheTtl = 0
	}

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
			jmapClient: jmapClient,
			errorTtl:   sessionFailureCacheTtl,
		}

		sessionCache = ttlcache.New(
			ttlcache.WithTTL[string, cachedSession](
				sessionCacheTtl,
			),
			ttlcache.WithDisableTouchOnHit[string, cachedSession](),
			ttlcache.WithLoader(sessionLoader),
		)
		go sessionCache.Start()
	}

	return &Groupware{
		mux:               mux,
		logger:            logger,
		sessionCache:      sessionCache,
		usernameProvider:  usernameProvider,
		jmap:              jmapClient,
		defaultEmailLimit: defaultEmailLimit,
		maxBodyValueBytes: maxBodyValueBytes,
	}, nil
}

func (g Groupware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

func (g Groupware) session(req *http.Request, ctx context.Context, logger *log.Logger) (jmap.Session, bool, error) {
	username, ok, err := g.usernameProvider.GetUsername(req, ctx, logger)
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve username")
		return jmap.Session{}, false, err
	}
	if !ok {
		logger.Debug().Msg("unauthenticated API access attempt")
		return jmap.Session{}, false, nil
	}

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

func (g Groupware) respond(w http.ResponseWriter, r *http.Request,
	handler func(r *http.Request, ctx context.Context, logger *log.Logger, session *jmap.Session) (any, string, error)) {
	ctx := r.Context()
	logger := g.logger.SubloggerWithRequestID(ctx)
	session, ok, err := g.session(r, ctx, &logger)
	if err != nil {
		logger.Error().Err(err).Interface(logQuery, r.URL.Query()).Msg("failed to determine JMAP session")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok {
		// no session = authentication failed
		logger.Warn().Err(err).Interface(logQuery, r.URL.Query()).Msg("could not authenticate")
		w.WriteHeader(http.StatusForbidden)
		return
	}
	logger = session.DecorateLogger(logger)

	response, state, err := handler(r, ctx, &logger, &session)
	if err != nil {
		logger.Error().Err(err).Interface(logQuery, r.URL.Query()).Msg(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if state != "" {
		w.Header().Add("ETag", state)
	}
	if response == nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		render.Status(r, http.StatusOK)
		render.JSON(w, r, response)
	}
}

func (g Groupware) withSession(w http.ResponseWriter, r *http.Request,
	handler func(r *http.Request, ctx context.Context, logger *log.Logger, session *jmap.Session) (any, string, error)) (any, string, error) {
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

	response, state, err := handler(r, ctx, &logger, &session)
	if err != nil {
		logger.Error().Err(err).Interface(logQuery, r.URL.Query()).Msg(err.Error())
	}
	return response, state, err
}
