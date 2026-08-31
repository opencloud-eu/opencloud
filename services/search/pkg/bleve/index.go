package bleve

import (
	"errors"
	"math"
	"path/filepath"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	regexpCharFilter "github.com/blevesearch/bleve/v2/analysis/char/regexp"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/single"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

const (
	wildcardSuffix = ".wildcard"
	indexVersion   = "v2"
)

func indexPath(root string) string {
	return filepath.Join(root, "bleve-"+indexVersion)
}

func NewIndex(root string) (bleve.Index, error) {
	destination := indexPath(root)
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
	words := func() *mapping.FieldMapping {
		fm := bleve.NewTextFieldMapping()
		fm.Analyzer = "lowercaseWords"

		return fm
	}

	whole := func(field string) *mapping.FieldMapping {
		fm := bleve.NewTextFieldMapping()
		fm.Analyzer = "lowercaseKeyword"
		fm.IncludeInAll = false
		fm.Name = field + wildcardSuffix

		return fm
	}

	lowercaseMapping := bleve.NewTextFieldMapping()
	lowercaseMapping.IncludeInAll = false
	lowercaseMapping.Analyzer = "lowercaseKeyword"

	contentMapping := words()
	contentMapping.IncludeInAll = false

	docMapping := bleve.NewDocumentMapping()
	docMapping.AddFieldMappingsAt("Name",
		words(),
		whole("Name"),
	)
	docMapping.AddFieldMappingsAt("Title",
		words(),
		whole("Title"),
	)
	docMapping.AddFieldMappingsAt("Tags", lowercaseMapping)
	docMapping.AddFieldMappingsAt("Favorites", lowercaseMapping)
	docMapping.AddFieldMappingsAt("Content", contentMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultAnalyzer = keyword.Name
	indexMapping.DefaultMapping = docMapping
	err := indexMapping.AddCustomCharFilter("dotToSpace",
		map[string]any{
			"type":    regexpCharFilter.Name,
			"regexp":  `\.`,
			"replace": " ",
		},
	)
	if err != nil {
		return nil, err
	}

	err = indexMapping.AddCustomAnalyzer("lowercaseWords",
		map[string]any{
			"type":         custom.Name,
			"char_filters": []string{"dotToSpace"},
			"tokenizer":    unicode.Name,
			"token_filters": []string{
				lowercase.Name,
			},
		},
	)
	if err != nil {
		return nil, err
	}

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
