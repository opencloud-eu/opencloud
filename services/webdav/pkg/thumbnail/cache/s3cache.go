package cache

import (
	"bytes"
	"context"
	"io"
	"mime"
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
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" || errResp.StatusCode == 404 {
			return nil, ErrCacheMiss
		}
		return nil, err
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, err
	}

	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
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
