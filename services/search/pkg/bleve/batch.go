package bleve

import (
	"errors"
	"path"
	"strings"

	"github.com/blevesearch/bleve/v2"
	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
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
		return b.indexResource(id, r)
	})
}

// indexResource prepares r for bleve (resolving json tags and splicing in
// type-specific adaptations via the mapping package) and appends it to the
// batch under id.
func (b *Batch) indexResource(id string, r search.Resource) error {
	doc, err := mapping.PrepareForIndex(r, r.SearchFieldOverrides())
	if err != nil {
		return err
	}
	return b.batch.Index(id, doc)
}

func (b *Batch) Move(id, parentID, location string) error {
	return b.withSizeLimit(func() error {
		nextPath := utils.MakeRelativePath(location)
		var currentPath string
		return b.forSelfAndDescendants(id, func(resource *search.Resource) error {
			if resource.ID == id {
				currentPath = resource.Path
				resource.Path = nextPath
				resource.Name = path.Base(nextPath)
				resource.ParentID = parentID
			} else {
				resource.Path = strings.Replace(resource.Path, currentPath, nextPath, 1)
			}
			resource.Hidden = search.IsHidden(resource.Path)
			return b.indexResource(resource.ID, *resource)
		})
	})
}

func (b *Batch) Delete(id string) error {
	return b.withSizeLimit(func() error {
		return b.setDeleted(id, true)
	})
}

func (b *Batch) Restore(id string) error {
	return b.withSizeLimit(func() error {
		return b.setDeleted(id, false)
	})
}

func (b *Batch) setDeleted(id string, deleted bool) error {
	return b.forSelfAndDescendants(id, func(resource *search.Resource) error {
		resource.Deleted = deleted
		return b.indexResource(resource.ID, *resource)
	})
}

func (b *Batch) Purge(id string, onlyDeleted bool) error {
	return b.withSizeLimit(func() error {
		return b.forSelfAndDescendants(id, func(resource *search.Resource) error {
			if onlyDeleted && !resource.Deleted {
				return nil
			}
			b.batch.Delete(resource.ID)
			return nil
		})
	})
}

// fn sees the root first; the root's original path drives the descendant lookup
func (b *Batch) forSelfAndDescendants(id string, fn func(*search.Resource) error) error {
	root, err := searchResourceByID(id, b.index)
	if err != nil {
		return err
	}
	rootID, rootPath := root.RootID, root.Path
	isContainer := root.Type == uint64(storageProvider.ResourceType_RESOURCE_TYPE_CONTAINER)

	apply := func(resource *search.Resource) error {
		if err := fn(resource); err != nil {
			return err
		}
		if b.batch.Size() >= b.size {
			return b.Push()
		}
		return nil
	}

	if err := apply(root); err != nil {
		return err
	}
	if !isContainer {
		return nil
	}
	return forEachResourceByPath(rootID, rootPath, b.index, func(resource *search.Resource) error {
		if resource.ID == id {
			return nil
		}
		return apply(resource)
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
