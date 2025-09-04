package groupware

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/metrics"
)

// An alias for the internal session cache key, which might become something composed in the future.
type sessionCacheKey string

func toSessionCacheKey(username string) sessionCacheKey {
	return sessionCacheKey(username)
}

func (k sessionCacheKey) username() string {
	return string(k)
}

// Interface for cached sessions in the session cache.
// The purpose here is mainly to be able to also persist failed
// attempts to retrieve a session.
type cachedSession interface {
	// Whether the Session retrieval was successful or not.
	Success() bool
	// When Success() returns true, one may use this method to retrieve the actual JMAP Session.
	Get() jmap.Session
	// When Success() returns false, one may use this method to retrieve the error that caused the failure.
	Error() *GroupwareError
	// The timestamp of when this cached session information was obtained, regardless of success or failure.
	Since() time.Time
}

// An implementation of a cachedSession that succeeded.
type succeededSession struct {
	// Timestamp of when this succeededSession was created.
	since time.Time
	// The JMAP Session itself.
	session jmap.Session
}

var _ cachedSession = succeededSession{}

func (s succeededSession) Success() bool {
	return true
}
func (s succeededSession) Get() jmap.Session {
	return s.session
}
func (s succeededSession) Error() *GroupwareError {
	return nil
}
func (s succeededSession) Since() time.Time {
	return s.since
}

// An implementation of a cachedSession that failed.
type failedSession struct {
	// Timestamp of when this failedSession was created.
	since time.Time
	// The error that caused the Session acquisition to fail.
	err *GroupwareError
}

var _ cachedSession = failedSession{}

func (s failedSession) Success() bool {
	return false
}
func (s failedSession) Get() jmap.Session {
	panic(fmt.Sprintf("never call %T.Get()", failedSession{}))
}
func (s failedSession) Error() *GroupwareError {
	return s.err
}
func (s failedSession) Since() time.Time {
	return s.since
}

// Implements the ttlcache.Loader interface, by loading JMAP Sessions for users
// using the jmap.Client.
type sessionCacheLoader struct {
	logger *log.Logger
	// A minimalistic contract for supplying the JMAP Session URL for a given username.
	sessionUrlProvider func(username string) (*url.URL, *GroupwareError)
	// A minimalistic contract for supplying JMAP Sessions using various input parameters.
	sessionSupplier func(sessionUrl *url.URL, username string, logger *log.Logger) (jmap.Session, jmap.Error)
	errorTtl        time.Duration
}

var _ ttlcache.Loader[sessionCacheKey, cachedSession] = &sessionCacheLoader{}

func (l *sessionCacheLoader) Load(c *ttlcache.Cache[sessionCacheKey, cachedSession], key sessionCacheKey) *ttlcache.Item[sessionCacheKey, cachedSession] {
	username := key.username()
	sessionUrl, gwerr := l.sessionUrlProvider(username)
	if gwerr != nil {
		l.logger.Warn().Str("username", username).Str("code", gwerr.Code).Msgf("failed to determine session URL for '%v'", key)
		return c.Set(key, failedSession{since: time.Now(), err: gwerr}, l.errorTtl)
	}
	session, jerr := l.sessionSupplier(sessionUrl, username, l.logger)
	if jerr != nil {
		l.logger.Warn().Str("username", username).Err(jerr).Msgf("failed to create session for '%v'", key)
		return c.Set(key, failedSession{since: time.Now(), err: groupwareErrorFromJmap(jerr)}, l.errorTtl)
	} else {
		l.logger.Debug().Str("username", username).Msgf("successfully created session for '%v'", key)
		return c.Set(key, succeededSession{since: time.Now(), session: session}, ttlcache.DefaultTTL) // use the TTL configured on the Cache
	}
}

type sessionCache interface {
	Get(username string) cachedSession
	jmap.SessionEventListener
}

type ttlcacheSessionCache struct {
	sessionCache           *ttlcache.Cache[sessionCacheKey, cachedSession]
	outdatedSessionCounter prometheus.Counter
	logger                 *log.Logger
}

var _ sessionCache = &ttlcacheSessionCache{}
var _ jmap.SessionEventListener = &ttlcacheSessionCache{}

func (c *ttlcacheSessionCache) Get(username string) cachedSession {
	item := c.sessionCache.Get(toSessionCacheKey(username))
	if item != nil {
		return item.Value()
	} else {
		return nil
	}
}

type sessionCacheBuilder struct {
	logger                    *log.Logger
	sessionSupplier           func(sessionUrl *url.URL, username string, logger *log.Logger) (jmap.Session, jmap.Error)
	defaultUrlResolver        func(string) (*url.URL, *GroupwareError)
	sessionUrlResolverFactory func() (func(string) (*url.URL, *GroupwareError), *GroupwareInitializationError)
	prometheusRegistry        prometheus.Registerer
	m                         *metrics.Metrics
	sessionCacheMaxCapacity   uint64
	sessionCacheTtl           time.Duration
	sessionFailureCacheTtl    time.Duration
}

