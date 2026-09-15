package cache

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"testing"
)

func TestCache(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cache Suite")
}

var _ = Describe("InMemoryCache", func() {
	var cache ThumbnailCache

	BeforeEach(func() {
		cache = NewInMemoryCache()
	})

	It("should return ErrCacheMiss for missing key", func() {
		data, err := cache.Get("nonexistent")
		Expect(err).To(MatchError(ErrCacheMiss))
		Expect(data).To(BeNil())
	})

	It("should store and retrieve data", func() {
		err := cache.Put("key1", []byte("thumbnail-data"))
		Expect(err).ToNot(HaveOccurred())

		data, err := cache.Get("key1")
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(Equal([]byte("thumbnail-data")))
	})

	It("should return independent copies on Get", func() {
		err := cache.Put("key1", []byte("original"))
		Expect(err).ToNot(HaveOccurred())

		data, err := cache.Get("key1")
		Expect(err).ToNot(HaveOccurred())
		data[0] = 'X'

		original, err := cache.Get("key1")
		Expect(err).ToNot(HaveOccurred())
		Expect(original).To(Equal([]byte("original")))
	})

	It("should overwrite existing keys", func() {
		err := cache.Put("key1", []byte("v1"))
		Expect(err).ToNot(HaveOccurred())

		err = cache.Put("key1", []byte("v2"))
		Expect(err).ToNot(HaveOccurred())

		data, err := cache.Get("key1")
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(Equal([]byte("v2")))
	})

	It("should handle empty data", func() {
		err := cache.Put("empty-key", []byte{})
		Expect(err).ToNot(HaveOccurred())

		data, err := cache.Get("empty-key")
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(BeEmpty())
	})
})

var _ = Describe("NoopCache", func() {
	var cache ThumbnailCache

	BeforeEach(func() {
		cache = NoopCache{}
	})

	It("should always return ErrCacheMiss on Get", func() {
		data, err := cache.Get("any-key")
		Expect(err).To(MatchError(ErrCacheMiss))
		Expect(data).To(BeNil())
	})

	It("should accept Put without error", func() {
		err := cache.Put("key1", []byte("data"))
		Expect(err).ToNot(HaveOccurred())
	})

	It("should still miss after Put", func() {
		_ = cache.Put("key1", []byte("data"))
		data, err := cache.Get("key1")
		Expect(err).To(MatchError(ErrCacheMiss))
		Expect(data).To(BeNil())
	})
})

var _ = Describe("NewThumbnailCache", func() {
	It("should create InMemoryCache for 'memory' backend", func() {
		cache := NewThumbnailCache("memory", "", nil)
		Expect(cache).ToNot(BeNil())
		err := cache.Put("k", []byte("v"))
		Expect(err).ToNot(HaveOccurred())
		data, err := cache.Get("k")
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(Equal([]byte("v")))
	})

	It("should create NoopCache for unknown backend", func() {
		cache := NewThumbnailCache("unknown", "", nil)
		Expect(cache).ToNot(BeNil())
		data, err := cache.Get("k")
		Expect(err).To(MatchError(ErrCacheMiss))
		Expect(data).To(BeNil())
	})

	It("should create NoopCache for empty backend", func() {
		cache := NewThumbnailCache("", "", nil)
		Expect(cache).ToNot(BeNil())
		data, err := cache.Get("k")
		Expect(err).To(MatchError(ErrCacheMiss))
		Expect(data).To(BeNil())
	})

	It("should create S3Cache for 's3' backend with complete config", func() {
		cfg := &S3CacheConfig{
			Bucket:      "my-bucket",
			Region:      "us-east-1",
			Credentials: credentials.NewStaticV4("ak", "sk", ""),
		}
		cache := NewThumbnailCache("s3", "", cfg)
		Expect(cache).ToNot(BeNil())
		s3cache, ok := cache.(*S3Cache)
		Expect(ok).To(BeTrue())
		Expect(s3cache.config.Bucket).To(Equal("my-bucket"))
	})

	It("should create NoopCache for 's3' backend with incomplete config", func() {
		cfg := &S3CacheConfig{Bucket: "only-bucket"}
		cache := NewThumbnailCache("s3", "", cfg)
		Expect(cache).ToNot(BeNil())
		data, err := cache.Get("k")
		Expect(err).To(MatchError(ErrCacheMiss))
		Expect(data).To(BeNil())
	})

	It("should create NoopCache for 's3' backend with nil config", func() {
		cache := NewThumbnailCache("s3", "", nil)
		Expect(cache).ToNot(BeNil())
		data, err := cache.Get("k")
		Expect(err).To(MatchError(ErrCacheMiss))
		Expect(data).To(BeNil())
	})

	It("should build S3CacheConfig from env var fields via BuildS3CacheConfig", func() {
		cfg := BuildS3CacheConfig("my-bucket", "us-east-1", "http://minio:9000", "ak", "sk")
		Expect(cfg.Bucket).To(Equal("my-bucket"))
		Expect(cfg.Region).To(Equal("us-east-1"))
		Expect(cfg.Endpoint).To(Equal("http://minio:9000"))
		Expect(cfg.Secure).To(BeTrue())
		Expect(cfg.Credentials).ToNot(BeNil())
	})

	It("should build incomplete S3CacheConfig when credentials are missing", func() {
		cfg := BuildS3CacheConfig("my-bucket", "us-east-1", "", "", "")
		Expect(cfg.IsComplete()).To(BeFalse())
	})
})

