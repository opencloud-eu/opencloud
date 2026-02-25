package backchannellogout

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v4/store"

	"github.com/opencloud-eu/opencloud/services/proxy/pkg/staticroutes/internal/backchannellogout/mocks"
)

func mustNewKey(t *testing.T, subject, session string) string {
	key, err := NewKey(subject, session)
	require.NoError(t, err)
	return key
}

func mustNewSuSe(t *testing.T, subject, session string) SuSe {
	suse, err := NewSuSe(mustNewKey(t, subject, session))
	require.NoError(t, err)
	return suse
}

func TestNewKey(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		session string
		wantKey string
		wantErr error
	}{
		{
			name:    "key variation: 'subject.session'",
			subject: "subject",
			session: "session",
			wantKey: "c3ViamVjdA==.c2Vzc2lvbg==",
		},
		{
			name:    "key variation: 'subject.'",
			subject: "subject",
			wantKey: "c3ViamVjdA==.",
		},
		{
			name:    "key variation: '.session'",
			session: "session",
			wantKey: ".c2Vzc2lvbg==",
		},
		{
			name:    "key variation: '.'",
			wantErr: ErrInvalidKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := NewKey(tt.subject, tt.session)
			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.wantKey, key)
		})
	}
}

func TestNewSuSe(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		wantSubject string
		wantSession string
		wantErr     error
	}{
		{
			name:        "key variation: '.session'",
			key:         mustNewKey(t, "", "session"),
			wantSession: "session",
		},
		{
			name:        "key variation: 'session'",
			key:         mustNewKey(t, "", "session"),
			wantSession: "session",
		},
		{
			name:        "key variation: 'subject.'",
			key:         mustNewKey(t, "subject", ""),
			wantSubject: "subject",
		},
		{
			name:        "key variation: 'subject.session'",
			key:         mustNewKey(t, "subject", "session"),
			wantSubject: "subject",
			wantSession: "session",
		},
		{
			name:    "key variation: 'dot'",
			key:     ".",
			wantErr: ErrInvalidSubjectOrSession,
		},
		{
			name:    "key variation: 'empty'",
			key:     "",
			wantErr: ErrInvalidSubjectOrSession,
		},
		{
			name:    "key variation: string('subject.session')",
			key:     "subject.session",
			wantErr: ErrInvalidSubjectOrSession,
		},
		{
			name:    "key variation: string('subject.')",
			key:     "subject.",
			wantErr: ErrInvalidSubjectOrSession,
		},
		{
			name:    "key variation: string('.session')",
			key:     ".session",
			wantErr: ErrInvalidSubjectOrSession,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suSe, err := NewSuSe(tt.key)
			require.ErrorIs(t, err, tt.wantErr)

			subject, _ := suSe.Subject()
			require.Equal(t, tt.wantSubject, subject)

			session, _ := suSe.Session()
			require.Equal(t, tt.wantSession, session)
		})
	}
}

func TestGetLogoutMode(t *testing.T) {
	tests := []struct {
		name string
		suSe SuSe
		want logoutMode
	}{
		{
			name: "key variation: '.session'",
			suSe: mustNewSuSe(t, "", "session"),
			want: logoutModeSession,
		},
		{
			name: "key variation: 'subject.session'",
			suSe: mustNewSuSe(t, "subject", "session"),
			want: logoutModeSession,
		},
		{
			name: "key variation: 'subject.'",
			suSe: mustNewSuSe(t, "subject", ""),
			want: logoutModeSubject,
		},
		{
			name: "key variation: 'empty'",
			suSe: SuSe{},
			want: logoutModeUndefined,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := getLogoutMode(tt.suSe)
			require.Equal(t, tt.want, mode)
		})
	}
}

