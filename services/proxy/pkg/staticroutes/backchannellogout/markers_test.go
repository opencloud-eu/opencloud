package backchannellogout

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go-micro.dev/v4/store"
)

func TestLogoutMarkersVerify(t *testing.T) {
	const cutoff = int64(1000)

	tests := []struct {
		name          string
		logoutSubject string
		logoutSession string
		subject       string
		session       string
		issuedAt      int64
		wantLoggedOut bool
	}{
		{name: "session: issued before the cutoff", logoutSession: "s1", session: "s1", issuedAt: cutoff - 1, wantLoggedOut: true},
		{name: "session: issued in the same second as the cutoff", logoutSession: "s1", session: "s1", issuedAt: cutoff, wantLoggedOut: true},
		{name: "session: issued after the cutoff", logoutSession: "s1", session: "s1", issuedAt: cutoff + 1},
		{name: "session: other session", logoutSession: "s1", session: "s2", issuedAt: cutoff - 1},
		{name: "session: token without iat", logoutSession: "s1", session: "s1", issuedAt: 0, wantLoggedOut: true},
		{name: "session: token without session", logoutSession: "s1", subject: "alice", issuedAt: cutoff - 1},
		{name: "session and subject in the logout token only mark the session", logoutSubject: "alice", logoutSession: "s1", subject: "alice", session: "s2", issuedAt: cutoff - 1},
		{name: "subject: issued before the cutoff", logoutSubject: "alice", subject: "alice", session: "s1", issuedAt: cutoff - 1, wantLoggedOut: true},
		{name: "subject: issued in the same second as the cutoff", logoutSubject: "alice", subject: "alice", session: "s1", issuedAt: cutoff, wantLoggedOut: true},
		{name: "subject: issued after the cutoff", logoutSubject: "alice", subject: "alice", session: "s1", issuedAt: cutoff + 1},
		{name: "subject: other subject", logoutSubject: "alice", subject: "bob", issuedAt: cutoff - 1},
		{name: "subject: token without iat", logoutSubject: "alice", subject: "alice", issuedAt: 0, wantLoggedOut: true},
		{name: "no marker for a token without subject and session", logoutSubject: "alice", issuedAt: cutoff - 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			markers := NewLogoutMarkers(store.NewMemoryStore(), time.Hour)
			require.NoError(t, markers.Record(tt.logoutSubject, tt.logoutSession, time.Unix(cutoff, 0)))

			err := markers.Verify(tt.subject, tt.session, tt.issuedAt)
			if tt.wantLoggedOut {
				require.ErrorIs(t, err, ErrLoggedOut)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestLogoutMarkersVerifyWithoutMarker(t *testing.T) {
	markers := NewLogoutMarkers(store.NewMemoryStore(), time.Hour)
	require.NoError(t, markers.Verify("alice", "s1", 1000))
	require.NoError(t, markers.Verify("alice", "s1", 0))
}

func TestLogoutMarkersRecordRejectsMissingSubjectAndSession(t *testing.T) {
	markers := NewLogoutMarkers(store.NewMemoryStore(), time.Hour)
	require.ErrorIs(t, markers.Record("", "", time.Unix(1000, 0)), ErrInvalidKey)
}

func TestLogoutMarkersCutoffNeverMovesBackwards(t *testing.T) {
	markers := NewLogoutMarkers(store.NewMemoryStore(), time.Hour)
	require.NoError(t, markers.Record("", "s1", time.Unix(2000, 0)))

	// A replayed or delayed logout token carries an older iat.
	require.NoError(t, markers.Record("", "s1", time.Unix(1000, 0)))
	require.ErrorIs(t, markers.Verify("", "s1", 1500), ErrLoggedOut)

	require.NoError(t, markers.Record("", "s1", time.Unix(3000, 0)))
	require.ErrorIs(t, markers.Verify("", "s1", 2500), ErrLoggedOut)
	require.NoError(t, markers.Verify("", "s1", 3001))
}

func TestLogoutMarkersExpire(t *testing.T) {
	markers := NewLogoutMarkers(store.NewMemoryStore(), 50*time.Millisecond)
	require.NoError(t, markers.Record("", "s1", time.Unix(1000, 0)))
	require.ErrorIs(t, markers.Verify("", "s1", 1000), ErrLoggedOut)

	require.Eventually(t, func() bool {
		return markers.Verify("", "s1", 1000) == nil
	}, time.Second, 10*time.Millisecond)
}

// failingStore fails every read and write after the configured error is set.
type failingStore struct {
	store.Store
	readErr  error
	writeErr error
}

func (s *failingStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	if s.readErr != nil {
		return nil, s.readErr
	}
	return s.Store.Read(key, opts...)
}

func (s *failingStore) Write(record *store.Record, opts ...store.WriteOption) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	return s.Store.Write(record, opts...)
}

func TestLogoutMarkersReportStoreFailures(t *testing.T) {
	failure := errors.New("store unavailable")

	t.Run("verify fails when a marker cannot be read", func(t *testing.T) {
		markers := NewLogoutMarkers(&failingStore{Store: store.NewMemoryStore(), readErr: failure}, time.Hour)
		err := markers.Verify("alice", "s1", 1000)
		require.ErrorIs(t, err, failure)
		require.NotErrorIs(t, err, ErrLoggedOut)
	})

	t.Run("record fails when the current marker cannot be read", func(t *testing.T) {
		markers := NewLogoutMarkers(&failingStore{Store: store.NewMemoryStore(), readErr: failure}, time.Hour)
		require.ErrorIs(t, markers.Record("", "s1", time.Unix(1000, 0)), failure)
	})

	t.Run("record fails when the marker cannot be written", func(t *testing.T) {
		markers := NewLogoutMarkers(&failingStore{Store: store.NewMemoryStore(), writeErr: failure}, time.Hour)
		require.ErrorIs(t, markers.Record("", "s1", time.Unix(1000, 0)), failure)
	})

	t.Run("verify fails on a corrupt marker", func(t *testing.T) {
		backing := store.NewMemoryStore()
		key, err := NewKey("", "s1")
		require.NoError(t, err)
		require.NoError(t, backing.Write(&store.Record{Key: key, Value: []byte("not-a-number")}))
		require.Error(t, NewLogoutMarkers(backing, time.Hour).Verify("", "s1", 1000))
	})
}
