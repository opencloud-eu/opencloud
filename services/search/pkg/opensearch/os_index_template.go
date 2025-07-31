package opensearch

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"path"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

var (
	IndexTemplateResourceV1 IndexTemplate = [2]string{"opencloud-default-resource", "resource_v1.json"}
)

//go:embed internal/indices/*.json
var indexTemplates embed.FS

type IndexTemplate [2]string

func (t IndexTemplate) Name() string {
	return t[0]
}
func (t IndexTemplate) String() string {
	b, err := t.MarshalJSON()
	if err != nil {
		return ""
	}

	return string(b)
}

func (t IndexTemplate) MarshalJSON() ([]byte, error) {
	file := t[1]
	body, err := indexTemplates.ReadFile(path.Join("./internal/indices", file))
	switch {
	case err != nil:
		return nil, fmt.Errorf("failed to read index template file %s: %w", file, err)
	case len(body) <= 0:
		return nil, fmt.Errorf("index template file %s is empty", file)
	}

	return body, nil
}

func (t IndexTemplate) Apply(ctx context.Context, client *opensearchgoAPI.Client) error {
	body, err := t.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to inspect index template %s: %w", t[1], err)
	}

	resp, err := client.IndexTemplate.Create(ctx, opensearchgoAPI.IndexTemplateCreateReq{
		IndexTemplate: t.Name(),
		Body:          bytes.NewBuffer(body),
	})
	switch {
	case err != nil:
		return fmt.Errorf("failed to create index template %s: %w", t.Name(), err)
	case !resp.Acknowledged:
		return fmt.Errorf("failed to create index template %s: not acknowledged", t.Name())
	default:
		return nil
	}
}
