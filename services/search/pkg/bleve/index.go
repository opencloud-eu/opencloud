package bleve

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/token/porter"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/single"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchmapping "github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// bolt_timeout makes a second process on the same datapath fail after 5s
// instead of blocking forever on the file lock.
var openRuntimeConfig = map[string]interface{}{"bolt_timeout": "5s"}

// NewIndex opens (or creates) the bleve index at root and reconciles its schema
// against NewMapping() via searchmapping.Reconcile.
func NewIndex(root string, logger log.Logger) (bleve.Index, searchmapping.Classification, error) {
	destination := filepath.Join(root, fmt.Sprintf("bleve-v%d", search.SchemaVersion))
	index, err := bleve.OpenUsing(destination, openRuntimeConfig)
	if errors.Is(err, bleve.ErrorIndexPathDoesNotExist) {
		indexMapping, err := NewMapping()
		if err != nil {
			return nil, searchmapping.Classification{}, err
		}
		index, err = bleve.New(destination, indexMapping)
		if err != nil {
			return nil, searchmapping.Classification{}, err
		}

		searchmapping.LogNewIndexCreated(logger, destination)
		return index, searchmapping.Classification{Verdict: searchmapping.VerdictEqual}, nil
	}
	if err != nil {
		return nil, searchmapping.Classification{}, err
	}

	r := &bleveReconciler{index: index, destination: destination}
	classification, err := searchmapping.Reconcile(destination, r, logger)
	if err != nil {
		if r.index != nil {
			_ = r.index.Close()
		}
		return nil, classification, err
	}

	return r.index, classification, nil
}

// bleveReconciler adapts a bleve index to searchmapping.SchemaReconciler.
type bleveReconciler struct {
	index       bleve.Index
	destination string
	codeB       []byte // marshaled code mapping, produced by Classify, used by ApplyAdditive
}

func (r *bleveReconciler) Classify() (searchmapping.Classification, error) {
	classification, codeB, err := classifyStoredMapping(r.index)
	r.codeB = codeB
	return classification, err
}

// ApplyAdditive persists the code mapping and reopens so the live mapping picks
// it up. persisted=true once SetInternal succeeds, even if the reopen then
// fails; on error it closes the index and clears the handle.
func (r *bleveReconciler) ApplyAdditive() (bool, error) {
	if err := r.index.SetInternal([]byte("_mapping"), r.codeB); err != nil {
		_ = r.index.Close()
		r.index = nil
		return false, fmt.Errorf("failed to store the updated index mapping: %w", err)
	}
	if err := r.index.Close(); err != nil {
		r.index = nil
		return true, err
	}
	index, err := bleve.OpenUsing(r.destination, openRuntimeConfig)
	if err != nil {
		r.index = nil
		return true, err
	}
	r.index = index
	return true, nil
}

// classifyStoredMapping diffs the stored mapping against NewMapping() and
// returns the marshaled code mapping. New-in-code fields that already hold
// data (previously indexed dynamically) are breaking. The compare is only
// stable within one bleve version: a changed marshaling default fails towards
// breaking, normalize the affected key here if that ever fires.
func classifyStoredMapping(index bleve.Index) (searchmapping.Classification, []byte, error) {
	storedB, err := index.GetInternal([]byte("_mapping"))
	if err != nil {
		return searchmapping.Classification{}, nil, fmt.Errorf("failed to read the stored index mapping: %w", err)
	}
	codeMapping, err := NewMapping()
	if err != nil {
		return searchmapping.Classification{}, nil, err
	}
	codeB, err := json.Marshal(codeMapping)
	if err != nil {
		return searchmapping.Classification{}, nil, err
	}

	var stored, code map[string]any
	if err := json.Unmarshal(storedB, &stored); err != nil {
		return searchmapping.Classification{}, nil, fmt.Errorf("failed to parse the stored index mapping: %w", err)
	}
	if err := json.Unmarshal(codeB, &code); err != nil {
		return searchmapping.Classification{}, nil, err
	}

	fields, err := index.Fields()
	if err != nil {
		return searchmapping.Classification{}, nil, fmt.Errorf("failed to list the indexed fields: %w", err)
	}
	indexedFields := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		if !strings.HasPrefix(f, "_") { // skip bleve-internal fields like _all
			indexedFields[f] = struct{}{}
		}
	}

	storedDM, _ := stored["default_mapping"].(map[string]any)
	codeDM, _ := code["default_mapping"].(map[string]any)
	storedProps, _ := storedDM["properties"].(map[string]any)
	codeProps, _ := codeDM["properties"].(map[string]any)

	classification := searchmapping.Classify(storedProps, codeProps, func(path string) bool {
		if _, ok := indexedFields[path]; ok {
			return true
		}
		nested := path + "."
		for f := range indexedFields {
			if strings.HasPrefix(f, nested) {
				return true
			}
		}
		return false
	})

	// everything outside default_mapping.properties (analyzer definitions,
	// default analyzer, dynamic flags, ...) must match exactly
	var reasons []string
	compareKeysExcept(stored, code, "default_mapping", "", &reasons)
	compareKeysExcept(storedDM, codeDM, "properties", "default_mapping.", &reasons)
	if len(reasons) > 0 {
		classification.Verdict = searchmapping.VerdictBreaking
		classification.Reasons = append(reasons, classification.Reasons...)
	}

	return classification, codeB, nil
}

