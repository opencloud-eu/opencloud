// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package goroutinelock

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"
)

// Lock tracks which goroutine currently owns a metadata lock.
//
// Held() only returns true in the owning goroutine, which prevents lock-held
// state from leaking across goroutine boundaries when helper nodes are shared.
type Lock struct {
	gid atomic.Uint64
}

var goidLogOnce sync.Once

func (o *Lock) Hold() {
	o.gid.Store(goid())
}

func (o *Lock) Release() {
	o.gid.Store(0)
}

func (o *Lock) Held() bool {
	return o.gid.Load() == goid()
}

func goid() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)

	// The header has the form "goroutine 1234 [running]:".
	fields := bytes.Fields(buf[:n])
	if len(fields) < 2 {
		goidLogOnce.Do(func() {
			log.Error().
				Str("stack_header", string(buf[:n])).
				Msg("goroutinelock: could not determine goroutine id from runtime stack header")
		})
		return 0
	}
	id, err := strconv.ParseUint(string(fields[1]), 10, 64)
	if err != nil {
		goidLogOnce.Do(func() {
			log.Error().Err(err).
				Str("goroutine_id_field", string(fields[1])).
				Msg("goroutinelock: could not parse goroutine id")
		})
		return 0
	}
	return id
}
