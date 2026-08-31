package bleve

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"reflect"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/char/regexp"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	searchmapping "github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func NewIndex(root string) (bleve.Index, error) {
	destination := filepath.Join(root, fmt.Sprintf("bleve-v%d", search.SchemaVersion))
	index, err := bleve.Open(destination)
	if errors.Is(err, bleve.ErrorIndexPathDoesNotExist) {
		indexMapping, err := NewMapping()
		if err != nil {
			return nil, err
		}
		index, err = bleve.New(destination, indexMapping)
		if err != nil {
			return nil, err
		}

		return index, nil
	}

	return index, err
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
	// words: split into lowercased words, a dot is a word boundary too so that
	// "report" finds "Report.txt"; no stemming, a name is not prose
	err = indexMapping.AddCustomCharFilter("dot_to_space", map[string]any{
		"type":    regexp.Name,
		"regexp":  `\.`,
		"replace": " ",
	})
	if err != nil {
		return nil, err
	}
	err = indexMapping.AddCustomAnalyzer(searchmapping.WordsAnalyzer,
		map[string]any{
			"type":          custom.Name,
			"char_filters":  []string{"dot_to_space"},
			"tokenizer":     unicode.Name,
			"token_filters": []string{lowercase.Name},
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

func searchResourcesByPath(rootID string, lookupPath string, index bleve.Index) ([]*search.Resource, error) {
	q := bleve.NewConjunctionQuery(
		bleve.NewQueryStringQuery("RootID:"+rootID),
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
