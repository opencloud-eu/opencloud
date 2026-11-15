// Package mfa implements two-factor authentication with time-based one-time passwords
// (TOTP, RFC 6238) for the builtin identity provider. Users enable it themselves with an
// authenticator app. Once enabled a one-time code or a recovery code is required in
// addition to the password to log in.
package mfa

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	microstore "go-micro.dev/v4/store"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

const (
	// maxFailedAttempts is the number of consecutive wrong codes that lock the second factor.
	maxFailedAttempts = 5
	// lockDuration is how long no codes are accepted after too many wrong ones.
	lockDuration = 5 * time.Minute
	// pendingSetupTTL is how long a started setup can be confirmed.
	pendingSetupTTL = 10 * time.Minute
	// recoveryCodeCount is the number of recovery codes a user gets.
	recoveryCodeCount = 10
	// recoveryCodeLength is the number of characters of a recovery code (5 bit each).
	recoveryCodeLength = 10
	// recoveryCodeAlphabet has 32 characters, so a random byte maps to it without bias.
	recoveryCodeAlphabet = "abcdefghijklmnopqrstuvwxyz234567"
)

var (
	// ErrAlreadyEnabled is returned when a setup is started although two-factor authentication is enabled.
	ErrAlreadyEnabled = errors.New("mfa: two-factor authentication is already enabled")
	// ErrNotEnabled is returned when two-factor authentication is not enabled for the user.
	ErrNotEnabled = errors.New("mfa: two-factor authentication is not enabled")
	// ErrNoPendingSetup is returned when there is no setup to confirm or it expired.
	ErrNoPendingSetup = errors.New("mfa: no pending setup")
	// ErrInvalidCode is returned when the code is wrong.
	ErrInvalidCode = errors.New("mfa: invalid code")
	// ErrLocked is returned when too many wrong codes were entered.
	ErrLocked = errors.New("mfa: too many failed attempts")
	// ErrUserNotFound is returned when no user with two-factor authentication matches.
	ErrUserNotFound = errors.New("mfa: no such user with two-factor authentication")
)

// LoginResult is the outcome of the second factor check during a login.
type LoginResult string

const (
	// LoginNotEnrolled means the user has no second factor, the password is sufficient.
	LoginNotEnrolled LoginResult = ""
	// LoginOK means the code was correct.
	LoginOK LoginResult = "ok"
	// LoginCodeRequired means the user has a second factor but no code was sent.
	LoginCodeRequired LoginResult = "required"
	// LoginCodeInvalid means the code was wrong.
	LoginCodeInvalid LoginResult = "invalid"
	// LoginLocked means too many wrong codes were sent.
	LoginLocked LoginResult = "locked"
)

// Passed tells if the login may proceed.
func (r LoginResult) Passed() bool {
	return r == LoginNotEnrolled || r == LoginOK
}

// User identifies a user of the identity provider.
type User struct {
	// ID is the subject of the user as provided by the identifier backend. It has to be stable.
	ID string
	// Username is the login name. It is used as the account name in authenticator apps.
	Username string
}

// Status describes the second factor of a user.
type Status struct {
	Enabled           bool
	EnabledAt         time.Time
	RecoveryCodesLeft int
}

// Enrollment describes a user with two-factor authentication enabled.
type Enrollment struct {
	User      User
	EnabledAt time.Time
}

// Service manages the second factors of the users.
type Service struct {
	store  microstore.Store
	sealer *sealer
	issuer string
	logger log.Logger

	// mu serializes read-modify-write cycles on the store within this process.
	mu sync.Mutex

	// now can be replaced in tests.
	now func() time.Time

	// passwordVerifier is set when the identifier backend is wrapped, see WrapBackend.
	passwordVerifier passwordVerifier
}

// New creates a new Service. The encryption key for the TOTP secrets is derived from
// masterSecret. The issuer is the name shown in authenticator apps.
func New(store microstore.Store, masterSecret []byte, issuer string, logger log.Logger) (*Service, error) {
	if store == nil {
		return nil, errors.New("mfa: a store is required")
	}
	sealer, err := newSealer(masterSecret)
	if err != nil {
		return nil, err
	}
	if issuer == "" {
		issuer = "OpenCloud"
	}

	return &Service{
		store:  store,
		sealer: sealer,
		issuer: issuer,
		logger: logger,
		now:    time.Now,
	}, nil
}

// Status returns the state of the second factor of the user.
func (s *Service) Status(user User) (Status, error) {
	rec, err := readRecord(s.store, user.ID)
	if err != nil {
		return Status{}, err
	}
	if !rec.enabled() {
		return Status{}, nil
	}
	return Status{
		Enabled:           true,
		EnabledAt:         rec.EnabledAt,
		RecoveryCodesLeft: len(rec.RecoveryCodes),
	}, nil
}

