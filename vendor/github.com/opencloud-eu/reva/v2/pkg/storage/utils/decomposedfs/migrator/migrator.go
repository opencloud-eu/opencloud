// Copyright 2018-2021 CERN
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

package migrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/node"
	"github.com/rogpeppe/go-internal/lockedfile"
	"github.com/rs/zerolog"
)

const (
	statePending           = "pending"
	stateFailed            = "failed"
	stateSucceeded         = "succeeded"
	stateDown              = "down"
	stateSucceededRunAgain = "runagain"
)

type migration interface {
	Migrate(*Migrator) (Result, error)
	Rollback(*Migrator) (Result, error)
}

var migrations = map[string]migration{}

type migrationStates map[string]MigrationState

func registerMigration(name string, migration migration) {
	migrations[name] = migration
}

func allMigrations() []string {
	ms := make([]string, 0, len(migrations))

	for k := range migrations {
		ms = append(ms, k)
	}

	sort.Strings(ms)
	return ms
}

// MigrationState holds the state of a migration
type MigrationState struct {
	State   string
	Message string
}

// Result represents the result of a migration run
type Result string

// Migrator runs migrations on an existing decomposedfs
type Migrator struct {
	lu     node.PathLookup
	states migrationStates
	log    *zerolog.Logger
}

// New returns a new Migrator instance
func New(lu node.PathLookup, log *zerolog.Logger) Migrator {
	return Migrator{
		lu:  lu,
		log: log,
	}
}

// Migrations returns the list of migrations and their states
func (m *Migrator) Migrations() (map[string]MigrationState, error) {
	err := m.readStates()
	if err != nil {
		return nil, err
	}

	states := map[string]MigrationState{}
	for _, migration := range allMigrations() {
		if s, ok := m.states[migration]; ok {
			states[migration] = s
		} else {
			states[migration] = MigrationState{
				State: statePending,
			}
		}
	}

	return states, nil
}

// RunMigration runs or rolls back a migration
func (m *Migrator) RunMigration(id string, rollback bool) error {
	if _, ok := migrations[id]; !ok {
		return fmt.Errorf("invalid migration '%s'", id)
	}

	lock, err := lockedfile.OpenFile(filepath.Join(m.lu.InternalRoot(), ".migrations.lock"), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()

	err = m.readStates()
	if err != nil {
		return err
	}

	var res Result
	if !rollback {
		m.log.Info().Msg("Running migration " + id + "...")
		res, err = migrations[id].Migrate(m)
	} else {
		m.log.Info().Msg("Rolling back migration " + id + "...")
		res, err = migrations[id].Rollback(m)
	}

	// write back state
	s := m.states[id]
	s.State = string(res)

	if err != nil {
		m.log.Error().Err(err).Msg("migration " + id + " failed")
		s.Message = err.Error()
	}

	m.states[id] = s
	err = m.writeStates()
	if err != nil {
		return err
	}
	m.log.Info().Msg("done")
	return nil
}

// RunMigrations runs all migrations in sequence. Note this sequence must not be changed or it might
// damage existing decomposed fs.
func (m *Migrator) RunMigrations() error {
	lock, err := lockedfile.OpenFile(filepath.Join(m.lu.InternalRoot(), ".migrations.lock"), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()

	err = m.readStates()
	if err != nil {
		return err
	}

	for _, migration := range allMigrations() {
		s := m.states[migration]
		if s.State == stateSucceeded || s.State == stateDown {
			continue
		}

		res, err := migrations[migration].Migrate(m)
		s.State = string(res)
		if err != nil {
			m.log.Error().Err(err).Msg("migration " + migration + " failed")
			s.Message = err.Error()
		}

		m.states[migration] = s
		err = m.writeStates()
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Migrator) readStates() error {
	m.states = migrationStates{}

	d, err := os.ReadFile(filepath.Join(m.lu.InternalRoot(), ".migrations"))
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	if len(d) > 0 {
		err = json.Unmarshal(d, &m.states)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) writeStates() error {
	d, err := json.Marshal(m.states)
	if err != nil {
		m.log.Error().Err(err).Msg("could not marshal migration states")
		return nil
	}
	return os.WriteFile(filepath.Join(m.lu.InternalRoot(), ".migrations"), d, 0600)
}
