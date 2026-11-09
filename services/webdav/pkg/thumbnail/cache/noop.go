package cache

import "github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"

// NoopCache always returns a cache miss and discards puts.
type NoopCache struct{}

func (NoopCache) Get(_ string) (thumbnail.Image, error) {
	return thumbnail.Image{}, ErrCacheMiss
}

func (NoopCache) Put(_ string, _ thumbnail.Image) error {
	return nil
}
