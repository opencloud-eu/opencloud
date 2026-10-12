// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package authguest

import (
	"errors"
	"time"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/guard"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/storage"
)

var (
	ErrExpired             = errors.New("token expired")
	ErrAlreadyRedeemed     = errors.New("token already redeemed")
	ErrShareNotFound       = errors.New("share not found")
	ErrShareExpired        = errors.New("share expired")
	ErrEventsNotConfigured = errors.New("event publisher not configured")
	ErrPinInvalid          = errors.New("pin invalid")
	ErrPinExpired          = errors.New("pin expired")
	ErrNonceInvalid        = errors.New("nonce invalid")
	ErrNonceExpired        = errors.New("nonce expired")
	ErrTooManyAttempts     = errors.New("too many attempts")
	ErrPermissionMismatch  = errors.New("permissionId does not match the provided credentials")
)

// GuestError wraps an auth-guest failure together with the share id. The HTTP
// transport inspects ErrorType to choose a status code and message.
type GuestError struct {
	ErrorType  error
	ShareID    string
	RetryAfter time.Duration
}

func (e *GuestError) Error() string { return e.ErrorType.Error() }

func nonceError(err error) error {
	if errors.Is(err, guard.ErrNonceExpired) {
		return ErrNonceExpired
	}
	return ErrNonceInvalid
}

// nonceStoreError wraps a store error as a GuestError, treating a missing
// record as an invalid nonce.
func nonceStoreError(err error, shareID string) *GuestError {
	if errors.Is(err, storage.ErrNotFound) {
		err = ErrNonceInvalid
	}
	return &GuestError{ErrorType: err, ShareID: shareID}
}
