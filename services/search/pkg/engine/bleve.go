package engine

import (
	"context"
	"errors"
	"math"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/token/porter"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/single"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"google.golang.org/protobuf/types/known/timestamppb"

	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	searchQuery "github.com/opencloud-eu/opencloud/services/search/pkg/query"
)

// Bleve represents a search engine which utilizes bleve to search and store resources.
type Bleve struct {
	index        bleve.Index
	queryCreator searchQuery.Creator[query.Query]
	log          log.Logger
}

type batch struct {
	engine *Bleve

	batch     *bleve.Batch
	batchSize int
	log       log.Logger
}

func (b *batch) Upsert(id string, r Resource) error {
	return b.engine.doUpsert(id, r, b)
}

func (b *batch) Move(id string, parentid string, target string) error {
	return b.engine.doMove(id, parentid, target, b)
}

func (b *batch) Delete(id string) error {
	return b.engine.setDeleted(id, true, b)
}

func (b *batch) Restore(id string) error {
	return b.engine.setDeleted(id, false, b)
}

func (b *batch) Purge(id string) error {
	return b.engine.doPurge(id, b)
}

func (b *batch) End() error {
	if b.batch == nil {
		return errors.New("no batch started")
	}

	b.log.Debug().Int("size", b.batch.Size()).Msg("Ending batch")
	if err := b.engine.index.Batch(b.batch); err != nil {
		return err
	}

	return nil
}

// NewBleveIndex returns a new bleve index
// given path must exist.
func NewBleveIndex(root string) (bleve.Index, error) {
	destination := filepath.Join(root, "bleve")
	index, err := bleve.Open(destination)
	if errors.Is(bleve.ErrorIndexPathDoesNotExist, err) {
		m, err := BuildBleveMapping()
		if err != nil {
			return nil, err
		}
		index, err = bleve.New(destination, m)
		if err != nil {
			return nil, err
		}

		return index, nil
	}

	return index, err
}

// NewBleveEngine creates a new Bleve instance
func NewBleveEngine(index bleve.Index, queryCreator searchQuery.Creator[query.Query], log log.Logger) *Bleve {
	return &Bleve{
		index:        index,
		queryCreator: queryCreator,
		log:          log,
	}
}

