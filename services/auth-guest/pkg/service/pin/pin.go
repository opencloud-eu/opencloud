// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package pin

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/alexedwards/argon2id"
)

const Length = 6

var ErrInvalidPin = errors.New("invalid pin")
var ErrInvalidHash = errors.New("invalid pin hash")

func Generate() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(math.Pow10(Length))))
	if err != nil {
		return "", fmt.Errorf("could not generate random pin: %w", err)
	}

	return fmt.Sprintf("%0*d", Length, n.Int64()), nil
}

func Hash(p string) (string, error) {
	if len(p) != Length {
		return "", ErrInvalidPin
	}

	return argon2id.CreateHash(p, argon2id.DefaultParams)
}

func Verify(candidate, hash string) (bool, error) {
	if hash == "" {
		return false, ErrInvalidHash
	}

	match, err := argon2id.ComparePasswordAndHash(candidate, hash)
	if err != nil {
		if errors.Is(err, argon2id.ErrInvalidHash) {
			return false, ErrInvalidHash
		}
		return false, err
	}

	return match, nil
}
