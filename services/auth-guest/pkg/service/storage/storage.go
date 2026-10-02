// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("record not found")
var ErrAlreadyRedeemed = errors.New("token already redeemed")
var ErrInvalidHash = errors.New("invalid share id hash")

// Record holds the data persisted for a guest share token.
type Record struct {
	ShareID     string    `json:"shareid"`
	ShareIDHash string    `json:"shareidhash"`
	SecretHash  string    `json:"secrethash"`
	Expiry      time.Time `json:"expiry,omitzero"`
	Redeemed    bool      `json:"redeemed"`
}

type Manager interface {
	Add(rec Record) error
	Get(shareIDHash string) (Record, error)
	Remove(shareIDHash string) error
	Redeem(shareIDHash string) error
}
