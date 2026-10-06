// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("record not found")
var ErrInvalidHash = errors.New("invalid share id hash")
var ErrConflict = errors.New("record was modified concurrently")

// Record holds the data persisted for a guest share token.
type Record struct {
	ShareID     string    `json:"shareid"`
	ShareIDHash string    `json:"shareidhash"`
	SecretHash  string    `json:"secrethash"`
	PinHash     string    `json:"pinhash"`
	Expiry      time.Time `json:"expiry,omitzero"`
	PinExpiry   time.Time `json:"pinexpiry,omitzero"`
	Redeemed    bool      `json:"redeemed"`
	Revision    uint64    `json:"revision"`
}

type Manager interface {
	Add(rec Record) error
	Get(shareIDHash string) (*Record, error)
	Remove(shareIDHash string) error
	Update(shareIDHash string, fn func(*Record) error) error
	UpdateFrom(seen Record, fn func(*Record) error) (*Record, error)
}
