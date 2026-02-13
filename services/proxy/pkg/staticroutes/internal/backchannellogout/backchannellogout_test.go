package backchannellogout

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go-micro.dev/v4/store"
)

func TestNewSuSe(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		wantsuSe SuSe
		wantOk   bool
	}{
		{
			name:     ".session",
			key:      ".session",
			wantsuSe: SuSe{Session: "session", Subject: ""},
			wantOk:   true,
		},
		{
			name:     ".session",
			key:      ".session",
			wantsuSe: SuSe{Session: "session", Subject: ""},
			wantOk:   true,
		},
		{
			name:     "session",
			key:      "session",
			wantsuSe: SuSe{Session: "session", Subject: ""},
			wantOk:   true,
		},
		{
			name:     "subject.",
			key:      "subject.",
			wantsuSe: SuSe{Session: "", Subject: "subject"},
			wantOk:   true,
		},
		{
			name:     "subject.session",
			key:      "subject.session",
			wantsuSe: SuSe{Session: "session", Subject: "subject"},
			wantOk:   true,
		},
		{
			name:     "dot",
			key:      ".",
			wantsuSe: SuSe{Session: "", Subject: ""},
			wantOk:   false,
		},
		{
			name:     "empty",
			key:      "",
			wantsuSe: SuSe{Session: "", Subject: ""},
			wantOk:   false,
		},
		{
			name:     "whitespace . whitespace",
			key:      " . ",
			wantsuSe: SuSe{Session: "", Subject: ""},
			wantOk:   false,
		},
		{
			name:     "whitespace subject whitespace . whitespace",
			key:      " subject . ",
			wantsuSe: SuSe{Session: "", Subject: "subject"},
			wantOk:   true,
		},
		{
			name:     "whitespace . whitespace session whitespace",
			key:      " . session ",
			wantsuSe: SuSe{Session: "session", Subject: ""},
			wantOk:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suSe, ok := NewSuSe(tt.key)
			require.Equal(t, tt.wantOk, ok)
			require.Equal(t, tt.wantsuSe, suSe)
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
			name: ".session",
			suSe: SuSe{Session: "session", Subject: ""},
			want: LogoutModeSession,
		},
		{
			name: "subject.session",
			suSe: SuSe{Session: "session", Subject: "subject"},
			want: LogoutModeSession,
		},
		{
			name: "subject.",
			suSe: SuSe{Session: "", Subject: "subject"},
			want: LogoutModeSubject,
		},
		{
			name: "",
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
	recordSessionA := &store.Record{Key: "session-a", Value: []byte(recordClaimA.Key)}
	recordSessionB := &store.Record{Key: "session-b", Value: []byte(recordClaimB.Key)}
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
		store       store.Store
		wantRecords []*store.Record
		wantError   error
	}{
		{
			name:        "session-c",
			suSe:        SuSe{Session: "session-c"},
			mode:        LogoutModeSession,
			store:       sessionStore,
			wantRecords: []*store.Record{recordSubjectASessionC},
		},
		{
			name:        "ession-c",
			suSe:        SuSe{Session: "ession-c"},
			mode:        LogoutModeSession,
			store:       sessionStore,
			wantError:   store.ErrNotFound,
			wantRecords: []*store.Record{},
		},
		{
			name:        "subject-a",
			suSe:        SuSe{Subject: "subject-a"},
			mode:        LogoutModeSubject,
			store:       sessionStore,
			wantRecords: []*store.Record{recordSubjectASessionC, recordSubjectASessionD},
		},
		{
			name:        "subject-",
			suSe:        SuSe{Subject: "subject-"},
			mode:        LogoutModeSubject,
			store:       sessionStore,
			wantError:   store.ErrNotFound,
			wantRecords: []*store.Record{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, err := GetLogoutRecords(tt.suSe, tt.mode, tt.store)
			require.ErrorIs(t, err, tt.wantError)
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
