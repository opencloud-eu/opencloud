// Copyright 2026 OpenCloud GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package migration

import (
	"cmp"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/share"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/metadata"
	"github.com/rs/zerolog"
)

const stateFile = "migrations/state.json"

const (
	lockFile              = "migrations/lock.json"
	lockTTL               = time.Minute
	lockHeartbeatInterval = 20 * time.Second
)

// lockPollInterval is how long acquireLock sleeps between retries when the
// lock is held by another instance. Declared as a variable so tests can
// shorten it without rebuilding.
var lockPollInterval = 5 * time.Second

// lockData is the content written to the lock file.
type lockData struct {
	Timestamp  time.Time `json:"timestamp"`
	InstanceID string    `json:"instance_id"`
}

type migration interface {
	Name() string
	Version() int
	Initialize(config)
	Migrate() error
}

// persistedState is the on-disk representation of the migration state.
type persistedState struct {
	Version int `json:"version"`
}

type state struct {
	version int
}

// MigrationConfig holds all caller-supplied options for a migration run.
// It is intentionally a plain struct so that new fields can be added without
// changing function signatures throughout the call chain.
type MigrationConfig struct {
	ServiceAccountID     string
	ServiceAccountSecret string
	ProviderRegistryAddr string
}

type config struct {
	logger               zerolog.Logger
	gatewaySelector      pool.Selectable[gatewayv1beta1.GatewayAPIClient]
	storage              metadata.Storage
	serviceAccountID     string
	serviceAccountSecret string
	providerRegistryAddr string
	manager              share.Manager
	loader               share.LoadableManager
}

type Migrations struct {
	config
	state      state
	instanceID string
}

var migrations []migration

// registerMigration is only supposed to be call from init(), which runs sequentially
// so we don't need ot protect migrations with a lock
func registerMigration(m migration) {
	migrations = append(migrations, m)
}

func New(logger zerolog.Logger,
	gatewaySelector pool.Selectable[gatewayv1beta1.GatewayAPIClient],
	storage metadata.Storage,
	cfg MigrationConfig,
	manager share.Manager,
	loader share.LoadableManager,
) Migrations {

	slices.SortFunc(migrations, func(a, b migration) int {
		return cmp.Compare(a.Version(), b.Version())
	})

	b := make([]byte, 8)
	_, _ = rand.Read(b)
	instanceID := fmt.Sprintf("%x", b)

	return Migrations{
		config{
			logger:               logger.With().Str("jsoncs3", "migrations").Logger(),
			gatewaySelector:      gatewaySelector,
			storage:              storage,
			serviceAccountID:     cfg.ServiceAccountID,
			serviceAccountSecret: cfg.ServiceAccountSecret,
			providerRegistryAddr: cfg.ProviderRegistryAddr,
			manager:              manager,
			loader:               loader,
		},
		state{},
		instanceID,
	}
}

// acquireLock tries to atomically create the lock file, blocking until the lock
// is obtained. It returns the etag of the lock file on success. It retries
// indefinitely until ctx is cancelled. A lock whose timestamp is older than
// lockTTL is considered stale and will be taken over.
func (m *Migrations) acquireLock(ctx context.Context) (string, error) {
	m.logger.Info().Str("instance", m.instanceID).Msg("acquiring migration lock")
	for {
		// Fast path: create the lock file only if it does not exist yet.
		data, err := json.Marshal(lockData{Timestamp: time.Now(), InstanceID: m.instanceID})
		if err != nil {
			return "", err
		}
		res, err := m.storage.Upload(ctx, metadata.UploadRequest{
			Path:        lockFile,
			Content:     data,
			IfNoneMatch: []string{"*"},
		})
		if err == nil {
			m.logger.Info().Str("instance", m.instanceID).Msg("migration lock acquired")
			return res.Etag, nil
		}

		// Propagate context cancellation immediately.
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		// Any error other than a conflict means something unexpected happened.
		if !isConflict(err) {
			return "", err
		}

		// Lock file already exists — read it to decide whether it is stale.
		dl, err := m.storage.Download(ctx, metadata.DownloadRequest{Path: lockFile})
		if err != nil {
			if _, ok := err.(errtypes.IsNotFound); ok {
				// Lock was released between our upload attempt and the download;
				// retry acquiring it immediately.
				m.logger.Info().Str("instance", m.instanceID).Msg("migration lock vanished during read; retrying")
				continue
			}
			return "", err
		}

		var existing lockData
		stale := true
		if err := json.Unmarshal(dl.Content, &existing); err == nil {
			stale = time.Since(existing.Timestamp) > lockTTL
		}

		if stale {
			m.logger.Info().
				Str("instance", m.instanceID).
				Str("held_by", existing.InstanceID).
				Time("lock_timestamp", existing.Timestamp).
				Msg("migration lock is stale; attempting takeover")

			// Atomically take over the stale lock using the etag we just read.
			newData, err := json.Marshal(lockData{Timestamp: time.Now(), InstanceID: m.instanceID})
			if err != nil {
				return "", err
			}
			res, err := m.storage.Upload(ctx, metadata.UploadRequest{
				Path:        lockFile,
				Content:     newData,
				IfMatchEtag: dl.Etag,
			})
			if err == nil {
				m.logger.Info().Str("instance", m.instanceID).Msg("migration lock acquired via stale takeover")
				return res.Etag, nil
			}
			// Another instance took the stale lock before us; loop and retry.
			m.logger.Info().Str("instance", m.instanceID).Err(err).Msg("stale lock takeover lost race; retrying")
			continue
		}

		m.logger.Info().
			Str("instance", m.instanceID).
			Str("held_by", existing.InstanceID).
			Time("lock_timestamp", existing.Timestamp).
			Dur("poll_interval", lockPollInterval).
			Msg("migration lock held by another instance; waiting")

		// Lock is fresh and held by another instance; wait before retrying.
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(lockPollInterval):
		}
	}
}