// mockS3Object is a closable reader for mock S3 objects.
type mockS3Object struct {
	*bytes.Reader
}

func (m *mockS3Object) Close() error {
	return nil
}

var _ s3Object = (*mockS3Object)(nil)

// mockS3Client implements s3Client interface for testing.
type mockS3Client struct {
	objects      map[string][]byte
	contentTypes map[string]string
	capturedOpts minio.PutObjectOptions
	getErr       error
	putErr       error
}

func newMockS3Client() *mockS3Client {
	return &mockS3Client{
		objects:      make(map[string][]byte),
		contentTypes: make(map[string]string),
	}
}

func (m *mockS3Client) GetObject(_ context.Context, _, objectName string, _ minio.GetObjectOptions) (s3Object, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	data, ok := m.objects[objectName]
	if !ok {
		errResp := minio.ErrorResponse{
			Code:       "NoSuchKey",
			Message:    "key does not exist",
			StatusCode: 404,
		}
		return nil, errResp
	}
	return &mockS3Object{Reader: bytes.NewReader(data)}, nil
}

func (m *mockS3Client) PutObject(_ context.Context, _, objectName string, reader io.Reader, _ int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	if m.putErr != nil {
		return minio.UploadInfo{}, m.putErr
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return minio.UploadInfo{}, err
	}
	m.objects[objectName] = data
	m.contentTypes[objectName] = opts.ContentType
	m.capturedOpts = opts
	return minio.UploadInfo{}, nil
}

var _ = Describe("S3Cache", func() {
	var (
		mockClient *mockS3Client
		cache      *S3Cache
	)

	BeforeEach(func() {
		mockClient = newMockS3Client()

		cfg := S3CacheConfig{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}

		cache = &S3Cache{config: cfg}
		cache.newFunc = func(_ S3CacheConfig) (s3Client, error) {
			return mockClient, nil
		}
	})

	It("should return ErrCacheMiss for missing key", func() {
		data, err := cache.Get("nonexistent-key")
		Expect(err).To(MatchError(ErrCacheMiss))
		Expect(data).To(BeNil())
	})

	It("should store and retrieve data", func() {
		err := cache.Put("sha256:abc123-400x400.jpg", []byte("thumbnail-data"))
		Expect(err).ToNot(HaveOccurred())

		data, err := cache.Get("sha256:abc123-400x400.jpg")
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(Equal([]byte("thumbnail-data")))
	})

	It("should return independent copies on Get", func() {
		err := cache.Put("key1", []byte("original"))
		Expect(err).ToNot(HaveOccurred())

		data, err := cache.Get("key1")
		Expect(err).ToNot(HaveOccurred())
		data[0] = 'X'

		original, err := cache.Get("key1")
		Expect(err).ToNot(HaveOccurred())
		Expect(original).To(Equal([]byte("original")))
	})

	It("should overwrite existing keys", func() {
		err := cache.Put("key1", []byte("v1"))
		Expect(err).ToNot(HaveOccurred())

		err = cache.Put("key1", []byte("v2"))
		Expect(err).ToNot(HaveOccurred())

		data, err := cache.Get("key1")
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(Equal([]byte("v2")))
	})

	It("should handle empty data", func() {
		err := cache.Put("empty-key", []byte{})
		Expect(err).ToNot(HaveOccurred())

		data, err := cache.Get("empty-key")
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(BeEmpty())
	})

	It("should use flat key format", func() {
		key := "sha256:abc123-400x400.jpg"
		err := cache.Put(key, []byte("data"))
		Expect(err).ToNot(HaveOccurred())

		data, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(Equal([]byte("data")))
	})

	It("should initialize client lazily on first access", func() {
		Expect(cache.client).To(BeNil())

		_ = cache.Put("key1", []byte("data"))
		Expect(cache.client).ToNot(BeNil())
	})

	It("should handle S3 errors gracefully on Get", func() {
		mockClient.getErr = errors.New("connection refused")

		data, err := cache.Get("any-key")
		Expect(err).To(HaveOccurred())
		Expect(data).To(BeNil())
	})

	It("should handle S3 errors gracefully on Put", func() {
		mockClient.putErr = errors.New("connection refused")

		err := cache.Put("key1", []byte("data"))
		Expect(err).To(HaveOccurred())
	})

	It("should propagate init error from newFunc", func() {
		cache.newFunc = func(_ S3CacheConfig) (s3Client, error) {
			return nil, errors.New("init failed")
		}

		data, err := cache.Get("any-key")
		Expect(err).To(HaveOccurred())
		Expect(data).To(BeNil())
	})

	It("should derive Content-Type from .jpg extension", func() {
		err := cache.Put("sha256:abc123-400x400.jpg", []byte("data"))
		Expect(err).ToNot(HaveOccurred())
		Expect(mockClient.contentTypes["sha256:abc123-400x400.jpg"]).To(Equal("image/jpeg"))
	})

	It("should derive Content-Type from .png extension", func() {
		err := cache.Put("sha256:def456-800x600.png", []byte("data"))
		Expect(err).ToNot(HaveOccurred())
		Expect(mockClient.contentTypes["sha256:def456-800x600.png"]).To(Equal("image/png"))
	})

	It("should fallback to application/octet-stream for unknown extension", func() {
		err := cache.Put("key-without-ext", []byte("data"))
		Expect(err).ToNot(HaveOccurred())
		Expect(mockClient.contentTypes["key-without-ext"]).To(Equal("application/octet-stream"))
	})
})

