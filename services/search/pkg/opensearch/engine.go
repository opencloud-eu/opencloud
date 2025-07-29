package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/engine"
)

type Engine struct {
	index  string
	client *opensearchgoAPI.Client
}

func NewEngine(index string, client *opensearchgoAPI.Client) (*Engine, error) {
	return &Engine{index: index, client: client}, nil
}

func (e *Engine) Search(ctx context.Context, sir *searchService.SearchIndexRequest) (*searchService.SearchIndexResponse, error) {
	return &searchService.SearchIndexResponse{}, nil
}

func (e *Engine) Upsert(id string, r engine.Resource) error {
	body, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("failed to marshal resource: %w", err)
	}

	_, err = e.client.Document.Create(context.Background(), opensearchgoAPI.DocumentCreateReq{
		Index:      e.index,
		DocumentID: id,
		Body:       bytes.NewReader(body),
	})
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}

	return nil
}

func (e *Engine) Move(id string, parentID string, target string) error {
	return nil
}

func (e *Engine) Delete(id string) error {
	return nil
}

func (e *Engine) Restore(id string) error {
	return nil
}

func (e *Engine) Purge(id string) error {
	_, err := e.client.Document.Delete(context.Background(), opensearchgoAPI.DocumentDeleteReq{
		Index:      e.index,
		DocumentID: id,
	})
	if err != nil {
		return fmt.Errorf("failed to purge document: %w", err)
	}

	return nil
}

func (e *Engine) DocCount() (uint64, error) {
	return 0, nil
}
