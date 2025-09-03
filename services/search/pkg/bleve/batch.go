package bleve

import (
	"errors"
	"path"
	"strings"

	"github.com/blevesearch/bleve/v2"
	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var _ search.BatchOperator = (*Batch)(nil) // ensure Batch implements BatchOperator

type Batch struct {
	batch *bleve.Batch
	index bleve.Index
	size  int
	log   log.Logger
}

func NewBatch(index bleve.Index, size int) (*Batch, error) {
	if size <= 0 {
		return nil, errors.New("batch size must be greater than 0")
	}

	return &Batch{
		batch: index.NewBatch(),
		index: index,
		size:  size,
	}, nil
}

func (b *Batch) Upsert(id string, r search.Resource) error {
	return b.withSizeLimit(func() error {
		return b.batch.Index(id, r)
	})
}

func (b *Batch) Move(id, parentID, location string) error {
	return b.withSizeLimit(func() error {
		rootResource, err := searchResourceByID(id, b.index)
		if err != nil {
			return err
		}
		currentPath := rootResource.Path
		nextPath := utils.MakeRelativePath(location)

		rootResource.Path = nextPath
		rootResource.Name = path.Base(nextPath)
		rootResource.ParentID = parentID

		resources := []*search.Resource{rootResource}

		if rootResource.Type == uint64(storageProvider.ResourceType_RESOURCE_TYPE_CONTAINER) {
			descendantResources, err := searchResourcesByPath(rootResource.RootID, currentPath, b.index)
			if err != nil {
				return err
			}

			for _, descendantResource := range descendantResources {
				descendantResource.Path = strings.Replace(descendantResource.Path, currentPath, nextPath, 1)
				resources = append(resources, descendantResource)
			}
		}

		for _, resource := range resources {
			if err := b.batch.Index(resource.ID, resource); err != nil {
				return err
			}
		}

		return nil
	})
}

func (b *Batch) Delete(id string) error {
	return b.withSizeLimit(func() error {
		affectedResources, err := searchAndUpdateResourcesDeletionState(id, true, b.index)
		if err != nil {
			return err
		}

		for _, resource := range affectedResources {
			if err := b.batch.Index(resource.ID, resource); err != nil {
				return err
			}
		}

		return nil
	})
}

func (b *Batch) Restore(id string) error {
	return b.withSizeLimit(func() error {
		affectedResources, err := searchAndUpdateResourcesDeletionState(id, false, b.index)
		if err != nil {
			return err
		}

		for _, resource := range affectedResources {
			if err := b.batch.Index(resource.ID, resource); err != nil {
				return err
			}
		}

		return nil
	})
}

func (b *Batch) Purge(id string, onlyDeleted bool) error {
	return b.withSizeLimit(func() error {
		rootResource, err := searchResourceByID(id, b.index)
		if err != nil {
			return err
		}

		var affectResources []*search.Resource
		add := func(resource *search.Resource) {
			if onlyDeleted && !resource.Deleted {
				return
			}

			affectResources = append(affectResources, resource)
		}

		add(rootResource)

		if rootResource.Type == uint64(storageProvider.ResourceType_RESOURCE_TYPE_CONTAINER) {
			descendantResources, err := searchResourcesByPath(rootResource.RootID, rootResource.Path, b.index)
			if err != nil {
				return err
			}

			for _, descendantResource := range descendantResources {
				add(descendantResource)
			}
		}

		for _, resource := range affectResources {
			b.batch.Delete(resource.ID)
		}

		return nil
	})
}

func (b *Batch) Push() error {
	if b.batch.Size() == 0 {
		return nil
	}

	if err := b.index.Batch(b.batch); err != nil {
		return err
	}

	b.batch.Reset()

	return nil
}

func (b *Batch) withSizeLimit(f func() error) error {
	if err := f(); err != nil {
		return err
	}

	if b.batch.Size() >= b.size {
		return b.Push()
	}

	return nil
}
