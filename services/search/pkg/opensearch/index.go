package opensearch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"

	"github.com/go-jose/go-jose/v3/json"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/tidwall/gjson"

	searchmapping "github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var (
	ErrManualActionRequired = errors.New("manual action required")

	// IndexManagerLatest identifies the current resource mapping; its version is
	// derived from search.SchemaVersion so it never drifts from the index name.
	IndexManagerLatest = IndexManager(fmt.Sprintf("resource_v%d", search.SchemaVersion))
)

// VersionedIndexName suffixes the base index name with the schema version, e.g.
// "opencloud-resource" -> "opencloud-resource-v3".
func VersionedIndexName(base string) string {
	return fmt.Sprintf("%s-v%d", base, search.SchemaVersion)
}

type IndexManager string

// indexGenerators dispatches each IndexManager variant to its builder.
var indexGenerators = map[IndexManager]func() ([]byte, error){
	IndexManagerLatest: buildResourceMapping,
}

func (m IndexManager) String() string {
	b, err := m.MarshalJSON()
	if err != nil {
		return ""
	}

	return string(b)
}

func (m IndexManager) MarshalJSON() ([]byte, error) {
	gen, ok := indexGenerators[m]
	if !ok {
		return nil, fmt.Errorf("unknown index manager %q", string(m))
	}
	return gen()
}

// buildResourceMapping renders the OpenSearch index template for a
// search.Resource from the shared SearchFieldOverrides. OpenSearch-specific
// tweaks (wildcard MimeType, path_hierarchy Path) are applied on top.
func buildResourceMapping() ([]byte, error) {
	resourceType := reflect.TypeFor[search.Resource]()
	overrides := maps.Clone(search.Resource{}.SearchFieldOverrides())
	overrides["MimeType"] = searchmapping.FieldOpts{Type: searchmapping.TypeWildcard}
	if err := searchmapping.Validate(resourceType, overrides); err != nil {
		return nil, err
	}
	props, err := searchmapping.OpenSearchBuildMapping(resourceType, overrides)
	if err != nil {
		return nil, err
	}

	index := map[string]any{
		"settings": map[string]any{
			"number_of_shards":   "1",
			"number_of_replicas": "1",
			// static setting, must be set at creation time: enables the knn
			// query type for the imageVector field
			"knn": "true",
			"analysis": map[string]any{
				// path_hierarchy is case-preserving; casing lives in the value.
				// fulltext mirrors the bleve fulltext analyzer (lowercase + porter
				// stemming) so full-text search behaves the same on both backends.
				"analyzer": map[string]any{
					"path_hierarchy": map[string]any{
						"type":      "custom",
						"tokenizer": "path_hierarchy",
					},
					"fulltext": map[string]any{
						"type":      "custom",
						"tokenizer": "standard",
						"filter":    []string{"lowercase", "porter_stem"},
					},
				},
				"tokenizer": map[string]any{
					"path_hierarchy": map[string]any{"type": "path_hierarchy"},
				},
			},
		},
		"mappings": map[string]any{
			"properties": props,
		},
	}
	return json.Marshal(index)
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

			var lvv, rvv any
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
				"index %s already exists with a different mapping than the requested version. "+
					"There is no in-place migration today: drop the index in OpenSearch (DELETE /%s) "+
					"and restart the search service. The index will be recreated with the new mapping. "+
					"%w: %w",
				name, name,
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
