package groupware

import (
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/metrics"
)

type sessionKey string

func toSessionKey(username string) sessionKey {
	return sessionKey(username)
}

func usernameFromSessionKey(key sessionKey) string {
	return string(key)
}

type cachedSession interface {
	Success() bool
	Get() jmap.Session
	Error() *GroupwareError
	Since() time.Time
}

type succeededSession struct {
	since   time.Time
	session jmap.Session
}

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

var _ cachedSession = succeededSession{}

type failedSession struct {
	since time.Time
	err   *GroupwareError
}

func (s failedSession) Success() bool {
	return false
}
func (s failedSession) Get() jmap.Session {
	panic("this should never be called")
}
func (s failedSession) Error() *GroupwareError {
	return s.err
}
func (s failedSession) Since() time.Time {
	return s.since
}

var _ cachedSession = failedSession{}

type sessionCacheLoader struct {
	logger     *log.Logger
	jmapClient *jmap.Client
	errorTtl   time.Duration
}

func (l *sessionCacheLoader) Load(c *ttlcache.Cache[sessionKey, cachedSession], key sessionKey) *ttlcache.Item[sessionKey, cachedSession] {
	username := usernameFromSessionKey(key)
	session, err := l.jmapClient.FetchSession(username, l.logger)
	if err != nil {
		l.logger.Warn().Str("username", username).Err(err).Msgf("failed to create session for '%v'", key)
		return c.Set(key, failedSession{since: time.Now(), err: groupwareErrorFromJmap(err)}, l.errorTtl)
	} else {
		l.logger.Debug().Str("username", username).Msgf("successfully created session for '%v'", key)
		return c.Set(key, succeededSession{since: time.Now(), session: session}, ttlcache.DefaultTTL) // use the TTL configured on the Cache
	}
}

var _ ttlcache.Loader[sessionKey, cachedSession] = &sessionCacheLoader{}

// Listens to JMAP Session outdated events, in order to remove outdated Sessions
// from the Groupware Session cache.
type sessionEventListener struct {
	logger       *log.Logger
	sessionCache *ttlcache.Cache[sessionKey, cachedSession]
	counter      prometheus.Counter
}

func (l sessionEventListener) OnSessionOutdated(session *jmap.Session, newSessionState jmap.SessionState) {
	// it's enough to remove the session from the cache, as it will be fetched on-demand
	// the next time an operation is performed on behalf of the user
	l.sessionCache.Delete(toSessionKey(session.Username))
	if l.counter != nil {
		l.counter.Inc()
	}

	l.logger.Trace().Msgf("removed outdated session for user '%v': state %v -> %v", session.Username, session.State, newSessionState)
}

var _ jmap.SessionEventListener = sessionEventListener{}

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
