package search

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
)

// recoveringEngine guards the engine calls that read from or write to the index.
// Calls that cannot reach the index are promoted from the wrapped Engine.
type recoveringEngine struct {
	Engine

	log log.Logger
}

// NewRecoveringEngine wraps engine so that panics surface as errors instead of
// terminating the process
func NewRecoveringEngine(engine Engine, logger log.Logger) Engine {
	return &recoveringEngine{Engine: engine, log: logger}
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

func (e *recoveringEngine) Search(ctx context.Context, req *searchService.SearchIndexRequest) (res *searchService.SearchIndexResponse, err error) {
	defer recoverPanic(e.log, "Search", &err)

	return e.Engine.Search(ctx, req)
}

// the engine level mutations are guarded as well: the backend builds its own
// batch internally, which is not wrapped by recoveringBatch
func (e *recoveringEngine) Upsert(id string, r Resource) (err error) {
	defer recoverPanic(e.log, "Upsert", &err)

	return e.Engine.Upsert(id, r)
}

func (e *recoveringEngine) Move(id string, parentID string, targetPath string) (err error) {
	defer recoverPanic(e.log, "Move", &err)

	return e.Engine.Move(id, parentID, targetPath)
}

func (e *recoveringEngine) Delete(id string) (err error) {
	defer recoverPanic(e.log, "Delete", &err)

	return e.Engine.Delete(id)
}

func (e *recoveringEngine) Restore(id string) (err error) {
	defer recoverPanic(e.log, "Restore", &err)

	return e.Engine.Restore(id)
}

func (e *recoveringEngine) Purge(id string, onlyDeleted bool) (err error) {
	defer recoverPanic(e.log, "Purge", &err)

	return e.Engine.Purge(id, onlyDeleted)
}

func (e *recoveringEngine) PurgeSpace(rootID string) (err error) {
	defer recoverPanic(e.log, "PurgeSpace", &err)

	return e.Engine.PurgeSpace(rootID)
}

// NewBatch doesn't recover, creating a batch only allocates and never touches
// the index. It wraps the returned batch so its index writes are guarded too.
func (e *recoveringEngine) NewBatch(batchSize int) (BatchOperator, error) {
	next, err := e.Engine.NewBatch(batchSize)
	if err != nil || next == nil {
		return next, err
	}

	return &recoveringBatch{BatchOperator: next, log: e.log}, nil
}

// recoveringBatch guards the batch calls that read from or write to the index.
// Upsert is promoted, it only stages a document in the in-memory batch.
type recoveringBatch struct {
	BatchOperator

	log log.Logger
}

func (b *recoveringBatch) Move(id string, parentID string, targetPath string) (err error) {
	defer recoverPanic(b.log, "Batch.Move", &err)

	return b.BatchOperator.Move(id, parentID, targetPath)
}

func (b *recoveringBatch) Delete(id string) (err error) {
	defer recoverPanic(b.log, "Batch.Delete", &err)

	return b.BatchOperator.Delete(id)
}

func (b *recoveringBatch) Restore(id string) (err error) {
	defer recoverPanic(b.log, "Batch.Restore", &err)

	return b.BatchOperator.Restore(id)
}

func (b *recoveringBatch) Purge(id string, onlyDeleted bool) (err error) {
	defer recoverPanic(b.log, "Batch.Purge", &err)

	return b.BatchOperator.Purge(id, onlyDeleted)
}

func (b *recoveringBatch) Push() (err error) {
	defer recoverPanic(b.log, "Batch.Push", &err)

	return b.BatchOperator.Push()
}
