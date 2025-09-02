package bleve

import (
	"errors"

	"github.com/blevesearch/bleve/v2"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var _ search.BatchOperator = &Batch{} // ensure Batch implements BatchOperator

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
		affectedResources, err := searchAndUpdateResourcesLocation(id, parentID, location, b.index)
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

func (b *Batch) Purge(id string) error {
	return b.withSizeLimit(func() error {
		b.batch.Delete(id)
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
