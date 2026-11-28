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

package usermapper

import (
	"context"
)

// Mapper is the interface that wraps the basic mapping methods
type Mapper interface {
	RunInBaseScope(f func() error) error
	ScopeBase() (func() error, error)
	ScopeUser(ctx context.Context) (func() error, error)
	ScopeUserByIds(uid, gid int) (func() error, error)
}

// UnscopeFunc is a function that unscopes the current user
type UnscopeFunc func() error

// NullMapper is a user mapper that does nothing
type NullMapper struct{}

// RunInBaseScope runs the given function in the scope of the base user
func (nm *NullMapper) RunInBaseScope(f func() error) error {
	return f()
}

// ScopeBase returns to the base uid and gid returning a function that can be used to restore the previous scope
func (nm *NullMapper) ScopeBase() (func() error, error) {
	return func() error { return nil }, nil
}

// ScopeUser returns to the base uid and gid returning a function that can be used to restore the previous scope
func (nm *NullMapper) ScopeUser(ctx context.Context) (func() error, error) {
	return func() error { return nil }, nil
}

func (nm *NullMapper) ScopeUserByIds(uid, gid int) (func() error, error) {
	return func() error { return nil }, nil
}
