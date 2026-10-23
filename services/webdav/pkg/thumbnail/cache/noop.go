package cache

// NoopCache always returns a cache miss and discards puts.
type NoopCache struct{}

func (NoopCache) Get(_ string) ([]byte, error) {
	return nil, ErrCacheMiss
}

func (NoopCache) Put(_ string, _ []byte) error {
	return nil
}