// startHeartbeat spawns a goroutine that periodically renews the lock file so
// that it is not considered stale while a long migration is running. Call the
// returned cancel function to stop the heartbeat.
func (m *Migrations) startHeartbeat(ctx context.Context, etag string) context.CancelFunc {
	hbCtx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(lockHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-ticker.C:
				data, err := json.Marshal(lockData{Timestamp: time.Now(), InstanceID: m.instanceID})
				if err != nil {
					m.logger.Warn().Err(err).Msg("failed to marshal heartbeat data for migration lock")
					return
				}
				res, err := m.storage.Upload(hbCtx, metadata.UploadRequest{
					Path:        lockFile,
					Content:     data,
					IfMatchEtag: etag,
				})
				if err != nil {
					m.logger.Warn().Err(err).Msg("failed to renew migration lock; another instance may take over")
					return
				}
				etag = res.Etag
			}
		}
	}()
	return cancel
}

// releaseLock deletes the lock file unconditionally.
func (m *Migrations) releaseLock(ctx context.Context) {
	if err := m.storage.Delete(ctx, lockFile); err != nil {
		m.logger.Warn().Err(err).Msg("failed to release migration lock")
	}
}

// isConflict returns true for errors that signal a conditional-upload conflict,
// i.e. the lock file already exists or the etag did not match.
func isConflict(err error) bool {
	switch err.(type) {
	case errtypes.IsAlreadyExists, errtypes.IsAborted, errtypes.IsPreconditionFailed:
		return true
	}
	return false
}

// loadState reads the persisted migration version from storage. If no state
// file exists yet (fresh deployment) it returns version 0 without error.
func (m *Migrations) loadState(ctx context.Context) error {
	data, err := m.storage.SimpleDownload(ctx, stateFile)
	if err != nil {
		if _, ok := err.(errtypes.IsNotFound); ok {
			m.state = state{version: 0}
			return nil
		}
		return err
	}
	var ps persistedState
	if err := json.Unmarshal(data, &ps); err != nil {
		return err
	}
	m.state = state{version: ps.Version}
	return nil
}

// saveState writes the current migration version to storage so that already-
// applied migrations are not re-run on the next server start.
func (m *Migrations) saveState(ctx context.Context) error {
	data, err := json.Marshal(persistedState{Version: m.state.version})
	if err != nil {
		return err
	}
	return m.storage.SimpleUpload(ctx, stateFile, data)
}

func (m *Migrations) RunMigrations() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	etag, err := m.acquireLock(ctx)
	if err != nil {
		m.logger.Error().Err(err).Msg("failed to acquire migration lock; skipping migrations")
		return
	}
	cancelHB := m.startHeartbeat(ctx, etag)
	defer cancelHB()
	defer m.releaseLock(ctx)

	if err := m.loadState(ctx); err != nil {
		m.logger.Error().Err(err).Msg("failed to load migration state; skipping migrations")
		return
	}

	m.logger.Info().Int("current state", m.state.version).Msg("checking migrations")

	for _, mig := range migrations {
		if mig.Version() > m.state.version {
			m.logger.Info().Str("migration", mig.Name()).Int("version", mig.Version()).Msg("running migration")
			mig.Initialize(m.config)
			if err := mig.Migrate(); err != nil {
				m.logger.Error().Err(err).Str("migration", mig.Name()).Msg("migration failed; stopping")
				return
			}
			m.state.version = mig.Version()
			if err := m.saveState(ctx); err != nil {
				m.logger.Error().Err(err).Msg("failed to save migration state; stopping")
				return
			}
		} else {
			m.logger.Info().Str("migration", mig.Name()).Int("version", mig.Version()).Msg("skipping migration")
		}
	}
}