func newSessionCacheBuilder(
	sessionUrl *url.URL,
	logger *log.Logger,
	sessionSupplier func(sessionUrl *url.URL, username string, logger *log.Logger) (jmap.Session, jmap.Error),
	prometheusRegistry prometheus.Registerer,
	m *metrics.Metrics,
	sessionCacheMaxCapacity uint64,
	sessionCacheTtl time.Duration,
	sessionFailureCacheTtl time.Duration,
) *sessionCacheBuilder {
	defaultUrlResolver := func(_ string) (*url.URL, *GroupwareError) {
		return sessionUrl, nil
	}

	return &sessionCacheBuilder{
		logger:             logger,
		sessionSupplier:    sessionSupplier,
		defaultUrlResolver: defaultUrlResolver,
		sessionUrlResolverFactory: func() (func(string) (*url.URL, *GroupwareError), *GroupwareInitializationError) {
			return defaultUrlResolver, nil
		},
		prometheusRegistry:      prometheusRegistry,
		m:                       m,
		sessionCacheMaxCapacity: sessionCacheMaxCapacity,
		sessionCacheTtl:         sessionCacheTtl,
		sessionFailureCacheTtl:  sessionFailureCacheTtl,
	}
}

func (b *sessionCacheBuilder) withDnsAutoDiscovery(
	defaultSessionDomain string,
	config *dns.ClientConfig,
	dnsDialTimeout time.Duration,
	dnsReadTimeout time.Duration,
	domainGreenList []string,
	domainRedList []string,
) *sessionCacheBuilder {
	dnsSessionUrlResolverFactory := func() (func(string) (*url.URL, *GroupwareError), *GroupwareInitializationError) {
		d, err := NewDnsSessionUrlResolver(
			b.defaultUrlResolver,
			defaultSessionDomain,
			config,
			domainGreenList,
			domainRedList,
			dnsDialTimeout,
			dnsReadTimeout,
		)
		if err != nil {
			return nil, &GroupwareInitializationError{Message: "failed to instantiate the DNS session URL resolver", Err: err}
		} else {
			return d.Resolve, nil
		}
	}
	b.sessionUrlResolverFactory = dnsSessionUrlResolverFactory
	return b
}

func (b sessionCacheBuilder) build() (sessionCache, error) {
	var cache *ttlcache.Cache[sessionCacheKey, cachedSession]

	sessionUrlResolver, err := b.sessionUrlResolverFactory()
	if err != nil {
		return nil, err
	}

	sessionLoader := &sessionCacheLoader{
		logger:             b.logger,
		sessionSupplier:    b.sessionSupplier,
		errorTtl:           b.sessionFailureCacheTtl,
		sessionUrlProvider: sessionUrlResolver,
	}

	cache = ttlcache.New(
		ttlcache.WithCapacity[sessionCacheKey, cachedSession](b.sessionCacheMaxCapacity),
		ttlcache.WithTTL[sessionCacheKey, cachedSession](b.sessionCacheTtl),
		ttlcache.WithDisableTouchOnHit[sessionCacheKey, cachedSession](),
		ttlcache.WithLoader(sessionLoader),
	)

	b.prometheusRegistry.Register(sessionCacheMetricsCollector{desc: b.m.SessionCacheDesc, supply: cache.Metrics})

	cache.OnEviction(func(c context.Context, r ttlcache.EvictionReason, item *ttlcache.Item[sessionCacheKey, cachedSession]) {
		if b.logger.Trace().Enabled() {
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
			b.logger.Trace().Msgf("%s session cache eviction of user '%v' after %v: %v", tipe, item.Key(), spentInCache, reason)
		}
	})

	s := &ttlcacheSessionCache{
		sessionCache:           cache,
		logger:                 b.logger,
		outdatedSessionCounter: b.m.OutdatedSessionsCounter,
	}

	go cache.Start()

	return s, nil
}

func (c ttlcacheSessionCache) OnSessionOutdated(session *jmap.Session, newSessionState jmap.SessionState) {
	// it's enough to remove the session from the cache, as it will be fetched on-demand
	// the next time an operation is performed on behalf of the user
	c.sessionCache.Delete(toSessionCacheKey(session.Username))
	if c.outdatedSessionCounter != nil {
		c.outdatedSessionCounter.Inc()
	}

	c.logger.Trace().Msgf("removed outdated session for user '%v': state %v -> %v", session.Username, session.State, newSessionState)
}

// A Prometheus Collector for the Session cache metrics.
type sessionCacheMetricsCollector struct {
	desc   *prometheus.Desc
	supply func() ttlcache.Metrics
}

func (s sessionCacheMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- s.desc
}
func (s sessionCacheMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	m := s.supply()
	ch <- prometheus.MustNewConstMetric(s.desc, prometheus.GaugeValue, float64(m.Evictions), metrics.Values.SessionCache.Evictions)
	ch <- prometheus.MustNewConstMetric(s.desc, prometheus.GaugeValue, float64(m.Insertions), metrics.Values.SessionCache.Insertions)
	ch <- prometheus.MustNewConstMetric(s.desc, prometheus.GaugeValue, float64(m.Hits), metrics.Values.SessionCache.Hits)
	ch <- prometheus.MustNewConstMetric(s.desc, prometheus.GaugeValue, float64(m.Misses), metrics.Values.SessionCache.Misses)
}

var _ prometheus.Collector = sessionCacheMetricsCollector{}

// Create a new log.Logger that is decorated with fields containing information about the Session.
func decorateLogger(l *log.Logger, session jmap.Session) *log.Logger {
	return log.From(l.With().
		Str(logUsername, log.SafeString(session.Username)).
		Str(logSessionState, log.SafeString(string(session.State))))
}
