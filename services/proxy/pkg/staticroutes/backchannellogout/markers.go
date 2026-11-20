package backchannellogout

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	microstore "go-micro.dev/v4/store"
)

// MarkerRetentionBuffer keeps markers a little longer than the longest accepted
// token lifetime, to cover late logout requests and identity provider nodes
// with slightly different clocks.
const MarkerRetentionBuffer = 5 * time.Minute

// ErrLoggedOut indicates that a token was issued before a backchannel logout that covers it.
var ErrLoggedOut = errors.New("access token was issued before a backchannel logout")

// ErrMarkersUnavailable indicates that a marker could not be read from the store,
// so it is unknown whether a logout covers the token.
var ErrMarkersUnavailable = errors.New("backchannel logout markers are unavailable")

// TokenRecord is the userinfo cache entry of a verified access token. It keeps
// the verified subject, session and issue time next to the claims, because a
// userinfo response can overwrite those values in the claims.
type TokenRecord struct {
	Claims    map[string]any
	Subject   string
	SessionID string
	// IssuedAt is the verified iat claim in unix seconds, or 0 if the token has none.
	IssuedAt int64
}

// LogoutMarkers records verified backchannel logouts and reports whether they cover a token.
//
// A marker holds a cutoff, which is the iat claim of the logout token. A token
// is covered if it belongs to the marked session or subject and was issued at
// or before the cutoff. Tokens issued later stay valid: the identity provider
// only issues them for sessions it still considers alive, for example a client
// with an offline token, or a new login that reuses the session id.
type LogoutMarkers struct {
	store     microstore.Store
	retention time.Duration
}

// NewLogoutMarkers returns markers that are kept in the given store for the given retention.
// The retention must be longer than the lifetime of any accepted access token.
func NewLogoutMarkers(store microstore.Store, retention time.Duration) *LogoutMarkers {
	return &LogoutMarkers{store: store, retention: retention}
}

// Record stores a marker for the session or, if the logout token has no session, for the subject.
// The cutoff of an existing marker never moves backwards, so replaying a logout token changes nothing.
//
// The store has no compare-and-swap. Two concurrent logouts for the same key can
// race, and the earlier cutoff can win until the next logout for that key.
func (m *LogoutMarkers) Record(subject, session string, cutoff time.Time) error {
	key, err := NewKey(subject, "")
	if session != "" {
		key, err = NewKey("", session)
	}
	if err != nil {
		return err
	}

	current, err := m.cutoff(key)
	if err != nil {
		return fmt.Errorf("failed to read logout marker: %w", err)
	}
	if current >= cutoff.Unix() {
		return nil
	}

	return m.store.Write(&microstore.Record{
		Key:    key,
		Value:  []byte(strconv.FormatInt(cutoff.Unix(), 10)),
		Expiry: m.retention,
	})
}

// Verify returns ErrLoggedOut if a marker covers a token with the given claims.
// It returns ErrMarkersUnavailable if the store cannot be read, and another error
// if a marker is corrupt. In both cases the token has not passed the check.
func (m *LogoutMarkers) Verify(subject, session string, issuedAt int64) error {
	var keys []string
	if session != "" {
		key, err := NewKey("", session)
		if err != nil {
			return err
		}
		keys = append(keys, key)
	}
	if subject != "" {
		key, err := NewKey(subject, "")
		if err != nil {
			return err
		}
		keys = append(keys, key)
	}

	for _, key := range keys {
		cutoff, err := m.cutoff(key)
		if err != nil {
			return fmt.Errorf("failed to read logout marker: %w", err)
		}
		// A token without an issue time cannot prove that it is newer than the marker.
		if cutoff != 0 && issuedAt <= cutoff {
			return ErrLoggedOut
		}
	}
	return nil
}

// cutoff returns the cutoff stored for the key, or 0 if there is no marker.
func (m *LogoutMarkers) cutoff(key string) (int64, error) {
	records, err := m.store.Read(key)
	switch {
	case errors.Is(err, microstore.ErrNotFound):
		return 0, nil
	case err != nil:
		return 0, fmt.Errorf("%w: %w", ErrMarkersUnavailable, err)
	case len(records) == 0:
		return 0, nil
	}

	cutoff, err := strconv.ParseInt(string(records[0].Value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("corrupt logout marker: %w", err)
	}
	return cutoff, nil
}
