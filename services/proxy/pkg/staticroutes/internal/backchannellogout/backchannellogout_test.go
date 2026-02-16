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

func TestNewSuSe(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		wantSuSe SuSe
		wantErr  error
	}{
		{
			name:     "key variation: '.session'",
			key:      ".session",
			wantSuSe: SuSe{Session: "session", Subject: ""},
		},
		{
			name:     "key variation: '.session'",
			key:      ".session",
			wantSuSe: SuSe{Session: "session", Subject: ""},
		},
		{
			name:     "key variation: 'session'",
			key:      "session",
			wantSuSe: SuSe{Session: "session", Subject: ""},
		},
		{
			name:     "key variation: 'subject.'",
			key:      "subject.",
			wantSuSe: SuSe{Session: "", Subject: "subject"},
		},
		{
			name:     "key variation: 'subject.session'",
			key:      "subject.session",
			wantSuSe: SuSe{Session: "session", Subject: "subject"},
		},
		{
			name:     "key variation: 'dot'",
			key:      ".",
			wantSuSe: SuSe{Session: "", Subject: ""},
			wantErr:  ErrInvalidSessionOrSubject,
		},
		{
			name:     "key variation: 'empty'",
			key:      "",
			wantSuSe: SuSe{Session: "", Subject: ""},
			wantErr:  ErrInvalidSessionOrSubject,
		},
		{
			name:     "key variation: 'whitespace . whitespace'",
			key:      " . ",
			wantSuSe: SuSe{Session: "", Subject: ""},
			wantErr:  ErrInvalidSessionOrSubject,
		},
		{
			name:     "key variation: 'whitespace subject whitespace . whitespace'",
			key:      " subject . ",
			wantSuSe: SuSe{Session: "", Subject: "subject"},
		},
		{
			name:     "key variation: 'whitespace . whitespace session whitespace'",
			key:      " . session ",
			wantSuSe: SuSe{Session: "session", Subject: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suSe, ok := NewSuSe(tt.key)
			require.ErrorIs(t, tt.wantErr, ok)
			require.Equal(t, tt.wantSuSe, suSe)
		})
	}
}

func TestGetLogoutMode(t *testing.T) {
	tests := []struct {
		name string
		suSe SuSe
		want LogoutMode
	}{
		{
			name: "key variation: '.session'",
			suSe: SuSe{Session: "session", Subject: ""},
			want: LogoutModeSession,
		},
		{
			name: "key variation: 'subject.session'",
			suSe: SuSe{Session: "session", Subject: "subject"},
			want: LogoutModeSession,
		},
		{
			name: "key variation: 'subject.'",
			suSe: SuSe{Session: "", Subject: "subject"},
			want: LogoutModeSubject,
		},
		{
			name: "key variation: 'empty'",
			suSe: SuSe{Session: "", Subject: ""},
			want: LogoutModeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := GetLogoutMode(tt.suSe)
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
	recordSessionA := &store.Record{Key: ".session-a", Value: []byte(recordClaimA.Key)}
	recordSessionB := &store.Record{Key: ".session-b", Value: []byte(recordClaimB.Key)}
	recordSubjectASessionC := &store.Record{Key: "subject-a.session-c", Value: []byte(recordSessionA.Key)}
	recordSubjectASessionD := &store.Record{Key: "subject-a.session-d", Value: []byte(recordSessionB.Key)}

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
		mode        LogoutMode
		store       func(t *testing.T) store.Store
		wantRecords []*store.Record
		wantErrs    []error
	}{
		{
			name: "fails if mode is unknown",
			suSe: SuSe{Session: "session-a"},
			mode: LogoutModeUnknown,
			store: func(t *testing.T) store.Store {
				return sessionStore
			},
			wantRecords: []*store.Record{},
			wantErrs:    []error{ErrSuspiciousCacheResult},
		},
		{
			name: "fails if mode is any random int",
			suSe: SuSe{Session: "session-a"},
			mode: 999,
			store: func(t *testing.T) store.Store {
				return sessionStore
			},
			wantRecords: []*store.Record{},
			wantErrs:    []error{ErrSuspiciousCacheResult}},
		{
			name: "fails if multiple session records are found",
			suSe: SuSe{Session: "session-a"},
			mode: LogoutModeSession,
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
			suSe: SuSe{Session: "session-a"},
			mode: LogoutModeSession,
			store: func(t *testing.T) store.Store {
				s := mocks.NewStore(t)
				s.EXPECT().Read(mock.Anything, mock.Anything).Return([]*store.Record{
					&store.Record{Key: "invalid.record.key"},
				}, nil)
				return s
			},
			wantRecords: []*store.Record{},
			wantErrs:    []error{ErrInvalidSessionOrSubject, ErrSuspiciousCacheResult},
		},
		{
			name: "fails if the session does not match the retrieved record",
			suSe: SuSe{Session: "session-a"},
			mode: LogoutModeSession,
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
			suSe: SuSe{Subject: "subject-a"},
			mode: LogoutModeSubject,
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
			suSe: SuSe{Session: "session-a"},
			mode: LogoutModeSession,
			store: func(*testing.T) store.Store {
				return sessionStore
			},
			wantRecords: []*store.Record{recordSessionA},
		},
		{
			name: "key variation: 'session-b'",
			suSe: SuSe{Session: "session-b"},
			mode: LogoutModeSession,
			store: func(*testing.T) store.Store {
				return sessionStore
			},
			wantRecords: []*store.Record{recordSessionB},
		},
		{
			name: "key variation: 'session-c'",
			suSe: SuSe{Session: "session-c"},
			mode: LogoutModeSession,
			store: func(*testing.T) store.Store {
				return sessionStore
			},
			wantRecords: []*store.Record{recordSubjectASessionC},
		},
		{
			name: "key variation: 'ession-c'",
			suSe: SuSe{Session: "ession-c"},
			mode: LogoutModeSession,
			store: func(*testing.T) store.Store {
				return sessionStore
			},
			wantRecords: []*store.Record{},
			wantErrs:    []error{store.ErrNotFound},
		},
		{
			name: "key variation: 'subject-a'",
			suSe: SuSe{Subject: "subject-a"},
			mode: LogoutModeSubject,
			store: func(*testing.T) store.Store {
				return sessionStore
			},
			wantRecords: []*store.Record{recordSubjectASessionC, recordSubjectASessionD},
		},
		{
			name: "key variation: 'subject-'",
			suSe: SuSe{Subject: "subject-"},
			mode: LogoutModeSubject,
			store: func(*testing.T) store.Store {
				return sessionStore
			},
			wantRecords: []*store.Record{},
			wantErrs:    []error{store.ErrNotFound},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, err := GetLogoutRecords(tt.suSe, tt.mode, tt.store(t))
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