func TestGetLogoutRecords(t *testing.T) {
	sessionStore := store.NewMemoryStore()

	recordClaimA := &store.Record{Key: "claim-a", Value: []byte("claim-a-data")}
	recordClaimB := &store.Record{Key: "claim-b", Value: []byte("claim-b-data")}
	recordClaimC := &store.Record{Key: "claim-c", Value: []byte("claim-c-data")}
	recordClaimD := &store.Record{Key: "claim-d", Value: []byte("claim-d-data")}
	recordSessionA := &store.Record{Key: mustNewKey(t, "", "session-a"), Value: []byte(recordClaimA.Key)}
	recordSessionB := &store.Record{Key: mustNewKey(t, "", "session-b"), Value: []byte(recordClaimB.Key)}
	recordSubjectASessionC := &store.Record{Key: mustNewKey(t, "subject-a", "session-c"), Value: []byte(recordSessionA.Key)}
	recordSubjectASessionD := &store.Record{Key: mustNewKey(t, "subject-a", "session-d"), Value: []byte(recordSessionA.Key)}

	for _, r := range []*store.Record{
		recordClaimA,
		recordClaimB,
		recordClaimC,
		recordClaimD,
		recordSessionA,
		recordSessionB,
		recordSubjectASessionC,
		recordSubjectASessionD,
	} {
		require.NoError(t, sessionStore.Write(r))
	}

	tests := []struct {
		name        string
		suSe        SuSe
		store       func(t *testing.T) store.Store
		wantRecords []*store.Record
		wantErrs    []error
	}{
		{
			name: "fails if multiple session records are found",
			suSe: mustNewSuSe(t, "", "session-a"),
			store: func(t *testing.T) store.Store {
				s := mocks.NewStore(t)
				s.EXPECT().Read(mock.Anything, mock.Anything).Return([]*store.Record{
					recordSessionA,
					recordSessionB,
				}, nil)
				return s
			},
			wantRecords: []*store.Record{},
			wantErrs:    []error{ErrSuspiciousCacheResult}},
		{
			name: "fails if the record key is not ok",
			suSe: mustNewSuSe(t, "", "session-a"),
			store: func(t *testing.T) store.Store {
				s := mocks.NewStore(t)
				s.EXPECT().Read(mock.Anything, mock.Anything).Return([]*store.Record{
					{Key: "invalid.record.key"},
				}, nil)
				return s
			},
			wantRecords: []*store.Record{},
			wantErrs:    []error{ErrInvalidSubjectOrSession, ErrSuspiciousCacheResult},
		},
		{
			name: "fails if the session does not match the retrieved record",
			suSe: mustNewSuSe(t, "", "session-a"),
			store: func(t *testing.T) store.Store {
				s := mocks.NewStore(t)
				s.EXPECT().Read(mock.Anything, mock.Anything).Return([]*store.Record{
					recordSessionB,
				}, nil)
				return s
			},
			wantRecords: []*store.Record{},
			wantErrs:    []error{ErrSuspiciousCacheResult}},
		{
			name: "fails if the subject does not match the retrieved record",
			suSe: mustNewSuSe(t, "subject-a", ""),
			store: func(t *testing.T) store.Store {
				s := mocks.NewStore(t)
				s.EXPECT().Read(mock.Anything, mock.Anything).Return([]*store.Record{
					recordSessionB,
				}, nil)
				return s
			},
			wantRecords: []*store.Record{},
			wantErrs:    []error{ErrSuspiciousCacheResult}},
		// key variation tests
		{
			name: "key variation: 'session-a'",
			suSe: mustNewSuSe(t, "", "session-a"),
			store: func(*testing.T) store.Store {
				return sessionStore
			},
			wantRecords: []*store.Record{recordSessionA},
		},
		{
			name: "key variation: 'session-b'",
			suSe: mustNewSuSe(t, "", "session-b"),
			store: func(*testing.T) store.Store {
				return sessionStore
			},
			wantRecords: []*store.Record{recordSessionB},
		},
		{
			name: "key variation: 'session-c'",
			suSe: mustNewSuSe(t, "", "session-c"),
			store: func(*testing.T) store.Store {
				return sessionStore
			},
			wantRecords: []*store.Record{recordSubjectASessionC},
		},
		{
			name: "key variation: 'ession-c'",
			suSe: mustNewSuSe(t, "", "ession-c"),
			store: func(*testing.T) store.Store {
				return sessionStore
			},
			wantRecords: []*store.Record{},
			wantErrs:    []error{store.ErrNotFound},
		},
		{
			name: "key variation: 'subject-a'",
			suSe: mustNewSuSe(t, "subject-a", ""),
			store: func(*testing.T) store.Store {
				return sessionStore
			},
			wantRecords: []*store.Record{recordSubjectASessionC, recordSubjectASessionD},
		},
		{
			name: "key variation: 'subject-'",
			suSe: mustNewSuSe(t, "subject-", ""),
			store: func(*testing.T) store.Store {
				return sessionStore
			},
			wantRecords: []*store.Record{},
			wantErrs:    []error{store.ErrNotFound},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, err := GetLogoutRecords(tt.suSe, tt.store(t))
			for _, wantErr := range tt.wantErrs {
				require.ErrorIs(t, err, wantErr)
			}
			require.Len(t, records, len(tt.wantRecords))

			sortRecords := func(r []*store.Record) []*store.Record {
				slices.SortFunc(r, func(a, b *store.Record) int {
					return strings.Compare(a.Key, b.Key)
				})

				return r
			}

			records = sortRecords(records)
			for i, wantRecords := range sortRecords(tt.wantRecords) {
				require.True(t, len(records) >= i+1)
				require.Equal(t, wantRecords.Key, records[i].Key)
				require.Equal(t, wantRecords.Value, records[i].Value)
			}
		})
	}
}
