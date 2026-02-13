// package backchannellogout provides functions to classify and lookup
// backchannel logout records from the cache store.

package backchannellogout

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	microstore "go-micro.dev/v4/store"
)

// SuSe 🦎 ;) is a struct that groups the subject and session together
// to prevent mix-ups for ('session, subject' || 'subject, session')
// return values.
type SuSe struct {
	Subject string
	Session string
}

// NewSuSe parses the subject and session id from the given key and returns a SuSe struct
func NewSuSe(key string) (SuSe, bool) {
	var subject, session string
	switch keys := strings.Split(strings.Join(strings.Fields(key), ""), "."); {
	case len(keys) == 2 && keys[0] == "" && keys[1] != "":
		session = keys[1]
	case len(keys) == 2 && keys[0] != "" && keys[1] == "":
		subject = keys[0]
	case len(keys) == 2 && keys[0] != "" && keys[1] != "":
		subject = keys[0]
		session = keys[1]
	case len(keys) == 1 && keys[0] != "":
		session = keys[0]
	default:
		return SuSe{}, false
	}

	return SuSe{Session: session, Subject: subject}, true
}

// LogoutMode defines the mode of backchannel logout, either by session or by subject
type LogoutMode int

const (
	// LogoutModeUnknown is used when the logout mode cannot be determined
	LogoutModeUnknown LogoutMode = iota
	// LogoutModeSession is used when the logout mode is determined by the session id
	LogoutModeSession
	// LogoutModeSubject is used when the logout mode is determined by the subject
	LogoutModeSubject
)

// GetLogoutMode determines the backchannel logout mode based on the presence of subject and session in the SuSe struct
func GetLogoutMode(suse SuSe) LogoutMode {
	switch {
	case suse.Session != "":
		return LogoutModeSession
	case suse.Subject != "":
		return LogoutModeSubject
	default:
		return LogoutModeUnknown
	}
}

// ErrSuspiciousCacheResult is returned when the cache result is suspicious
var ErrSuspiciousCacheResult = errors.New("suspicious cache result")

// GetLogoutRecords retrieves the records from the user info cache based on the backchannel
// logout mode and the provided SuSe struct.
// it uses a seperator to prevent sufix and prefix exploration in the cache and checks
// if the retrieved records match the requested subject and or session id as well, to prevent false positives.
func GetLogoutRecords(suse SuSe, mode LogoutMode, store microstore.Store) ([]*microstore.Record, error) {
	var key string
	var opts []microstore.ReadOption
	switch mode {
	case LogoutModeSession:
		// the dot at the beginning prevents sufix exploration in the cache,
		// so only keys that end with '*.session' will be returned, but not '*sion'.
		key = "." + suse.Session
		opts = append(opts, microstore.ReadSuffix())
	case LogoutModeSubject:
		// the dot at the end prevents prefix exploration in the cache,
		// so only keys that start with 'subject.*' will be returned, but not 'sub*'.
		key = suse.Subject + "."
		opts = append(opts, microstore.ReadPrefix())
	default:
		return nil, fmt.Errorf("%w: cannot determine logout mode", ErrSuspiciousCacheResult)
	}

	records, err := store.Read(key, append(opts, microstore.ReadLimit(1000))...)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, microstore.ErrNotFound
	}

	if mode == LogoutModeSession && len(records) > 1 {
		return nil, fmt.Errorf("%w: multiple session records found", ErrSuspiciousCacheResult)
	}

	// double-check if the found records match the requested subject and or session id as well,
	// to prevent false positives.
	for _, record := range records {
		recordSuSe, ok := NewSuSe(record.Key)
		if !ok {
			return nil, microstore.ErrNotFound
		}

		switch {
		// in session mode, the session id must match, but the subject can be different
		case mode == LogoutModeSession && suse.Session == recordSuSe.Session:
			continue
		// in subject mode, the subject must match, but the session id can be different
		case mode == LogoutModeSubject && suse.Subject == recordSuSe.Subject:
			continue
		}

		return nil, fmt.Errorf("%w: record key does not match the requested subject or session", ErrSuspiciousCacheResult)
	}

	return records, nil
}
