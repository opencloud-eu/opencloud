package mfa

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	microstore "go-micro.dev/v4/store"
)

// record is what is persisted per user.
type record struct {
	// UserID is the subject of the user as provided by the identifier backend.
	UserID string `json:"user_id"`
	// Username is only stored to make records readable for admins.
	Username string `json:"username"`

	// Secret is the encrypted TOTP secret. Two-factor authentication is enabled when it is set.
	Secret    []byte    `json:"secret,omitempty"`
	EnabledAt time.Time `json:"enabled_at,omitzero"`

	// PendingSecret is the encrypted TOTP secret of a setup that was not confirmed yet.
	PendingSecret  []byte    `json:"pending_secret,omitempty"`
	PendingExpires time.Time `json:"pending_expires,omitzero"`

	// RecoveryCodes holds the SHA-256 hashes of the unused recovery codes.
	RecoveryCodes []string `json:"recovery_codes,omitempty"`

	// LastStep is the time step of the last accepted one-time code (replay protection).
	LastStep int64 `json:"last_step,omitempty"`

	// FailedAttempts counts wrong codes since the last correct one.
	FailedAttempts int       `json:"failed_attempts,omitempty"`
	LockedUntil    time.Time `json:"locked_until,omitzero"`
}

func (r *record) enabled() bool {
	return r != nil && len(r.Secret) > 0
}

// storeKey returns the key of a user's record. The user id is hashed because it is
// backend specific and may contain characters the store does not accept in keys.
func storeKey(userID string) string {
	sum := sha256.Sum256([]byte(userID))
	return hex.EncodeToString(sum[:])
}

// readRecord returns nil if there is no record for the user.
func readRecord(st microstore.Store, userID string) (*record, error) {
	return readRecordByKey(st, storeKey(userID))
}

func readRecordByKey(st microstore.Store, key string) (*record, error) {
	recs, err := st.Read(key)
	if errors.Is(err, microstore.ErrNotFound) || (err == nil && len(recs) == 0) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	r := &record{}
	if err := json.Unmarshal(recs[0].Value, r); err != nil {
		return nil, err
	}
	return r, nil
}

func writeRecord(st microstore.Store, r *record) error {
	value, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return st.Write(&microstore.Record{Key: storeKey(r.UserID), Value: value})
}

func deleteRecord(st microstore.Store, userID string) error {
	err := st.Delete(storeKey(userID))
	if errors.Is(err, microstore.ErrNotFound) {
		return nil
	}
	return err
}

func listRecords(st microstore.Store) ([]*record, error) {
	keys, err := st.List()
	if err != nil {
		return nil, err
	}

	records := make([]*record, 0, len(keys))
	for _, key := range keys {
		r, err := readRecordByKey(st, key)
		if err != nil {
			return nil, err
		}
		if r != nil {
			records = append(records, r)
		}
	}
	return records, nil
}
