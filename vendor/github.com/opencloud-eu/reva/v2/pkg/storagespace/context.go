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

package storagespace

import (
	"context"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"

	"github.com/opencloud-eu/reva/v2/pkg/storage"
)

type key int

const (
	sendOwnerChanKey key = iota
	moveResultSlotKey
)

// ContextSendSpaceOwnerID stores the space owner in the context.
func ContextSendSpaceOwnerID(ctx context.Context, id *userpb.UserId) {
	ch, ok := ctx.Value(sendOwnerChanKey).(chan<- *userpb.UserId)
	if ok {
		go func() {
			ch <- id
			close(ch)
		}()
	}
}

// ContextRegisterSendOwnerChan registers a channel to send the current space owner
func ContextRegisterSendOwnerChan(ctx context.Context, ch chan<- *userpb.UserId) context.Context {
	return context.WithValue(ctx, sendOwnerChanKey, ch)
}

// MoveResultSlot is a mutable container for a *storage.MoveResult.
type MoveResultSlot struct {
	Result *storage.MoveResult
}

// ContextRegisterMoveResultSlot stores an empty slot in ctx so the handler can fill it.
func ContextRegisterMoveResultSlot(ctx context.Context, slot *MoveResultSlot) context.Context {
	return context.WithValue(ctx, moveResultSlotKey, slot)
}

// ContextSetMoveResult fills the slot registered in ctx with the given result.
func ContextSetMoveResult(ctx context.Context, r *storage.MoveResult) {
	if slot, ok := ctx.Value(moveResultSlotKey).(*MoveResultSlot); ok {
		slot.Result = r
	}
}