// BuildBleveMapping builds a bleve index mapping which can be used for indexing
func BuildBleveMapping() (mapping.IndexMapping, error) {
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

// Search executes a search request operation within the index.
// Returns a SearchIndexResponse object or an error.
func (b *Bleve) Search(ctx context.Context, sir *searchService.SearchIndexRequest) (*searchService.SearchIndexResponse, error) {
	createdQuery, err := b.queryCreator.Create(sir.Query)
	if err != nil {
		if searchQuery.IsValidationError(err) {
			return nil, errtypes.BadRequest(err.Error())
		}
		return nil, err
	}

	q := bleve.NewConjunctionQuery(
		// Skip documents that have been marked as deleted
		&query.BoolFieldQuery{
			Bool:     false,
			FieldVal: "Deleted",
		},
		createdQuery,
	)

	if sir.Ref != nil {
		q.Conjuncts = append(
			q.Conjuncts,
			&query.TermQuery{
				FieldVal: "RootID",
				Term: storagespace.FormatResourceID(
					&storageProvider.ResourceId{
						StorageId: sir.Ref.GetResourceId().GetStorageId(),
						SpaceId:   sir.Ref.GetResourceId().GetSpaceId(),
						OpaqueId:  sir.Ref.GetResourceId().GetOpaqueId(),
					},
				),
			},
		)
	}

	bleveReq := bleve.NewSearchRequest(q)
	bleveReq.Highlight = bleve.NewHighlight()

	switch {
	case sir.PageSize == -1:
		bleveReq.Size = math.MaxInt
	case sir.PageSize == 0:
		bleveReq.Size = 200
	default:
		bleveReq.Size = int(sir.PageSize)
	}

	bleveReq.Fields = []string{"*"}
	res, err := b.index.Search(bleveReq)
	if err != nil {
		return nil, err
	}

	matches := make([]*searchMessage.Match, 0, len(res.Hits))
	totalMatches := res.Total
	for _, hit := range res.Hits {
		if sir.Ref != nil {
			hitPath := strings.TrimSuffix(getFieldValue[string](hit.Fields, "Path"), "/")
			requestedPath := utils.MakeRelativePath(sir.Ref.Path)
			isRoot := hitPath == requestedPath

			if !isRoot && requestedPath != "." && !strings.HasPrefix(hitPath, requestedPath+"/") {
				totalMatches--
				continue
			}
		}

		rootID, err := storagespace.ParseID(getFieldValue[string](hit.Fields, "RootID"))
		if err != nil {
			return nil, err
		}

		rID, err := storagespace.ParseID(getFieldValue[string](hit.Fields, "ID"))
		if err != nil {
			return nil, err
		}

		pID, _ := storagespace.ParseID(getFieldValue[string](hit.Fields, "ParentID"))
		match := &searchMessage.Match{
			Score: float32(hit.Score),
			Entity: &searchMessage.Entity{
				Ref: &searchMessage.Reference{
					ResourceId: resourceIDtoSearchID(rootID),
					Path:       getFieldValue[string](hit.Fields, "Path"),
				},
				Id:         resourceIDtoSearchID(rID),
				Name:       getFieldValue[string](hit.Fields, "Name"),
				ParentId:   resourceIDtoSearchID(pID),
				Size:       uint64(getFieldValue[float64](hit.Fields, "Size")),
				Type:       uint64(getFieldValue[float64](hit.Fields, "Type")),
				MimeType:   getFieldValue[string](hit.Fields, "MimeType"),
				Deleted:    getFieldValue[bool](hit.Fields, "Deleted"),
				Tags:       getFieldSliceValue[string](hit.Fields, "Tags"),
				Highlights: getFragmentValue(hit.Fragments, "Content", 0),
				Audio:      getAudioValue[searchMessage.Audio](hit.Fields),
				Image:      getImageValue[searchMessage.Image](hit.Fields),
				Location:   getLocationValue[searchMessage.GeoCoordinates](hit.Fields),
				Photo:      getPhotoValue[searchMessage.Photo](hit.Fields),
			},
		}

		if mtime, err := time.Parse(time.RFC3339, getFieldValue[string](hit.Fields, "Mtime")); err == nil {
			match.Entity.LastModifiedTime = &timestamppb.Timestamp{Seconds: mtime.Unix(), Nanos: int32(mtime.Nanosecond())}
		}

		matches = append(matches, match)
	}

	return &searchService.SearchIndexResponse{
		Matches:      matches,
		TotalMatches: int32(totalMatches),
	}, nil
}

func (b *Bleve) StartBatch(batchSize int) (Batch, error) {
	if batchSize <= 0 {
		return nil, errors.New("batch size must be greater than 0")
	}

	return &batch{
		engine:    b,
		batch:     b.index.NewBatch(),
		batchSize: batchSize,
	}, nil
}

// Upsert indexes or stores Resource data fields.
func (b *Bleve) Upsert(id string, r Resource) error {
	return b.doUpsert(id, r, nil)
}

func (b *Bleve) doUpsert(id string, r Resource, batch *batch) error {
	if batch != nil {
		if err := batch.batch.Index(id, r); err != nil {
			return err
		}
		if batch.batch.Size() >= batch.batchSize {
			b.log.Debug().Int("size", batch.batch.Size()).Msg("Committing batch")
			if err := b.index.Batch(batch.batch); err != nil {
				return err
			}
			batch.batch = b.index.NewBatch()
		}
		return nil
	}
	return b.index.Index(id, r)
}

// Move updates the resource location and all of its necessary fields.
func (b *Bleve) Move(id string, parentid string, target string) error {
	return b.doMove(id, parentid, target, nil)
}

func (b *Bleve) doMove(id string, parentid string, target string, batch *batch) error {
	r, err := b.getResource(id)
	if err != nil {
		return err
	}
	currentPath := r.Path
	nextPath := utils.MakeRelativePath(target)

	r, err = b.updateEntity(id, func(r *Resource) {
		r.Path = nextPath
		r.Name = path.Base(nextPath)
		r.ParentID = parentid
	}, batch)
	if err != nil {
		return err
	}

	if r.Type == uint64(storageProvider.ResourceType_RESOURCE_TYPE_CONTAINER) {
		q := bleve.NewConjunctionQuery(
			bleve.NewQueryStringQuery("RootID:"+r.RootID),
			bleve.NewQueryStringQuery("Path:"+escapeQuery(currentPath+"/*")),
		)
		bleveReq := bleve.NewSearchRequest(q)
		bleveReq.Size = math.MaxInt
		bleveReq.Fields = []string{"*"}
		res, err := b.index.Search(bleveReq)
		if err != nil {
			return err
		}

		for _, h := range res.Hits {
			_, err := b.updateEntity(h.ID, func(r *Resource) {
				r.Path = strings.Replace(r.Path, currentPath, nextPath, 1)
			}, batch)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Delete marks the resource as deleted.
// The resource object will stay in the bleve index,
// instead of removing the resource it just marks it as deleted!
// can be undone
func (b *Bleve) Delete(id string) error {
	return b.setDeleted(id, true, nil)
}

// Restore is the counterpart to Delete.
// It restores the resource which makes it available again.
func (b *Bleve) Restore(id string) error {
	return b.setDeleted(id, false, nil)
}

// Purge removes a resource from the index, irreversible operation.
func (b *Bleve) Purge(id string) error {
	return b.doPurge(id, nil)
}

func (b *Bleve) doPurge(id string, batch *batch) error {
	if batch != nil {
		err := batch.Delete(id)
		if err != nil {
			return err
		}
		if batch.batch.Size() >= batch.batchSize {
			if err := b.index.Batch(batch.batch); err != nil {
				return err
			}
			batch.batch = b.index.NewBatch()
		}
		return nil
	}
	return b.index.Delete(id)
}

// DocCount returns the number of resources in the index.
func (b *Bleve) DocCount() (uint64, error) {
	return b.index.DocCount()
}

func (b *Bleve) getResource(id string) (*Resource, error) {
	req := bleve.NewSearchRequest(bleve.NewDocIDQuery([]string{id}))
	req.Fields = []string{"*"}
	res, err := b.index.Search(req)
	if err != nil {
		return nil, err
	}
	if res.Hits.Len() == 0 {
		return nil, errors.New("entity not found")
	}

	fields := res.Hits[0].Fields

	return &Resource{
		ID:       getFieldValue[string](fields, "ID"),
		RootID:   getFieldValue[string](fields, "RootID"),
		Path:     getFieldValue[string](fields, "Path"),
		ParentID: getFieldValue[string](fields, "ParentID"),
		Type:     uint64(getFieldValue[float64](fields, "Type")),
		Deleted:  getFieldValue[bool](fields, "Deleted"),
		Document: content.Document{
			Name:     getFieldValue[string](fields, "Name"),
			Title:    getFieldValue[string](fields, "Title"),
			Size:     uint64(getFieldValue[float64](fields, "Size")),
			Mtime:    getFieldValue[string](fields, "Mtime"),
			MimeType: getFieldValue[string](fields, "MimeType"),
			Content:  getFieldValue[string](fields, "Content"),
			Tags:     getFieldSliceValue[string](fields, "Tags"),
			Audio:    getAudioValue[libregraph.Audio](fields),
			Image:    getImageValue[libregraph.Image](fields),
			Location: getLocationValue[libregraph.GeoCoordinates](fields),
			Photo:    getPhotoValue[libregraph.Photo](fields),
		},
	}, nil
}

func newPointerOfType[T any]() *T {
	t := reflect.TypeOf((*T)(nil)).Elem()
	ptr := reflect.New(t).Interface()
	return ptr.(*T)
}

func getFieldName(structField reflect.StructField) string {
	tag := structField.Tag.Get("json")
	if tag == "" {
		return structField.Name
	}

	return strings.Split(tag, ",")[0]
}

func unmarshalInterfaceMap(out any, flatMap map[string]interface{}, prefix string) bool {
	nonEmpty := false
	obj := reflect.ValueOf(out).Elem()
	for i := 0; i < obj.NumField(); i++ {
		field := obj.Field(i)
		structField := obj.Type().Field(i)
		mapKey := prefix + getFieldName(structField)

		if value, ok := flatMap[mapKey]; ok {
			if field.Kind() == reflect.Ptr {
				alloc := reflect.New(field.Type().Elem())
				elemType := field.Type().Elem()

				// convert time strings from index for search requests
				if elemType == reflect.TypeOf(timestamppb.Timestamp{}) {
					if strValue, ok := value.(string); ok {
						if parsedTime, err := time.Parse(time.RFC3339, strValue); err == nil {
							alloc.Elem().Set(reflect.ValueOf(*timestamppb.New(parsedTime)))
							field.Set(alloc)
							nonEmpty = true
						}
					}
					continue
				}

				// convert time strings from index for libregraph structs when updating resources
				if elemType == reflect.TypeOf(time.Time{}) {
					if strValue, ok := value.(string); ok {
						if parsedTime, err := time.Parse(time.RFC3339, strValue); err == nil {
							alloc.Elem().Set(reflect.ValueOf(parsedTime))
							field.Set(alloc)
							nonEmpty = true
						}
					}
					continue
				}

				alloc.Elem().Set(reflect.ValueOf(value).Convert(elemType))
				field.Set(alloc)
				nonEmpty = true
			}
		}
	}

	return nonEmpty
}

func getAudioValue[T any](fields map[string]interface{}) *T {
	if !strings.HasPrefix(getFieldValue[string](fields, "MimeType"), "audio/") {
		return nil
	}

	var audio = newPointerOfType[T]()
	if ok := unmarshalInterfaceMap(audio, fields, "audio."); ok {
		return audio
	}

	return nil
}

func getImageValue[T any](fields map[string]interface{}) *T {
	var image = newPointerOfType[T]()
	if ok := unmarshalInterfaceMap(image, fields, "image."); ok {
		return image
	}

	return nil
}

func getLocationValue[T any](fields map[string]interface{}) *T {
	var location = newPointerOfType[T]()
	if ok := unmarshalInterfaceMap(location, fields, "location."); ok {
		return location
	}

	return nil
}

func getPhotoValue[T any](fields map[string]interface{}) *T {
	var photo = newPointerOfType[T]()
	if ok := unmarshalInterfaceMap(photo, fields, "photo."); ok {
		return photo
	}

	return nil
}

func (b *Bleve) updateEntity(id string, mutateFunc func(r *Resource), batch *batch) (*Resource, error) {
	it, err := b.getResource(id)
	if err != nil {
		return nil, err
	}

	mutateFunc(it)

	return it, b.doUpsert(id, *it, batch)
}

func (b *Bleve) setDeleted(id string, deleted bool, batch *batch) error {
	it, err := b.updateEntity(id, func(r *Resource) {
		r.Deleted = deleted
	}, batch)
	if err != nil {
		return err
	}

	if it.Type == uint64(storageProvider.ResourceType_RESOURCE_TYPE_CONTAINER) {
		q := bleve.NewConjunctionQuery(
			bleve.NewQueryStringQuery("RootID:"+it.RootID),
			bleve.NewQueryStringQuery("Path:"+escapeQuery(it.Path+"/*")),
		)

		bleveReq := bleve.NewSearchRequest(q)
		bleveReq.Size = math.MaxInt
		bleveReq.Fields = []string{"*"}
		res, err := b.index.Search(bleveReq)
		if err != nil {
			return err
		}

		for _, h := range res.Hits {
			_, err := b.updateEntity(h.ID, func(r *Resource) {
				r.Deleted = deleted
			}, batch)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
