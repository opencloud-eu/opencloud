package staticroutes_test

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	bcl "github.com/opencloud-eu/opencloud/services/proxy/pkg/staticroutes/backchannellogout"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v4/store"
	"golang.org/x/crypto/sha3"
)

func TestBackchannelLogoutDuringClaimsWrite(t *testing.T) {
	idp := newLogoutTestIDP(t)
	token := idp.accessToken(t, "alice", "session", "token", time.Hour)
	hash := make([]byte, 64)
	sha3.ShakeSum256(hash, []byte(token))
	key := base64.URLEncoding.EncodeToString(hash)
	started, release := make(chan struct{}), make(chan struct{})
	var unblock sync.Once
	t.Cleanup(func() { unblock.Do(func() { close(release) }) })
	cache := &logoutInterceptStore{Store: bcl.NewCache(store.NewMemoryStore(), true)}
	cache.write = func(record *store.Record) error {
		if record.Key == key {
			close(started)
			<-release
		}
		return nil
	}
	auth, routes := idp.handlers(cache, true)
	result := make(chan int, 1)
	go func() { result <- logoutTestRequest(auth, token) }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("claims write did not start")
	}
	require.Equal(t, http.StatusOK, logoutTestLogout(routes, idp.logoutToken(t, "alice", "session")).Code)
	unblock.Do(func() { close(release) })
	select {
	case status := <-result:
		require.Equal(t, http.StatusUnauthorized, status)
	case <-time.After(5 * time.Second):
		t.Fatal("authentication did not finish")
	}
	require.Equal(t, http.StatusUnauthorized, logoutTestRequest(auth, token), "the late claims write must not restore the token")
}

func TestBackchannelLogoutWithoutClaimsCacheOrTokenExpiry(t *testing.T) {
	for _, cacheClaims := range []bool{false, true} {
		t.Run(map[bool]string{false: "claims disabled", true: "claims enabled"}[cacheClaims], func(t *testing.T) {
			idp := newLogoutTestIDP(t)
			cache := bcl.NewCache(store.NewMemoryStore(), cacheClaims)
			auth, routes := idp.handlers(cache, true)
			token := idp.sign(t, jwt.MapClaims{"iss": idp.server.URL, "sub": "alice", "sid": "session", "aud": "opencloud"})
			require.Equal(t, http.StatusOK, logoutTestRequest(auth, token))
			require.Equal(t, http.StatusOK, logoutTestLogout(routes, idp.logoutToken(t, "alice", "session")).Code)
			require.Equal(t, http.StatusUnauthorized, logoutTestRequest(auth, token))
		})
	}
}

func TestBackchannelLogoutStorageFailures(t *testing.T) {
	idp := newLogoutTestIDP(t)
	token := idp.accessToken(t, "alice", "session", "token", time.Hour)
	logout := idp.logoutToken(t, "alice", "session")
	failure := errors.New("store unavailable")

	t.Run("cannot register session", func(t *testing.T) {
		cache := &logoutInterceptStore{Store: bcl.NewCache(store.NewMemoryStore(), true)}
		cache.write = func(record *store.Record) error {
			if strings.Contains(record.Key, ".") {
				return failure
			}
			return nil
		}
		auth, _ := idp.handlers(cache, true)
		require.Equal(t, http.StatusUnauthorized, logoutTestRequest(auth, token))
	})
	t.Run("cannot check revocation", func(t *testing.T) {
		cache := &logoutInterceptStore{Store: bcl.NewCache(store.NewMemoryStore(), true)}
		auth, _ := idp.handlers(cache, true)
		require.Equal(t, http.StatusOK, logoutTestRequest(auth, token))
		cache.read = func(key string) error {
			if strings.HasPrefix(key, "revoked/") {
				return failure
			}
			return nil
		}
		require.Equal(t, http.StatusUnauthorized, logoutTestRequest(auth, token))
	})
	for _, operation := range []string{"lookup", "revoke"} {
		t.Run(operation, func(t *testing.T) {
			cache := &logoutInterceptStore{Store: bcl.NewCache(store.NewMemoryStore(), true)}
			auth, routes := idp.handlers(cache, true)
			require.Equal(t, http.StatusOK, logoutTestRequest(auth, token))
			if operation == "lookup" {
				cache.read = func(key string) error {
					if strings.Contains(key, ".") {
						return failure
					}
					return nil
				}
			} else {
				cache.write = func(record *store.Record) error {
					if strings.HasPrefix(record.Key, "revoked/") {
						return failure
					}
					return nil
				}
			}
			require.Equal(t, http.StatusBadRequest, logoutTestLogout(routes, logout).Code, "a storage failure must not report a successful logout")
			cache.read, cache.write = nil, nil
			require.Equal(t, http.StatusOK, logoutTestLogout(routes, logout).Code, "the lookup must survive so the logout can be retried")
			require.Equal(t, http.StatusUnauthorized, logoutTestRequest(auth, token))
		})
	}
}

type logoutInterceptStore struct {
	store.Store
	read  func(string) error
	write func(*store.Record) error
}

func (s *logoutInterceptStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	if s.read != nil {
		if err := s.read(key); err != nil {
			return nil, err
		}
	}
	return s.Store.Read(key, opts...)
}

func (s *logoutInterceptStore) Write(record *store.Record, opts ...store.WriteOption) error {
	if s.write != nil {
		if err := s.write(record); err != nil {
			return err
		}
	}
	return s.Store.Write(record, opts...)
}
