package opensearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/opencloud-eu/reva/v2/pkg/utils"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var _ search.BatchOperator = (*Batch)(nil) // ensure Batch implements BatchOperator

type Batch struct {
	client     *opensearchgoAPI.Client
	index      string
	size       int
	log        log.Logger
	operations []any
	mu         sync.Mutex
}

func NewBatch(client *opensearchgoAPI.Client, index string, size int) (*Batch, error) {
	if size <= 0 {
		return nil, errors.New("batch size must be greater than 0")
	}

	return &Batch{
		client: client,
		size:   size,
		index:  index,
	}, nil
}

func (b *Batch) Upsert(id string, r search.Resource) error {
	return b.withSizeLimit(func() error {
		body, err := conversions.To[map[string]any](r)
		if err != nil {
			return fmt.Errorf("failed to marshal resource: %w", err)
		}

		op := func() []map[string]any {
			return []map[string]any{
				{"index": map[string]any{"_index": b.index, "_id": id}},
				body,
			}
		}

		b.mu.Lock()
		b.operations = append(b.operations, op)
		b.mu.Unlock()

		return nil
	})
}

func (b *Batch) Move(id, parentID, location string) error {
	return b.withSizeLimit(func() error {
		op := func() error {
			return updateSelfAndDescendants(context.Background(), b.client, b.index, id, func(rootResource search.Resource) *osu.BodyParamScript {
				return &osu.BodyParamScript{
					Source: `
					if (ctx._source.ID == params.id ) { ctx._source.Name = params.newName; ctx._source.ParentID = params.parentID; }
					ctx._source.Path = ctx._source.Path.replace(params.oldPath, params.newPath)
				`,
					Lang: "painless",
					Params: map[string]any{
						"id":       id,
						"parentID": parentID,
						"oldPath":  rootResource.Path,
						"newPath":  utils.MakeRelativePath(location),
						"newName":  path.Base(utils.MakeRelativePath(location)),
					},
				}
			})
		}

		b.mu.Lock()
		b.operations = append(b.operations, op)
		b.mu.Unlock()

		return nil
	})
}

func (b *Batch) Delete(id string) error {
	return b.withSizeLimit(func() error {
		op := func() error {
			return updateSelfAndDescendants(context.Background(), b.client, b.index, id, func(_ search.Resource) *osu.BodyParamScript {
				return &osu.BodyParamScript{
					Source: "ctx._source.Deleted = params.deleted",
					Lang:   "painless",
					Params: map[string]any{
						"deleted": true,
					},
				}
			})
		}

		b.mu.Lock()
		b.operations = append(b.operations, op)
		b.mu.Unlock()

		return nil
	})
}

func (b *Batch) Restore(id string) error {
	return b.withSizeLimit(func() error {
		op := func() error {
			return updateSelfAndDescendants(context.Background(), b.client, b.index, id, func(_ search.Resource) *osu.BodyParamScript {
				return &osu.BodyParamScript{
					Source: "ctx._source.Deleted = params.deleted",
					Lang:   "painless",
					Params: map[string]any{
						"deleted": false,
					},
				}
			})
		}

		b.mu.Lock()
		b.operations = append(b.operations, op)
		b.mu.Unlock()

		return nil
	})
}

func (b *Batch) Purge(id string, onlyDeleted bool) error {
	return b.withSizeLimit(func() error {
		resource, err := searchResourceByID(context.Background(), b.client, b.index, id)
		if err != nil {
			return fmt.Errorf("failed to get resource: %w", err)
		}

		query := osu.NewBoolQuery().Must(osu.NewTermQuery[string]("Path").Value(resource.Path))
		if onlyDeleted {
			query.Must(osu.NewTermQuery[bool]("Deleted").Value(true))
		}

		req, err := osu.BuildDocumentDeleteByQueryReq(
			opensearchgoAPI.DocumentDeleteByQueryReq{
				Indices: []string{b.index},
				Params: opensearchgoAPI.DocumentDeleteByQueryParams{
					WaitForCompletion: conversions.ToPointer(true),
				},
			},
			query,
		)
		if err != nil {
			return fmt.Errorf("failed to build delete by query request: %w", err)
		}

		op := func() error {
			resp, err := b.client.Document.DeleteByQuery(context.TODO(), req)
			switch {
			case err != nil:
				return fmt.Errorf("failed to delete by query: %w", err)
			case len(resp.Failures) != 0:
				return fmt.Errorf("failed to delete by query, failures: %v", resp.Failures)
			}

			return nil
		}

		b.mu.Lock()
		b.operations = append(b.operations, op)
		b.mu.Unlock()

		return nil
	})
}

func (b *Batch) Push() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	defer func() { // cleanup
		b.operations = nil
	}()

	var bulkOperations []map[string]any
	pushBulkOperations := func() error {
		if len(bulkOperations) == 0 {
			return nil
		}

		var body strings.Builder
		for _, operation := range bulkOperations {
			part, err := json.Marshal(operation)
			if err != nil {
				return fmt.Errorf("failed to marshal bulk operation: %w", err)
			}
			body.Write(part)
			body.WriteString("\n")
		}

		if _, err := b.client.Bulk(context.Background(), opensearchgoAPI.BulkReq{
			Body: strings.NewReader(body.String()),
		}); err != nil {
			return fmt.Errorf("failed to execute bulk operations: %w", err)
		}

		bulkOperations = nil
		return nil
	}

	// keep the order of operations in the batch intact,
	//  unfortunately, operations like DeleteByQuery cannot be part of the bulk API,
	//  so we need to push the previous bulk operations before executing such operations
	//  this might lead to smaller bulks than the configured size, but ensures correct order
	for _, operation := range b.operations {
		switch op := operation.(type) {
		case func() []map[string]any:
			bulkOperations = append(bulkOperations, op()...)
		case func() error:
			if err := pushBulkOperations(); err != nil {
				return fmt.Errorf("failed to push operations: %w", err)
			}
			if err := op(); err != nil {
				return fmt.Errorf("failed to execute operation: %w", err)
			}
		}
	}

	return pushBulkOperations()
}

func (b *Batch) withSizeLimit(f func() error) error {
	if err := f(); err != nil {
		return err
	}

	if len(b.operations) >= b.size {
		return b.Push()
	}

	return nil
}
