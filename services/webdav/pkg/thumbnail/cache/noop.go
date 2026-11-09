package cache

// NoopCache always returns a cache miss and discards puts.
type NoopCache struct{}

func (NoopCache) Get(_ string) (Image, error) {
	return Image{}, ErrCacheMiss
}

func (NoopCache) Put(_ string, _ Image) error {
	return nil
}
