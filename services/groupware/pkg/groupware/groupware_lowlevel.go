package groupware

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"

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
			jmapClient: jmapClient,
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

// using a wrapper class for requests, to group multiple parameters, really to avoid crowding the
// API of handlers but also to make it easier to expand it in the future without having to modify
// the parameter list of every single handler function
type Request struct {
	r       *http.Request
	ctx     context.Context
	logger  *log.Logger
	session *jmap.Session
}

func (g Groupware) respond(w http.ResponseWriter, r *http.Request, handler func(r Request) (any, string, *ApiError)) {
	ctx := r.Context()
	logger := g.logger.SubloggerWithRequestID(ctx)
	session, ok, err := g.session(r, ctx, &logger)
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
		logger.Warn().Interface("error", apierr).Msgf("API error: %v", apierr)
		w.Header().Add("Content-Type", ContentTypeJsonApi)
		render.Status(r, apierr.NumStatus)
		render.Render(w, r, errorResponses(*apierr))
		return
	}

	if state != "" {
		w.Header().Add("ETag", state)
	}
	if response == nil {
		render.Status(r, http.StatusNotFound)
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
