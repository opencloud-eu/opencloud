package osu

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"path"
	"reflect"

	"github.com/go-jose/go-jose/v3/json"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/tidwall/gjson"
)

var (
	ErrManualActionRequired                  = errors.New("manual action required")
	IndexManagerLatest                       = IndexIndexManagerResourceV1
	IndexIndexManagerResourceV1 IndexManager = "resource_v1.json"
)

//go:embed indexes/*.json
var indexes embed.FS

type IndexManager string

func (m IndexManager) String() string {
	b, err := m.MarshalJSON()
	if err != nil {
		return ""
	}

	return string(b)
}

func (m IndexManager) MarshalJSON() ([]byte, error) {
	filePath := string(m)
	body, err := indexes.ReadFile(path.Join("./indexes", filePath))
	switch {
	case err != nil:
		return nil, fmt.Errorf("failed to read index file %s: %w", filePath, err)
	case len(body) <= 0:
		return nil, fmt.Errorf("index file %s is empty", filePath)
	}

	return body, nil
}

func (m IndexManager) Apply(ctx context.Context, name string, client *opensearchgoAPI.Client) error {
	localIndexB, err := m.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal index %s: %w", name, err)
	}

	indicesExistsResp, err := client.Indices.Exists(ctx, opensearchgoAPI.IndicesExistsReq{
		Indices: []string{name},
	})
	switch {
	case indicesExistsResp != nil && indicesExistsResp.StatusCode == 404:
		break
	case err != nil:
		return fmt.Errorf("failed to check if index %s exists: %w", name, err)
	case indicesExistsResp == nil:
		return fmt.Errorf("indicesExistsResp is nil for index %s", name)
	}

	if indicesExistsResp.StatusCode == 200 {
		resp, err := client.Indices.Get(ctx, opensearchgoAPI.IndicesGetReq{
			Indices: []string{name},
		})
		if err != nil {
			return fmt.Errorf("failed to get index %s: %w", name, err)
		}

		remoteIndex, ok := resp.Indices[name]
		if !ok {
			return fmt.Errorf("index %s not found in response", name)
		}
		remoteIndexB, err := json.Marshal(remoteIndex)
		if err != nil {
			return fmt.Errorf("failed to marshal index %s: %w", name, err)
		}

		localIndexJson := gjson.ParseBytes(localIndexB)
		remoteIndexJson := gjson.ParseBytes(remoteIndexB)

		compare := func(lvPath, rvPath string) (any, any, bool) {
			lv := localIndexJson.Get(lvPath).Raw
			rv := remoteIndexJson.Get(rvPath).Raw

			var lvv, rvv interface{}
			if err := json.Unmarshal([]byte(lv), &lvv); err != nil {
				return nil, nil, false
			}

			if err := json.Unmarshal([]byte(rv), &rvv); err != nil {
				return nil, nil, false
			}

			return lv, rv, reflect.DeepEqual(lvv, rvv)
		}

		var errs []error

		for k := range localIndexJson.Get("settings").Map() {
			if lv, rv, ok := compare("settings."+k, "settings.index."+k); !ok {
				errs = append(errs, fmt.Errorf("settings.%s local %s, remote %s", k, lv, rv))
			}
		}

		for k := range localIndexJson.Get("mappings.properties").Map() {
			if _, _, ok := compare("mappings.properties."+k, "mappings.properties."+k); !ok {
				errs = append(errs, fmt.Errorf("mappings.properties.%s", k))
			}
		}

		if errs != nil {
			return fmt.Errorf(
				"index %s allready exists and is different from the requested version, %w: %w",
				name,
				ErrManualActionRequired,
				errors.Join(errs...),
			)
		}

		return nil // Index is already up to date, no action needed
	}

	createResp, err := client.Indices.Create(ctx, opensearchgoAPI.IndicesCreateReq{
		Index: name,
		Body:  bytes.NewReader(localIndexB),
	})
	switch {
	case err != nil:
		return fmt.Errorf("failed to create index %s: %w", name, err)
	case !createResp.Acknowledged:
		return fmt.Errorf("failed to create index %s: not acknowledged", name)
	}

	return nil
}