// compareKeysExcept deep-compares all keys present on either side except skip.
func compareKeysExcept(stored, code map[string]any, skip, prefix string, reasons *[]string) {
	keys := slices.Collect(maps.Keys(stored))
	for k := range code {
		if _, ok := stored[k]; !ok {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	for _, k := range keys {
		if k == skip {
			continue
		}
		if !reflect.DeepEqual(stored[k], code[k]) {
			*reasons = append(*reasons, fmt.Sprintf("%s%s changed", prefix, k))
		}
	}
}

func NewMapping() (mapping.IndexMapping, error) {
	resourceType := reflect.TypeFor[search.Resource]()
	overrides := search.Resource{}.SearchFieldOverrides()
	if err := searchmapping.Validate(resourceType, overrides); err != nil {
		return nil, err
	}
	docMapping, err := searchmapping.BleveBuildMapping(resourceType, overrides)
	if err != nil {
		return nil, err
	}

	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultAnalyzer = keyword.Name
	indexMapping.DefaultMapping = docMapping
	err = indexMapping.AddCustomAnalyzer("lowercaseKeyword",
		map[string]any{
			"type":      custom.Name,
			"tokenizer": single.Name,
			"token_filters": []string{
				lowercase.Name,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	err = indexMapping.AddCustomAnalyzer("fulltext",
		map[string]any{
			"type":      custom.Name,
			"tokenizer": unicode.Name,
			"token_filters": []string{
				lowercase.Name,
				porter.Name,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return indexMapping, nil
}

func searchResourceByID(id string, index bleve.Index) (*search.Resource, error) {
	req := bleve.NewSearchRequest(bleve.NewDocIDQuery([]string{id}))
	req.Fields = []string{"*"}
	res, err := index.Search(req)
	if err != nil {
		return nil, err
	}
	if res.Hits.Len() == 0 {
		return nil, errors.New("entity not found")
	}

	return matchToResource(res.Hits[0]), nil
}

func searchResourcesByPath(rootId, lookupPath string, index bleve.Index) ([]*search.Resource, error) {
	q := bleve.NewConjunctionQuery(
		bleve.NewQueryStringQuery("RootID:"+rootId),
		bleve.NewQueryStringQuery("Path:"+escapeQuery(lookupPath+"/*")),
	)
	bleveReq := bleve.NewSearchRequest(q)
	bleveReq.Size = math.MaxInt
	bleveReq.Fields = []string{"*"}
	res, err := index.Search(bleveReq)
	if err != nil {
		return nil, err
	}

	resources := make([]*search.Resource, 0, res.Hits.Len())
	for _, match := range res.Hits {
		resources = append(resources, matchToResource(match))
	}

	return resources, nil
}

func searchAndUpdateResourcesDeletionState(id string, state bool, index bleve.Index) ([]*search.Resource, error) {
	rootResource, err := searchResourceByID(id, index)
	if err != nil {
		return nil, err
	}
	rootResource.Deleted = state

	resources := []*search.Resource{rootResource}

	if rootResource.Type == uint64(storageProvider.ResourceType_RESOURCE_TYPE_CONTAINER) {
		descendantResources, err := searchResourcesByPath(rootResource.RootID, rootResource.Path, index)
		if err != nil {
			return nil, err
		}

		for _, descendantResource := range descendantResources {
			descendantResource.Deleted = state
			resources = append(resources, descendantResource)
		}
	}

	return resources, nil
}
