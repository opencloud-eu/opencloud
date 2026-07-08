package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"

	opensearchgo "github.com/opensearch-project/opensearch-go/v4"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/tidwall/gjson"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchmapping "github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var (
	// ErrManualActionRequired is the shared sentinel, see the mapping package.
	ErrManualActionRequired                  = searchmapping.ErrManualActionRequired
	IndexManagerLatest                       = IndexIndexManagerResourceV2
	IndexIndexManagerResourceV2 IndexManager = "resource_v2"
)

type IndexManager string

// indexGenerators dispatches each IndexManager variant to its builder.
var indexGenerators = map[IndexManager]func() ([]byte, error){
	IndexIndexManagerResourceV2: buildResourceV2Mapping,
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

// buildResourceV2Mapping renders the OpenSearch index template for a
// search.Resource from the shared SearchFieldOverrides. OpenSearch-specific
// tweaks (wildcard MimeType, path_hierarchy Path) are applied on top.
func buildResourceV2Mapping() ([]byte, error) {
	resourceType := reflect.TypeFor[search.Resource]()
	overrides := maps.Clone(search.Resource{}.SearchFieldOverrides())
	overrides["MimeType"] = searchmapping.FieldOpts{Type: searchmapping.TypeWildcard}
	overrides["Path"] = searchmapping.FieldOpts{Type: searchmapping.TypePath}
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
				"analyzer": map[string]any{
					"path_hierarchy": map[string]any{
						"type":      "custom",
						"tokenizer": "path_hierarchy",
						"filter":    []string{"lowercase"},
					},
					"lowercaseKeyword": map[string]any{
						"type":      "custom",
						"tokenizer": "keyword",
						"filter":    []string{"lowercase"},
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

// Apply ensures the index exists and matches the schema generated from code:
// created if missing, additive changes applied via PUT _mapping, breaking ones
// refused with ErrManualActionRequired. The classifier judges, PUT _mapping
// only applies (its merge semantics hide removals and renames).
func (m IndexManager) Apply(ctx context.Context, name string, client *opensearchgoAPI.Client, logger log.Logger) error {
	localIndexB, err := m.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal index %s: %w", name, err)
	}

	createResp, err := client.Indices.Create(ctx, opensearchgoAPI.IndicesCreateReq{
		Index: name,
		Body:  bytes.NewReader(localIndexB),
	})
	var createErr *opensearchgo.StructError
	switch {
	case err == nil && createResp.Acknowledged:
		return nil
	case err == nil:
		return fmt.Errorf("failed to create index %s: not acknowledged", name)
	case !errors.As(err, &createErr) || createErr.Err.Type != "resource_already_exists_exception":
		// transport errors, disk-full etc. stay plain fatal, the restart policy retries
		return fmt.Errorf("failed to create index %s: %w", name, err)
	}

	// the index already exists: compare settings and classify the mapping diff
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

	var reasons []string
	for k := range localIndexJson.Get("settings").Map() {
		lv := localIndexJson.Get("settings." + k).Raw
		rv := remoteIndexJson.Get("settings.index." + k).Raw
		if !jsonEqual(lv, rv) {
			reasons = append(reasons, fmt.Sprintf("settings.%s changed: index %s, code %s", k, rawOrUnset(rv), rawOrUnset(lv)))
		}
	}

	classification := searchmapping.Classify(
		propertiesMap(remoteIndexJson.Get("mappings.properties").Raw),
		propertiesMap(localIndexJson.Get("mappings.properties").Raw),
		nil,
	)
	reasons = append(reasons, classification.Reasons...)
	if len(reasons) > 0 {
		return searchmapping.ManualActionRequiredError(name, fmt.Sprintf("delete the index (DELETE /%s)", name), reasons)
	}
	if len(classification.NewFields) == 0 {
		return nil // schema is up to date
	}

	// additive: the classifier guarantees every existing field matches the
	// remote state, so putting the full code properties can only add fields
	putResp, err := client.Indices.Mapping.Put(ctx, opensearchgoAPI.MappingPutReq{
		Indices: []string{name},
		Body:    strings.NewReader(localIndexJson.Get("mappings").Raw),
	})
	var putErr *opensearchgo.StructError
	switch {
	case err != nil && errors.As(err, &putErr) && putErr.Err.Type == "illegal_argument_exception" &&
		(strings.Contains(putErr.Err.Reason, "cannot be changed") || strings.Contains(putErr.Err.Reason, "Cannot update parameter")):
		// backstop, should be unreachable after the classification above
		return searchmapping.ManualActionRequiredError(name, fmt.Sprintf("delete the index (DELETE /%s)", name), []string{putErr.Err.Reason})
	case err != nil:
		return fmt.Errorf("failed to update mapping of index %s: %w", name, err)
	case !putResp.Acknowledged:
		return fmt.Errorf("failed to update mapping of index %s: not acknowledged", name)
	}

	logger.Warn().Strs("fields", classification.NewFields).Str("index", name).Msg("extended the search index mapping with new fields; documents indexed before the upgrade do not contain them and queries on these fields will miss those documents until they are re-indexed; to re-index everything run: opencloud search index --all-spaces")
	return nil
}

func jsonEqual(a, b string) bool {
	var av, bv any
	if err := json.Unmarshal([]byte(a), &av); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &bv); err != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}

// propertiesMap parses a raw mappings.properties object; missing or empty
// input yields an empty map, which classifies as purely additive.
func propertiesMap(raw string) map[string]any {
	props := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &props)
	return props
}

func rawOrUnset(raw string) string {
	if raw == "" {
		return "(unset)"
	}
	return raw
}
