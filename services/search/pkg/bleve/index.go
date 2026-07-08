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

	searchmapping "github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// bolt_timeout makes a second process on the same datapath fail after 5s
// instead of blocking forever on the file lock.
var openRuntimeConfig = map[string]interface{}{"bolt_timeout": "5s"}

// NewIndex opens (or creates) the bleve index at root and classifies the
// stored schema against the one generated from code. On a breaking change it
// refuses with ErrManualActionRequired. On an additive one the code schema is
// persisted into the index (the bleve analogue of an OpenSearch PUT _mapping),
// so the new fields are properly typed from now on and later startups classify
// equal; the caller must still warn that documents indexed before the upgrade
// lack the Classification.NewFields until they are re-indexed.
func NewIndex(root string) (bleve.Index, searchmapping.Classification, error) {
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

		return index, searchmapping.Classification{Verdict: searchmapping.VerdictEqual}, nil
	}
	if err != nil {
		return nil, searchmapping.Classification{}, err
	}

	classification, codeB, err := classifyStoredMapping(index)
	if err != nil {
		_ = index.Close()
		return nil, searchmapping.Classification{}, err
	}

	switch classification.Verdict {
	case searchmapping.VerdictBreaking:
		_ = index.Close()
		return nil, classification, searchmapping.ManualActionRequiredError(destination, classification.Reasons)
	case searchmapping.VerdictAdditive:
		// The classifier guarantees everything else is identical and the new
		// fields hold no data yet, so storing the code mapping only adds
		// fields. Reopen so the live mapping picks it up: without this the
		// new fields would be indexed dynamically and the data-aware rule
		// would turn them breaking on the next startup.
		if err := index.SetInternal([]byte("_mapping"), codeB); err != nil {
			_ = index.Close()
			return nil, searchmapping.Classification{}, fmt.Errorf("failed to store the updated index mapping: %w", err)
		}
		if err := index.Close(); err != nil {
			return nil, searchmapping.Classification{}, err
		}
		index, err = bleve.OpenUsing(destination, openRuntimeConfig)
		if err != nil {
			return nil, searchmapping.Classification{}, err
		}
	}

	return index, classification, nil
}

// classifyStoredMapping diffs the mapping stored in the index against
// NewMapping() and also returns the marshaled code mapping. Fields that are
// new in the code schema but already have data in the index (previously
// indexed dynamically) are breaking, their de-facto form is unknown. The JSON
// compare is stable within one bleve version; if a bleve upgrade changes
// marshaling defaults it fails towards breaking, normalize the affected key
// here if that ever fires.
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