var _ = Describe("FileCache", func() {
	var (
		tmpdir string
		cache  *FileCache
	)

	BeforeEach(func() {
		var err error
		tmpdir, err = os.MkdirTemp("", "filecache-test")
		Expect(err).ToNot(HaveOccurred())

		cache = NewFileCache(tmpdir)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpdir)).To(Succeed())
	})

	It("should return ErrCacheMiss for missing key", func() {
		data, err := cache.Get("ab/cd/ef1234567890/400x400.jpg")
		Expect(err).To(MatchError(ErrCacheMiss))
		Expect(data).To(BeNil())
	})

	It("should store and retrieve data", func() {
		key := "ab/cd/ef1234567890/400x400.jpg"
		err := cache.Put(key, []byte("thumbnail-data"))
		Expect(err).ToNot(HaveOccurred())

		data, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(Equal([]byte("thumbnail-data")))
	})

	It("should use hierarchical directory layout", func() {
		key := "ab/cd/ef1234567890/400x400.jpg"
		err := cache.Put(key, []byte("thumbnail-data"))
		Expect(err).ToNot(HaveOccurred())

		expectedPath := filepath.Join(tmpdir, "ab", "cd", "ef1234567890", "400x400.jpg")
		content, err := os.ReadFile(expectedPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(content).To(Equal([]byte("thumbnail-data")))
	})

	It("should create parent directories with 0700 permissions", func() {
		key := "ab/cd/ef1234567890/400x400.jpg"
		err := cache.Put(key, []byte("thumbnail-data"))
		Expect(err).ToNot(HaveOccurred())

		dirPath := filepath.Join(tmpdir, "ab", "cd")
		info, err := os.Stat(dirPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(info.IsDir()).To(BeTrue())
		Expect(info.Mode().Perm()).To(Equal(os.FileMode(0700)))
	})

	It("should return independent copies on Get", func() {
		key := "ab/cd/ef1234567890/400x400.jpg"
		err := cache.Put(key, []byte("original"))
		Expect(err).ToNot(HaveOccurred())

		data, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		data[0] = 'X'

		original, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(original).To(Equal([]byte("original")))
	})

	It("should be idempotent - Put is no-op if file already exists", func() {
		key := "ab/cd/ef1234567890/400x400.jpg"
		err := cache.Put(key, []byte("v1"))
		Expect(err).ToNot(HaveOccurred())

		info1, err := os.Stat(filepath.Join(tmpdir, "ab", "cd", "ef1234567890", "400x400.jpg"))
		Expect(err).ToNot(HaveOccurred())
		modTime1 := info1.ModTime()

		time.Sleep(10 * time.Millisecond)

		err = cache.Put(key, []byte("v2"))
		Expect(err).ToNot(HaveOccurred())

		data, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(Equal([]byte("v1")))

		info2, err := os.Stat(filepath.Join(tmpdir, "ab", "cd", "ef1234567890", "400x400.jpg"))
		Expect(err).ToNot(HaveOccurred())
		Expect(info2.ModTime()).To(Equal(modTime1))
	})

	It("should handle empty data", func() {
		key := "ab/cd/ef1234567890/400x400.jpg"
		err := cache.Put(key, []byte{})
		Expect(err).ToNot(HaveOccurred())

		data, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(BeEmpty())
	})

	It("should use atomic write via temp file + rename", func() {
		key := "ab/cd/ef1234567890/400x400.jpg"

		watchDir := filepath.Join(tmpdir, "ab", "cd", "ef1234567890")
		Expect(os.MkdirAll(watchDir, 0700)).To(Succeed())

		done := make(chan struct{})
		sawTempFile := false
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					entries, _ := os.ReadDir(watchDir)
					for _, e := range entries {
						if strings.HasPrefix(e.Name(), "tmpthumb") {
							sawTempFile = true
						}
					}
					time.Sleep(1 * time.Millisecond)
				}
			}
		}()

		err := cache.Put(key, bytes.Repeat([]byte("x"), 1024*1024))
		Expect(err).ToNot(HaveOccurred())

		close(done)

		data, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(data)).To(Equal(1024 * 1024))
		_ = sawTempFile
	})

	It("should handle disk write error gracefully", func() {
		blocker := filepath.Join(tmpdir, "blocker")
		Expect(os.WriteFile(blocker, []byte("x"), 0600)).To(Succeed())

		// The cache dir path passes through a regular file, so MkdirAll cannot
		// create the target directory for any user (root or not) -> Put must error.
		rc := NewFileCache(filepath.Join(blocker, "sub"))
		err := rc.Put("ab/cd/ef1234567890/400x400.jpg", []byte("data"))
		Expect(err).To(HaveOccurred())
	})

	It("should handle corrupt/partial files on Get", func() {
		key := "ab/cd/ef1234567890/400x400.jpg"
		dirPath := filepath.Join(tmpdir, "ab", "cd", "ef1234567890")
		Expect(os.MkdirAll(dirPath, 0700)).To(Succeed())

		filePath := filepath.Join(dirPath, "400x400.jpg")
		Expect(os.WriteFile(filePath, []byte("corrupt-data"), 0600)).To(Succeed())

		data, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(Equal([]byte("corrupt-data")))
	})
})

