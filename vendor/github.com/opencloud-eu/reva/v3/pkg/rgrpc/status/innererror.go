// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"errors"

	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
)

// StatusInnerErrorProvider is an opt-in interface for error types that carry
// safe, already-encoded failure details meant to be transported in
// rpc.Status.InnerError.
type StatusInnerErrorProvider interface {
	// StatusInnerError returns an already encoded OpaqueEntry (decoder and
	// value) to be attached to rpc.Status.InnerError, or nil if no detail
	// should be attached.
	StatusInnerError() *types.OpaqueEntry
}

// InnerErrorFromErr looks for an error implementing StatusInnerErrorProvider
// in err's chain (using errors.As, so wrapped typed errors are supported)
// and returns the safe entry it provides.
func InnerErrorFromErr(err error) *types.OpaqueEntry {
	if err == nil {
		return nil
	}

	var provider StatusInnerErrorProvider
	if !errors.As(err, &provider) {
		return nil
	}

	return provider.StatusInnerError()
}
