package bleve

import (
	"errors"
	"math"
	"path"
	"path/filepath"
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
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func NewIndex(root string) (bleve.Index, error) {
	destination := filepath.Join(root, "bleve")
	index, err := bleve.Open(destination)
	if errors.Is(bleve.ErrorIndexPathDoesNotExist, err) {
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
	nameMapping := bleve.NewTextFieldMapping()
	nameMapping.Analyzer = "lowercaseKeyword"

	lowercaseMapping := bleve.NewTextFieldMapping()
	lowercaseMapping.IncludeInAll = false
	lowercaseMapping.Analyzer = "lowercaseKeyword"

	fulltextFieldMapping := bleve.NewTextFieldMapping()
	fulltextFieldMapping.Analyzer = "fulltext"
	fulltextFieldMapping.IncludeInAll = false

	docMapping := bleve.NewDocumentMapping()
	docMapping.AddFieldMappingsAt("Name", nameMapping)
	docMapping.AddFieldMappingsAt("Tags", lowercaseMapping)
	docMapping.AddFieldMappingsAt("Content", fulltextFieldMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultAnalyzer = keyword.Name
	indexMapping.DefaultMapping = docMapping
	err := indexMapping.AddCustomAnalyzer("lowercaseKeyword",
		map[string]interface{}{
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
		map[string]interface{}{
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

func searchAndUpdateResourcesLocation(rootID, parentID, location string, index bleve.Index) ([]*search.Resource, error) {
	rootResource, err := searchResourceByID(rootID, index)
	if err != nil {
		return nil, err
	}
	currentPath := rootResource.Path
	nextPath := utils.MakeRelativePath(location)

	rootResource.Path = nextPath
	rootResource.Name = path.Base(nextPath)
	rootResource.ParentID = parentID

	resources := []*search.Resource{rootResource}

	if rootResource.Type == uint64(storageProvider.ResourceType_RESOURCE_TYPE_CONTAINER) {
		descendantResources, err := searchResourcesByPath(rootResource.RootID, currentPath, index)
		if err != nil {
			return nil, err
		}

		for _, descendantResource := range descendantResources {
			descendantResource.Path = strings.Replace(descendantResource.Path, currentPath, nextPath, 1)
			resources = append(resources, descendantResource)
		}
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