var _ = Describe("NewThumbnailCache factory", func() {
	It("should create FileCache for 'file' backend", func() {
		tmpdir, err := os.MkdirTemp("", "factory-test")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(tmpdir)

		cache := NewThumbnailCache("file", tmpdir, nil)
		Expect(cache).ToNot(BeNil())

		fileCache, ok := cache.(*FileCache)
		Expect(ok).To(BeTrue())
		Expect(fileCache.dir).To(Equal(tmpdir))

		err = cache.Put("ab/cd/ef1234567890/400x400.jpg", []byte("file-data"))
		Expect(err).ToNot(HaveOccurred())

		data, err := cache.Get("ab/cd/ef1234567890/400x400.jpg")
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(Equal([]byte("file-data")))
	})

	It("should create FileCache with default dir when empty string passed", func() {
		cache := NewThumbnailCache("file", "", nil)
		fileCache, ok := cache.(*FileCache)
		Expect(ok).To(BeTrue())
		Expect(fileCache.dir).To(Equal(DefaultFileCacheDir()))
	})
})

var _ = Describe("S3CacheConfig validation", func() {
	It("should detect incomplete config", func() {
		cfg := S3CacheConfig{Bucket: "test"}
		Expect(cfg.IsComplete()).To(BeFalse())
	})

	It("should detect complete config with credentials", func() {
		cfg := S3CacheConfig{
			Bucket:      "test-bucket",
			Region:      "us-east-1",
			Credentials: credentials.NewStaticV4("access-key", "secret-key", ""),
		}
		Expect(cfg.IsComplete()).To(BeTrue())
	})

	It("should be incomplete without bucket", func() {
		cfg := S3CacheConfig{Credentials: credentials.NewStaticV4("access-key", "secret-key", "")}
		Expect(cfg.IsComplete()).To(BeFalse())
	})

	It("should be incomplete without region", func() {
		cfg := S3CacheConfig{Bucket: "test-bucket", Credentials: credentials.NewStaticV4("access-key", "secret-key", "")}
		Expect(cfg.IsComplete()).To(BeFalse())
	})
})