// Setup starts the setup of an authenticator app. It returns the base32 encoded secret and
// the otpauth:// URI for the QR code. The setup has to be confirmed with Enable.
func (s *Service) Setup(user User) (secret string, uri string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, err := readRecord(s.store, user.ID)
	if err != nil {
		return "", "", err
	}
	if rec.enabled() {
		return "", "", ErrAlreadyEnabled
	}

	raw, err := generateSecret()
	if err != nil {
		return "", "", err
	}
	sealed, err := s.sealer.seal(raw, user.ID)
	if err != nil {
		return "", "", err
	}

	rec = &record{
		UserID:         user.ID,
		Username:       user.Username,
		PendingSecret:  sealed,
		PendingExpires: s.now().Add(pendingSetupTTL),
	}
	if err := writeRecord(s.store, rec); err != nil {
		return "", "", err
	}

	return encodeSecret(raw), provisioningURI(s.issuer, user.Username, raw), nil
}

// Enable confirms a setup with a one-time code from the authenticator app and enables
// two-factor authentication. It returns the recovery codes, they cannot be retrieved again.
func (s *Service) Enable(user User, code string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, err := readRecord(s.store, user.ID)
	if err != nil {
		return nil, err
	}
	if rec.enabled() {
		return nil, ErrAlreadyEnabled
	}
	now := s.now()
	if rec == nil || len(rec.PendingSecret) == 0 || now.After(rec.PendingExpires) {
		return nil, ErrNoPendingSetup
	}

	secret, err := s.sealer.open(rec.PendingSecret, user.ID)
	if err != nil {
		return nil, err
	}
	step, ok := verifyTOTP(secret, normalizeCode(code), now, 0)
	if !ok {
		return nil, ErrInvalidCode
	}

	codes, hashes, err := generateRecoveryCodes()
	if err != nil {
		return nil, err
	}

	rec = &record{
		UserID:        user.ID,
		Username:      user.Username,
		Secret:        rec.PendingSecret,
		EnabledAt:     now,
		RecoveryCodes: hashes,
		LastStep:      step,
	}
	if err := writeRecord(s.store, rec); err != nil {
		return nil, err
	}

	s.logger.Info().Str("userid", user.ID).Str("username", user.Username).Msg("two-factor authentication enabled")
	return codes, nil
}

// Disable turns two-factor authentication off for the user. A one-time code or a recovery
// code is required.
func (s *Service) Disable(user User, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, err := s.verifyEnrolled(user, code)
	if err != nil {
		return err
	}
	if err := deleteRecord(s.store, rec.UserID); err != nil {
		return err
	}

	s.logger.Info().Str("userid", user.ID).Str("username", user.Username).Msg("two-factor authentication disabled")
	return nil
}

// RegenerateRecoveryCodes replaces the recovery codes of the user. A one-time code or a
// recovery code is required.
func (s *Service) RegenerateRecoveryCodes(user User, code string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, err := s.verifyEnrolled(user, code)
	if err != nil {
		return nil, err
	}

	codes, hashes, err := generateRecoveryCodes()
	if err != nil {
		return nil, err
	}
	rec.RecoveryCodes = hashes
	if err := writeRecord(s.store, rec); err != nil {
		return nil, err
	}

	s.logger.Info().Str("userid", user.ID).Str("username", user.Username).Msg("two-factor recovery codes regenerated")
	return codes, nil
}

// VerifyLogin checks the second factor during a login. It must only be called after the
// password of the user was verified. An error means the check could not be done, the login
// has to be rejected then.
func (s *Service) VerifyLogin(user User, code string) (LoginResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, err := readRecord(s.store, user.ID)
	if err != nil {
		return LoginCodeInvalid, err
	}
	if !rec.enabled() {
		return LoginNotEnrolled, nil
	}
	if normalizeCode(code) == "" {
		return LoginCodeRequired, nil
	}

	ok, locked, err := s.verifyCode(rec, code)
	if err != nil {
		return LoginCodeInvalid, err
	}
	if rec.Username != user.Username && user.Username != "" {
		rec.Username = user.Username
	}
	// always persist: the counters, the replay protection or the used recovery code changed
	if err := writeRecord(s.store, rec); err != nil {
		return LoginCodeInvalid, err
	}

	switch {
	case locked:
		s.logger.Warn().Str("userid", user.ID).Str("username", user.Username).Msg("login rejected, second factor is locked after too many wrong codes")
		return LoginLocked, nil
	case !ok:
		return LoginCodeInvalid, nil
	}
	return LoginOK, nil
}

