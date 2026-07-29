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
	ErrManualActionRequired = searchmapping.ErrManualActionRequired

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

// Apply ensures the index exists and matches the code schema: created if
// missing, otherwise reconciled via searchmapping.Reconcile (see osReconciler).
func (m IndexManager) Apply(ctx context.Context, name string, client *opensearchgoAPI.Client, logger log.Logger) error {
	localIndexB, err := m.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal index %s: %w", name, err)
	}

	// Exists first: a pre-provisioned index must not require create privileges
	indicesExistsResp, err := client.Indices.Exists(ctx, opensearchgoAPI.IndicesExistsReq{
		Indices: []string{name},
	})
	switch {
	case indicesExistsResp != nil && indicesExistsResp.StatusCode == 404:
		createResp, createErr := client.Indices.Create(ctx, opensearchgoAPI.IndicesCreateReq{
			Index: name,
			Body:  bytes.NewReader(localIndexB),
		})
		var structErr *opensearchgo.StructError
		switch {
		case createErr == nil && createResp.Acknowledged:
			searchmapping.LogNewIndexCreated(logger, name)
			return nil
		case createErr == nil:
			return fmt.Errorf("failed to create index %s: not acknowledged", name)
		case !errors.As(createErr, &structErr) || structErr.Err.Type != "resource_already_exists_exception":
			// transport errors, disk-full etc. stay plain fatal, the restart policy retries
			return fmt.Errorf("failed to create index %s: %w", name, createErr)
		}
		// lost the creation race to another instance, compare against its index
	case err != nil:
		return fmt.Errorf("failed to check if index %s exists: %w", name, err)
	case indicesExistsResp == nil:
		return fmt.Errorf("indicesExistsResp is nil for index %s", name)
	}

	// the index exists: reconcile its schema through the shared verdict flow
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

	r := &osReconciler{
		ctx:    ctx,
		name:   name,
		client: client,
		local:  gjson.ParseBytes(localIndexB),
		remote: gjson.ParseBytes(remoteIndexB),
	}
	_, err = searchmapping.Reconcile(name, r, logger)
	return err
}

// osReconciler adapts an existing OpenSearch index to searchmapping.SchemaReconciler.
type osReconciler struct {
	ctx    context.Context
	name   string
	client *opensearchgoAPI.Client
	local  gjson.Result
	remote gjson.Result
}

func (r *osReconciler) Classify() (searchmapping.Classification, error) {
	// Only the analysis settings affect indexing correctness; shard/replica
	// counts and other operational knobs are the operator's to tune (and a
	// pre-provisioned index's to own), so they are not compared.
	var reasons []string
	lv := r.local.Get("settings.analysis").Raw
	rv := r.remote.Get("settings.index.analysis").Raw
	if !jsonEqual(lv, rv) {
		reasons = append(reasons, fmt.Sprintf("settings.analysis changed: index %s, code %s", rawOrUnset(rv), rawOrUnset(lv)))
	}

	classification := searchmapping.Classify(
		propertiesMap(r.remote.Get("mappings.properties").Raw),
		propertiesMap(r.local.Get("mappings.properties").Raw),
		nil,
	)
	reasons = append(reasons, classification.Reasons...)
	if len(reasons) > 0 {
		classification.Verdict = searchmapping.VerdictBreaking
		classification.Reasons = reasons
	}
	return classification, nil
}

// ApplyAdditive puts the full code properties (only additions, per the
// classifier). The PUT is atomic, so persisted is true only on success.
func (r *osReconciler) ApplyAdditive() (bool, error) {
	putResp, err := r.client.Indices.Mapping.Put(r.ctx, opensearchgoAPI.MappingPutReq{
		Indices: []string{r.name},
		Body:    strings.NewReader(r.local.Get("mappings").Raw),
	})
	var putErr *opensearchgo.StructError
	switch {
	case err != nil && errors.As(err, &putErr) && putErr.Err.Type == "illegal_argument_exception" &&
		(strings.Contains(putErr.Err.Reason, "cannot be changed") || strings.Contains(putErr.Err.Reason, "Cannot update parameter")):
		// backstop, should be unreachable after the classification above
		return false, searchmapping.ManualActionRequiredError(r.name, []string{putErr.Err.Reason})
	case err != nil:
		return false, fmt.Errorf("failed to update mapping of index %s: %w", r.name, err)
	case !putResp.Acknowledged:
		return false, fmt.Errorf("failed to update mapping of index %s: not acknowledged", r.name)
	}
	return true, nil
}

// jsonEqual reports whether two raw JSON values are deeply equal. gjson yields
// an empty string for a path that does not exist; two such unset values are
// equal, an unset value never equals a present one, and a value that fails to
// parse counts as unequal.
func jsonEqual(a, b string) bool {
	if a == "" || b == "" {
		return a == b
	}
	var av, bv any
	if err := json.Unmarshal([]byte(a), &av); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &bv); err != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}

// propertiesMap parses a raw mappings.properties object into a map. Missing,
// empty, null or malformed input yields an empty (non-nil) map, which
// classifies as purely additive.
func propertiesMap(raw string) map[string]any {
	props := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &props); err != nil || props == nil {
		return map[string]any{}
	}
	return props
}

func rawOrUnset(raw string) string {
	if raw == "" {
		return "(unset)"
	}
	return raw
}
