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

package ctx

import (
	"context"
)

// ContextGetLockID returns the lock id if set in the given context.
func ContextGetLockID(ctx context.Context) (string, bool) {
	u, ok := ctx.Value(lockIDKey).(string)
	return u, ok
}

// ContextSetLockID stores the lock id in the context.
func ContextSetLockID(ctx context.Context, t string) context.Context {
	return context.WithValue(ctx, lockIDKey, t)
}
