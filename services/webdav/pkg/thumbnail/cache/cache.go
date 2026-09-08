package cache

import (
	"errors"
	"path/filepath"

	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
)

var ErrCacheMiss = errors.New("thumbnail cache miss")

// DefaultFileCacheDir is the on-disk location of file-backend thumbnails,
// mirroring main's $OC_BASE_DATA_PATH/thumbnails/files layout.
func DefaultFileCacheDir() string {
	return filepath.Join(defaults.BaseDataPath(), "thumbnails", "files")
}

// ThumbnailCache provides cached storage for generated thumbnails.
type ThumbnailCache interface {
	Get(key string) ([]byte, error)
	Put(key string, data []byte) error
}

// NewThumbnailCache creates a cache instance based on the configured backend.
func NewThumbnailCache(backend string, cacheDir string, s3cfg *S3CacheConfig) ThumbnailCache {
	switch backend {
	case "none":
		return NoopCache{}
	case "memory":
		return NewInMemoryCache()
	case "file":
		dir := cacheDir
		if dir == "" {
			dir = DefaultFileCacheDir()
		}
		return NewFileCache(dir)
	case "s3":
		if s3cfg != nil && s3cfg.IsComplete() {
			return NewS3Cache(*s3cfg)
		}
		return NoopCache{}
	default:
		return NoopCache{}
	}
}

// BuildS3CacheConfig creates an S3CacheConfig from raw config fields.
func BuildS3CacheConfig(bucket, region, endpoint, accessKey, secretKey string) *S3CacheConfig {
	cfg := &S3CacheConfig{
		Bucket:   bucket,
		Region:   region,
		Endpoint: endpoint,
		Secure:   true,
	}
	if accessKey != "" && secretKey != "" {
		cfg.Credentials = credentials.NewStaticV4(accessKey, secretKey, "")
	}
	return cfg
}
