package search

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
)

// RecoveringEngine guards the engine calls that read from or write to the index.
// Every Engine method is implemented explicitly so that a method added to the
// interface later can't be left unguarded by accident.
type RecoveringEngine struct {
	next Engine

	log log.Logger
}

// NewRecoveringEngine wraps engine so that panics surface as errors instead of
// terminating the process
func NewRecoveringEngine(engine Engine, logger log.Logger) *RecoveringEngine {
	return &RecoveringEngine{next: engine, log: logger}
}

// recoverPanic converts a panic into errp and logs it with the stack,
// this function should be used always with defer
func recoverPanic(logger log.Logger, op string, errp *error) {
	if r := recover(); r != nil {
		logger.Error().
			Str("op", op).
			Interface("panic", r).
			Str("stack", string(debug.Stack())).
			Msg("recovered panic from the search engine")

		*errp = fmt.Errorf("%s: search engine panic: %v", op, r)
	}
}

// DocCount delegates to the wrapped Engine and converts a panic into an error.
func (e *RecoveringEngine) DocCount() (count uint64, err error) {
	defer recoverPanic(e.log, "DocCount", &err)

	return e.next.DocCount()
}

// Search delegates to the wrapped Engine and converts a panic into an error.
func (e *RecoveringEngine) Search(ctx context.Context, req *searchService.SearchIndexRequest) (res *searchService.SearchIndexResponse, err error) {
	defer recoverPanic(e.log, "Search", &err)

	return e.next.Search(ctx, req)
}

// Upsert guards the engine level mutations as well: the backend builds its own
// batch internally, which is not wrapped by recoveringBatch
func (e *RecoveringEngine) Upsert(id string, r Resource) (err error) {
	defer recoverPanic(e.log, "Upsert", &err)

	return e.next.Upsert(id, r)
}

// Move delegates to the wrapped Engine and converts a panic into an error.
func (e *RecoveringEngine) Move(id string, parentID string, targetPath string) (err error) {
	defer recoverPanic(e.log, "Move", &err)

	return e.next.Move(id, parentID, targetPath)
}

// Delete delegates to the wrapped Engine and converts a panic into an error.
func (e *RecoveringEngine) Delete(id string) (err error) {
	defer recoverPanic(e.log, "Delete", &err)

	return e.next.Delete(id)
}

// Restore delegates to the wrapped Engine and converts a panic into an error.
func (e *RecoveringEngine) Restore(id string) (err error) {
	defer recoverPanic(e.log, "Restore", &err)

	return e.next.Restore(id)
}

// Purge delegates to the wrapped Engine and converts a panic into an error.
func (e *RecoveringEngine) Purge(id string, onlyDeleted bool) (err error) {
	defer recoverPanic(e.log, "Purge", &err)

	return e.next.Purge(id, onlyDeleted)
}

// PurgeSpace delegates to the wrapped Engine and converts a panic into an error.
func (e *RecoveringEngine) PurgeSpace(rootID string) (err error) {
	defer recoverPanic(e.log, "PurgeSpace", &err)

	return e.next.PurgeSpace(rootID)
}

// NewBatch doesn't recover, creating a batch only allocates and never touches
// the index. It wraps the returned batch so its index writes are guarded too.
func (e *RecoveringEngine) NewBatch(batchSize int) (BatchOperator, error) {
	next, err := e.next.NewBatch(batchSize)
	if err != nil || next == nil {
		return next, err
	}

	return &recoveringBatch{next: next, log: e.log}, nil
}

// recoveringBatch guards the batch calls that read from or write to the index.
// Upsert is guarded too: the batch flushes (Push) to the index when it is full
// (e.g. bleve withSizeLimit), so staging a document can reach the index.
type recoveringBatch struct {
	next BatchOperator

	log log.Logger
}

func (b *recoveringBatch) Upsert(id string, r Resource) (err error) {
	defer recoverPanic(b.log, "Batch.Upsert", &err)

	return b.next.Upsert(id, r)
}

func (b *recoveringBatch) Move(id string, parentID string, targetPath string) (err error) {
	defer recoverPanic(b.log, "Batch.Move", &err)

	return b.next.Move(id, parentID, targetPath)
}

func (b *recoveringBatch) Delete(id string) (err error) {
	defer recoverPanic(b.log, "Batch.Delete", &err)

	return b.next.Delete(id)
}

func (b *recoveringBatch) Restore(id string) (err error) {
	defer recoverPanic(b.log, "Batch.Restore", &err)

	return b.next.Restore(id)
}

func (b *recoveringBatch) Purge(id string, onlyDeleted bool) (err error) {
	defer recoverPanic(b.log, "Batch.Purge", &err)

	return b.next.Purge(id, onlyDeleted)
}

func (b *recoveringBatch) Push() (err error) {
	defer recoverPanic(b.log, "Batch.Push", &err)

	return b.next.Push()
}
