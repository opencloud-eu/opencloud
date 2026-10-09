// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"errors"
	"io/fs"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRecord(shareID string) Record {
	svc := token.NewTokenService()
	tok, _ := svc.Generate(shareID)

	return Record{
		ShareID:     shareID,
		ShareIDHash: tok.ShareIDHash,
		SecretHash:  tok.SecretHash(),
		Expiry:      time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
	}
}

func TestFileManagerAddGet(t *testing.T) {
	dir := t.TempDir()
	s := NewFileManager(dir)

	rec := newRecord("e0123456-7890-abcd-ef01-234567890abc")
	require.NoError(t, s.Add(rec))

	got, err := s.Get(rec.ShareIDHash)
	require.NoError(t, err)
	assert.Equal(t, rec, *got)
}

func TestFileManagerGetMissing(t *testing.T) {
	dir := t.TempDir()
	s := NewFileManager(dir)

	_, err := s.Get("doesnotexist")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestFileManagerAddExisting(t *testing.T) {
	dir := t.TempDir()
	s := NewFileManager(dir)

	rec := newRecord("e0123456-7890-abcd-ef01-234567890abc")
	require.NoError(t, s.Add(rec))

	rec.SecretHash = "other"
	require.ErrorIs(t, s.Add(rec), fs.ErrExist)
}

func TestFileManagerInvalidHash(t *testing.T) {
	dir := t.TempDir()
	s := NewFileManager(dir)

	_, err := s.Get("ab")
	require.ErrorIs(t, err, ErrInvalidHash)

	_, err = s.Get("../../etc/passwd-xyz")
	require.ErrorIs(t, err, ErrInvalidHash)

	require.ErrorIs(t, s.Remove("ab"), ErrInvalidHash)
	require.ErrorIs(t, s.Add(Record{ShareIDHash: "ab"}), ErrInvalidHash)
	require.ErrorIs(t, s.Update("ab", func(*Record) error { return nil }), ErrInvalidHash)
	_, err = s.UpdateFrom(Record{ShareIDHash: "ab"}, func(*Record) error { return nil })
	require.ErrorIs(t, err, ErrInvalidHash)
}

func TestFileManagerRemove(t *testing.T) {
	dir := t.TempDir()
	s := NewFileManager(dir)

	rec := newRecord("e0123456-7890-abcd-ef01-234567890abc")
	require.NoError(t, s.Add(rec))

	require.NoError(t, s.Remove(rec.ShareIDHash))

	_, err := s.Get(rec.ShareIDHash)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestFileManagerRemoveMissing(t *testing.T) {
	dir := t.TempDir()
	s := NewFileManager(dir)

	err := s.Remove("doesnotexist")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestFileManagerUpdate(t *testing.T) {
	dir := t.TempDir()
	s := NewFileManager(dir)

	rec := newRecord("e0123456-7890-abcd-ef01-234567890abc")
	require.NoError(t, s.Add(rec))

	require.NoError(t, s.Update(rec.ShareIDHash, func(r *Record) error {
		r.PinHash = "pinhash"
		r.Redeemed = true
		return nil
	}))

	got, err := s.Get(rec.ShareIDHash)
	require.NoError(t, err)
	assert.Equal(t, "pinhash", got.PinHash)
	assert.True(t, got.Redeemed)
	assert.Equal(t, uint64(1), got.Revision)
}

func TestFileManagerUpdateErrorLeavesRecordUnchanged(t *testing.T) {
	dir := t.TempDir()
	s := NewFileManager(dir)

	rec := newRecord("e0123456-7890-abcd-ef01-234567890abc")
	require.NoError(t, s.Add(rec))

	errBoom := errors.New("boom")
	err := s.Update(rec.ShareIDHash, func(r *Record) error {
		r.PinHash = "pinhash"
		return errBoom
	})
	assert.ErrorIs(t, err, errBoom)

	got, err := s.Get(rec.ShareIDHash)
	require.NoError(t, err)
	assert.Equal(t, rec, *got)
}

func TestFileManagerUpdateMissing(t *testing.T) {
	dir := t.TempDir()
	s := NewFileManager(dir)

	err := s.Update("doesnotexist", func(*Record) error { return nil })
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestFileManagerUpdateFrom(t *testing.T) {
	dir := t.TempDir()
	s := NewFileManager(dir)

	rec := newRecord("e0123456-7890-abcd-ef01-234567890abc")
	require.NoError(t, s.Add(rec))

	seen, err := s.Get(rec.ShareIDHash)
	require.NoError(t, err)

	updated, err := s.UpdateFrom(*seen, func(r *Record) error {
		r.PinHash = "pinhash"
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, seen.Revision+1, updated.Revision)
	assert.Equal(t, "pinhash", updated.PinHash)

	got, err := s.Get(rec.ShareIDHash)
	require.NoError(t, err)
	assert.Equal(t, *updated, *got)
}

func TestFileManagerUpdateFromConflict(t *testing.T) {
	dir := t.TempDir()
	s := NewFileManager(dir)

	rec := newRecord("e0123456-7890-abcd-ef01-234567890abc")
	require.NoError(t, s.Add(rec))

	seen, err := s.Get(rec.ShareIDHash)
	require.NoError(t, err)

	require.NoError(t, s.Update(rec.ShareIDHash, func(r *Record) error {
		r.SecretHash = "other"
		return nil
	}))

	_, err = s.UpdateFrom(*seen, func(r *Record) error {
		r.PinHash = "pinhash"
		return nil
	})
	assert.ErrorIs(t, err, ErrConflict)

	got, err := s.Get(rec.ShareIDHash)
	require.NoError(t, err)
	assert.Equal(t, "other", got.SecretHash)
	assert.Empty(t, got.PinHash)
}

func TestFileManagerUpdateFromMissing(t *testing.T) {
	dir := t.TempDir()
	s := NewFileManager(dir)

	_, err := s.UpdateFrom(Record{ShareIDHash: "doesnotexist"}, func(*Record) error { return nil })
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestFileManagerAddConcurrent(t *testing.T) {
	dir := t.TempDir()
	s := NewFileManager(dir)

	rec := newRecord("e0123456-7890-abcd-ef01-234567890abc")

	const workers = 20
	var (
		wg      sync.WaitGroup
		success atomic.Int32
	)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Add(rec); err == nil {
				success.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), success.Load())
}

func TestFileManagerUpdateConcurrent(t *testing.T) {
	dir := t.TempDir()
	s := NewFileManager(dir)

	rec := newRecord("e0123456-7890-abcd-ef01-234567890abc")
	require.NoError(t, s.Add(rec))

	const workers = 20
	var (
		wg      sync.WaitGroup
		success atomic.Int32
	)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.Update(rec.ShareIDHash, func(r *Record) error {
				if r.Redeemed {
					return errors.New("already set")
				}
				r.Redeemed = true
				return nil
			})
			if err == nil {
				success.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), success.Load())
}
