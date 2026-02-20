// package backchannellogout provides functions to classify and lookup
// backchannel logout records from the cache store.

package backchannellogout

import (
	"encoding/base64"
	"errors"
	"strings"

	microstore "go-micro.dev/v4/store"
)

// keyEncoding is the base64 encoding used for session and subject keys
var keyEncoding = base64.URLEncoding

// ErrInvalidKey indicates that the provided key does not conform to the expected format.
var ErrInvalidKey = errors.New("invalid key format")

// NewKey converts the subject and session to a base64 encoded key
func NewKey(subject, session string) (string, error) {
	subjectSession := strings.Join([]string{
		keyEncoding.EncodeToString([]byte(subject)),
		keyEncoding.EncodeToString([]byte(session)),
	}, ".")

	if subjectSession == "." {
		return "", ErrInvalidKey
	}

	return subjectSession, nil
}

// ErrDecoding is returned when decoding fails
var ErrDecoding = errors.New("failed to decode")

// SuSe 🦎 ;) is a struct that groups the subject and session together
// to prevent mix-ups for ('session, subject' || 'subject, session')
// return values.
type SuSe struct {
	encodedSubject string
	encodedSession string
}

// Subject decodes and returns the subject or an error
func (suse SuSe) Subject() (string, error) {
	subject, err := keyEncoding.DecodeString(suse.encodedSubject)
	if err != nil {
		return "", errors.Join(errors.New("failed to decode subject"), ErrDecoding, err)
	}

	return string(subject), nil
}

// Session decodes and returns the session or an error
func (suse SuSe) Session() (string, error) {
	subject, err := keyEncoding.DecodeString(suse.encodedSession)
	if err != nil {
		return "", errors.Join(errors.New("failed to decode session"), ErrDecoding, err)
	}

	return string(subject), nil
}

// ErrInvalidSubjectOrSession is returned when the provided key does not match the expected key format
var ErrInvalidSubjectOrSession = errors.New("invalid subject or session")

// NewSuSe parses the subject and session id from the given key and returns a SuSe struct
func NewSuSe(key string) (SuSe, error) {
	suse := SuSe{}
	switch keys := strings.Split(strings.Join(strings.Fields(key), ""), "."); {
	// key: '.session'
	case len(keys) == 2 && keys[0] == "" && keys[1] != "":
		suse.encodedSession = keys[1]
	// key: 'subject.'
	case len(keys) == 2 && keys[0] != "" && keys[1] == "":
		suse.encodedSubject = keys[0]
	// key: 'subject.session'
	case len(keys) == 2 && keys[0] != "" && keys[1] != "":
		suse.encodedSubject = keys[0]
		suse.encodedSession = keys[1]
	// key: 'session'
	case len(keys) == 1 && keys[0] != "":
		suse.encodedSession = keys[0]
	default:
		return suse, ErrInvalidSubjectOrSession
	}

	if _, err := suse.Subject(); err != nil {
		return suse, errors.Join(ErrInvalidSubjectOrSession, err)
	}

	if _, err := suse.Session(); err != nil {
		return suse, errors.Join(ErrInvalidSubjectOrSession, err)
	}

	return suse, nil
}

// LogoutMode defines the mode of backchannel logout, either by session or by subject
type LogoutMode int

const (
	// LogoutModeUndefined is used when the logout mode cannot be determined
	LogoutModeUndefined LogoutMode = iota
	// LogoutModeSubject is used when the logout mode is determined by the subject
	LogoutModeSubject
	// LogoutModeSession is used when the logout mode is determined by the session id
	LogoutModeSession
)

// GetLogoutMode determines the backchannel logout mode based on the presence of subject and session in the SuSe struct
func GetLogoutMode(suse SuSe) LogoutMode {
	switch {
	case suse.encodedSession == "" && suse.encodedSubject != "":
		return LogoutModeSubject
	case suse.encodedSession != "":
		return LogoutModeSession
	default:
		return LogoutModeUndefined
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
	case LogoutModeSubject:
		// the dot at the end prevents prefix exploration in the cache,
		// so only keys that start with 'subject.*' will be returned, but not 'sub*'.
		key = suse.encodedSubject + "."
		opts = append(opts, microstore.ReadPrefix())
	case LogoutModeSession:
		// the dot at the beginning prevents sufix exploration in the cache,
		// so only keys that end with '*.session' will be returned, but not '*sion'.
		key = "." + suse.encodedSession
		opts = append(opts, microstore.ReadSuffix())
	default:
		return nil, errors.Join(errors.New("cannot determine logout mode"), ErrSuspiciousCacheResult)
	}

	// the go micro memory store requires a limit to work, why???
	records, err := store.Read(key, append(opts, microstore.ReadLimit(1000))...)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, microstore.ErrNotFound
	}

	if mode == LogoutModeSession && len(records) > 1 {
		return nil, errors.Join(errors.New("multiple session records found"), ErrSuspiciousCacheResult)
	}

	// double-check if the found records match the requested subject and or session id as well,
	// to prevent false positives.
	for _, record := range records {
		recordSuSe, err := NewSuSe(record.Key)
		if err != nil {
			// never leak any key-related information
			return nil, errors.Join(errors.New("failed to parse key"), ErrSuspiciousCacheResult, err)
		}

		switch {
		// in subject mode, the subject must match, but the session id can be different
		case mode == LogoutModeSubject && suse.encodedSubject == recordSuSe.encodedSubject:
			continue
		// in session mode, the session id must match, but the subject can be different
		case mode == LogoutModeSession && suse.encodedSession == recordSuSe.encodedSession:
			continue
		}

		return nil, errors.Join(errors.New("key does not match the requested subject or session"), ErrSuspiciousCacheResult)
	}

	return records, nil
}
