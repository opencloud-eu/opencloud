// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofrs/flock"
	"github.com/google/renameio/v2"
)

func NewFileManager(root string) *FileManager {
	return &FileManager{
		root: root,
	}
}

type FileManager struct {
	root string
}

const dirPerm = 0700
const filePerm = 0600

// minHashLength is the minimum share id hash length needed to derive a path.
const minHashLength = 4

func (s *FileManager) Add(rec Record) error {
	lock, err := s.lockRecord(rec.ShareIDHash)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Unlock() }()

	p, err := s.path(rec.ShareIDHash)
	if err != nil {
		return err
	}

	if _, err := os.Stat(p); err == nil {
		return fmt.Errorf("record %q already exists: %w", rec.ShareIDHash, fs.ErrExist)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	return s.upsert(rec)
}

// Get returns the record for the given share id hash.
func (s *FileManager) Get(shareIDHash string) (Record, error) {
	return s.get(shareIDHash)
}

func (s *FileManager) Remove(shareIDHash string) error {
	lock, err := s.lockRecord(shareIDHash)
	if err != nil {
		return err
	}
	defer func() {
		_ = lock.Unlock()
		_ = os.Remove(lock.Path())
	}()

	p, err := s.path(shareIDHash)
	if err != nil {
		return err
	}

	if err := os.Remove(p); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrNotFound
		}
		return err
	}

	return nil
}

// Update applies fn to the record under the record lock and writes it back.
// If fn returns an error the record is left unchanged.
func (s *FileManager) Update(shareIDHash string, fn func(*Record) error) error {
	lock, err := s.lockRecord(shareIDHash)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Unlock() }()

	rec, err := s.get(shareIDHash)
	if err != nil {
		return err
	}

	if err := fn(&rec); err != nil {
		return err
	}

	return s.upsert(rec)
}

func (s *FileManager) lockRecord(shareIDHash string) (*flock.Flock, error) {
	p, err := s.path(shareIDHash)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(p), dirPerm); err != nil {
		return nil, fmt.Errorf("could not create directory %s: %w", filepath.Dir(p), err)
	}

	lock := flock.New(p + ".lock")
	if err := lock.Lock(); err != nil {
		return nil, err
	}

	return lock, nil
}

func (s *FileManager) upsert(rec Record) error {
	p, err := s.path(rec.ShareIDHash)
	if err != nil {
		return err
	}

	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}

	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("could not create directory %s: %w", dir, err)
	}

	return renameio.WriteFile(p, data, filePerm)
}

func (s *FileManager) get(shareIDHash string) (Record, error) {
	p, err := s.path(shareIDHash)
	if err != nil {
		return Record{}, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Record{}, ErrNotFound
		}
		return Record{}, err
	}

	rec := Record{}
	if err := json.Unmarshal(data, &rec); err != nil {
		return Record{}, err
	}

	return rec, nil
}

func (s *FileManager) path(shareIDHash string) (string, error) {
	if len(shareIDHash) < minHashLength {
		return "", ErrInvalidHash
	}

	p := filepath.Join(s.root, shareIDHash[:2], shareIDHash[2:4], shareIDHash[4:]+".json")
	root := filepath.Clean(s.root)
	if !strings.HasPrefix(p, root+string(os.PathSeparator)) {
		return "", ErrInvalidHash
	}

	return p, nil
}
