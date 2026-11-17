// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package guestlinks

import (
	"encoding/json"

	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
)

const (
	innerErrorType       = "opencloud_guest_link_error"
	reasonSessionExpired = "session_expired"
)

// errSessionExpired is returned when JWT expiry is the *only* validation
// failure. It implements errtypes.IsInvalidCredentials (-> CODE_UNAUTHENTICATED)
// and status.StatusInnerErrorProvider so that a safe, versioned JSON detail
// is attached to rpc.Status.InnerError.
type errSessionExpired struct {
	shareID string
}

func (e *errSessionExpired) Error() string {
	return "guestlinks: session expired"
}

// IsInvalidCredentials implements the errtypes.IsInvalidCredentials interface.
func (e *errSessionExpired) IsInvalidCredentials() {}

// innerErrorPayload is the JSON payload shape for the guest-link session
// expired InnerError.
type innerErrorPayload struct {
	Type    string `json:"type"`
	Reason  string `json:"reason"`
	ShareID string `json:"share_id"`
}

// StatusInnerError implements status.StatusInnerErrorProvider.
func (e *errSessionExpired) StatusInnerError() *types.OpaqueEntry {
	payload := innerErrorPayload{
		Type:    innerErrorType,
		Reason:  reasonSessionExpired,
		ShareID: e.shareID,
	}
	// encoding a static, safe struct: an error here can only happen on
	// programmer error (e.g. unmarshalable field), never in practice.
	b, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return &types.OpaqueEntry{
		Decoder: "json",
		Value:   b,
	}
}

func newSessionExpiredError(shareID string) error {
	return &errSessionExpired{shareID: shareID}
}
