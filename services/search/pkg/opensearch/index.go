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
// "opencloud-resource" -> the name suffixed with the current schema version.
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
	mimeType := overrides["MimeType"]
	mimeType.Type = searchmapping.TypeWildcard
	overrides["MimeType"] = mimeType
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
			"analysis": map[string]any{
				// path_hierarchy is case-preserving; casing lives in the value.
				"analyzer": map[string]any{
					"path_hierarchy": map[string]any{
						"type":      "custom",
						"tokenizer": "path_hierarchy",
					},
					// words: split into lowercased words, a dot is a word boundary
					// too so that "report" finds "Report.txt"; no stemming, a name
					// is not prose
					searchmapping.WordsAnalyzer: map[string]any{
						"type":        "custom",
						"char_filter": []string{"dot_to_space"},
						"tokenizer":   "standard",
						"filter":      []string{"lowercase"},
					},
				},
				"char_filter": map[string]any{
					"dot_to_space": map[string]any{
						"type":     "mapping",
						"mappings": []string{`. => \u0020`},
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

func coveredAt(declared, index gjson.Result, declaredPath, indexPath string) (string, string, bool) {
	declaredRaw := declared.Get(declaredPath).Raw
	indexRaw := index.Get(indexPath).Raw

	var declaredValue, indexValue any
	if err := json.Unmarshal([]byte(declaredRaw), &declaredValue); err != nil {
		return declaredRaw, indexRaw, false
	}

	if err := json.Unmarshal([]byte(indexRaw), &indexValue); err != nil {
		return declaredRaw, indexRaw, false
	}

	return declaredRaw, indexRaw, covered(declaredValue, indexValue)
}

func covered(declared, index any) bool {
	declaredMap, ok := declared.(map[string]any)
	if !ok {
		return reflect.DeepEqual(declared, index)
	}

	indexMap, ok := index.(map[string]any)
	if !ok {
		return false
	}

	for key, declaredValue := range declaredMap {
		indexValue, ok := indexMap[key]
		if !ok || !covered(declaredValue, indexValue) {
			return false
		}
	}

	return true
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

		remoteIndex, ok := (*resp.IndicesGetRespData)[name]
		if !ok {
			return fmt.Errorf("index %s not found in response", name)
		}
		remoteIndexB, err := json.Marshal(remoteIndex)
		if err != nil {
			return fmt.Errorf("failed to marshal index %s: %w", name, err)
		}

		localIndexJson := gjson.ParseBytes(localIndexB)
		remoteIndexJson := gjson.ParseBytes(remoteIndexB)

		var errs []error

		for k := range localIndexJson.Get("settings").Map() {
			if lv, rv, ok := coveredAt(localIndexJson, remoteIndexJson, "settings."+k, "settings.index."+k); !ok {
				errs = append(errs, fmt.Errorf("settings.%s local %s, remote %s", k, lv, rv))
			}
		}

		for k := range localIndexJson.Get("mappings.properties").Map() {
			if _, _, ok := coveredAt(localIndexJson, remoteIndexJson, "mappings.properties."+k, "mappings.properties."+k); !ok {
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
