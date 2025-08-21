package groupware

import (
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
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
	jmapClient *jmap.Client
	errorTtl   time.Duration
}

func (l *sessionCacheLoader) Load(c *ttlcache.Cache[string, cachedSession], username string) *ttlcache.Item[string, cachedSession] {
	session, err := l.jmapClient.FetchSession(username, l.logger)
	if err != nil {
		l.logger.Warn().Str("username", username).Err(err).Msgf("failed to create session for '%v'", username)
		return c.Set(username, failedSession{err: err}, l.errorTtl)
	} else {
		l.logger.Debug().Str("username", username).Msgf("successfully created session for '%v'", username)
		return c.Set(username, succeededSession{session: session}, ttlcache.DefaultTTL) // use the TTL configured on the Cache
	}
}

var _ ttlcache.Loader[string, cachedSession] = &sessionCacheLoader{}