// List returns all users with two-factor authentication enabled.
func (s *Service) List() ([]Enrollment, error) {
	records, err := listRecords(s.store)
	if err != nil {
		return nil, err
	}

	enrollments := make([]Enrollment, 0, len(records))
	for _, rec := range records {
		if !rec.enabled() {
			continue
		}
		enrollments = append(enrollments, Enrollment{
			User:      User{ID: rec.UserID, Username: rec.Username},
			EnabledAt: rec.EnabledAt,
		})
	}
	sort.Slice(enrollments, func(i, j int) bool {
		return enrollments[i].User.Username < enrollments[j].User.Username
	})
	return enrollments, nil
}

// AdminDisable turns two-factor authentication off for a user without a code. It is meant
// for administrators helping users who lost their authenticator app and recovery codes.
// The user is identified by user id or username.
func (s *Service) AdminDisable(idOrUsername string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := listRecords(s.store)
	if err != nil {
		return User{}, err
	}
	for _, rec := range records {
		if !rec.enabled() {
			continue
		}
		if rec.UserID == idOrUsername || strings.EqualFold(rec.Username, idOrUsername) {
			if err := deleteRecord(s.store, rec.UserID); err != nil {
				return User{}, err
			}
			s.logger.Info().Str("userid", rec.UserID).Str("username", rec.Username).Msg("two-factor authentication disabled by an administrator")
			return User{ID: rec.UserID, Username: rec.Username}, nil
		}
	}
	return User{}, ErrUserNotFound
}

// verifyEnrolled loads the record of an enrolled user and checks the code. Failed attempts
// are persisted. The caller has to hold the lock.
func (s *Service) verifyEnrolled(user User, code string) (*record, error) {
	rec, err := readRecord(s.store, user.ID)
	if err != nil {
		return nil, err
	}
	if !rec.enabled() {
		return nil, ErrNotEnabled
	}

	ok, locked, err := s.verifyCode(rec, code)
	if err != nil {
		return nil, err
	}
	if !ok || locked {
		if err := writeRecord(s.store, rec); err != nil {
			return nil, err
		}
		if locked {
			return nil, ErrLocked
		}
		return nil, ErrInvalidCode
	}
	return rec, nil
}

// verifyCode checks a one-time code or a recovery code against the record and updates the
// record accordingly. The caller has to persist the record.
func (s *Service) verifyCode(rec *record, code string) (ok bool, locked bool, err error) {
	now := s.now()
	if now.Before(rec.LockedUntil) {
		return false, true, nil
	}
	rec.LockedUntil = time.Time{}

	code = normalizeCode(code)
	if isOneTimeCode(code) {
		secret, err := s.sealer.open(rec.Secret, rec.UserID)
		if err != nil {
			return false, false, err
		}
		var step int64
		if step, ok = verifyTOTP(secret, code, now, rec.LastStep); ok {
			rec.LastStep = step
		}
	} else if len(code) == recoveryCodeLength {
		hash := hashRecoveryCode(code)
		remaining := make([]string, 0, len(rec.RecoveryCodes))
		for _, h := range rec.RecoveryCodes {
			if subtle.ConstantTimeCompare([]byte(h), []byte(hash)) == 1 {
				// a recovery code can only be used once
				ok = true
				continue
			}
			remaining = append(remaining, h)
		}
		rec.RecoveryCodes = remaining
	}

	if ok {
		rec.FailedAttempts = 0
		return true, false, nil
	}

	rec.FailedAttempts++
	if rec.FailedAttempts >= maxFailedAttempts {
		rec.FailedAttempts = 0
		rec.LockedUntil = now.Add(lockDuration)
		return false, true, nil
	}
	return false, false, nil
}

// normalizeCode removes the characters users add when typing codes, e.g. "123 456" or
// "abcde-fghij".
func normalizeCode(code string) string {
	code = strings.ToLower(code)
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, code)
}

func isOneTimeCode(code string) bool {
	if len(code) != totpDigits {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// generateRecoveryCodes returns new recovery codes for the user and the hashes to store.
func generateRecoveryCodes() (codes []string, hashes []string, err error) {
	for i := 0; i < recoveryCodeCount; i++ {
		raw := make([]byte, recoveryCodeLength)
		if _, err := rand.Read(raw); err != nil {
			return nil, nil, err
		}
		for j := range raw {
			raw[j] = recoveryCodeAlphabet[int(raw[j])%len(recoveryCodeAlphabet)]
		}
		code := string(raw)
		// shown in two groups for readability, the dash is removed again by normalizeCode
		codes = append(codes, code[:recoveryCodeLength/2]+"-"+code[recoveryCodeLength/2:])
		hashes = append(hashes, hashRecoveryCode(code))
	}
	return codes, hashes, nil
}

// hashRecoveryCode hashes a normalized recovery code. The codes are random with 50 bit of
// entropy and attempts are limited, so a plain hash is sufficient.
func hashRecoveryCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
