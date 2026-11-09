package cache

import (
	"bytes"
	"context"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"sync"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3CacheConfig holds the configuration for an S3 cache backend.
type S3CacheConfig struct {
	Bucket      string
	Region      string
	Endpoint    string
	Secure      bool
	Credentials *credentials.Credentials
}

// IsComplete returns true if all required fields are set.
func (c S3CacheConfig) IsComplete() bool {
	return c.Bucket != "" && c.Region != "" && c.Credentials != nil
}

// s3Object is the subset of minio.Object used by S3Cache.
type s3Object interface {
	io.ReadCloser
}

// s3Client is the subset of minio.Client used by S3Cache.
type s3Client interface {
	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (s3Object, error)
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (info minio.UploadInfo, err error)
}

// realS3Client wraps *minio.Client to satisfy s3Client interface.
type realS3Client struct {
	*minio.Client
}

func (r *realS3Client) GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (s3Object, error) {
	return r.Client.GetObject(ctx, bucketName, objectName, opts)
}

// S3Cache stores thumbnails in an S3-compatible bucket.
type S3Cache struct {
	config  S3CacheConfig
	client  s3Client
	once    sync.Once
	initErr error
	newFunc func(cfg S3CacheConfig) (s3Client, error)
}

func NewS3Cache(cfg S3CacheConfig) *S3Cache {
	c := &S3Cache{config: cfg}
	c.newFunc = func(cfg S3CacheConfig) (s3Client, error) {
		cl, err := minio.New(cfg.Endpoint, &minio.Options{
			Creds:  cfg.Credentials,
			Secure: cfg.Secure,
			Region: cfg.Region,
		})
		if err != nil {
			return nil, err
		}
		return &realS3Client{Client: cl}, nil
	}
	return c
}

// initClient lazily initializes the S3 client on first access.
func (c *S3Cache) initClient() error {
	c.once.Do(func() {
		cl, err := c.newFunc(c.config)
		if err != nil {
			c.initErr = err
			return
		}
		c.client = cl
	})
	return c.initErr
}

func (c *S3Cache) Get(key string) ([]byte, error) {
	if err := c.initClient(); err != nil {
		return nil, err
	}

	obj, err := c.client.GetObject(context.Background(), c.config.Bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, asCacheError(err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, asCacheError(err)
	}

	return data, nil
}

// asCacheError reports a missing object as a cache miss. minio's GetObject only
// validates its arguments and sends the request on the first read, so a key that
// is not there surfaces from the read, not from GetObject. A missing bucket
// answers 404 as well and must not read as a miss: it is a misconfiguration that
// would otherwise hide behind every lookup.
func asCacheError(err error) error {
	rsp := minio.ToErrorResponse(err)
	if rsp.Code == minio.NoSuchKey {
		return ErrCacheMiss
	}
	if rsp.Code == "" && rsp.StatusCode == http.StatusNotFound {
		return ErrCacheMiss
	}

	return err
}

func (c *S3Cache) Put(key string, data []byte) error {
	if err := c.initClient(); err != nil {
		return err
	}

	ext := filepath.Ext(key)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := c.client.PutObject(
		context.Background(),
		c.config.Bucket,
		key,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	return err
}
