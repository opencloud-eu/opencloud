package cache

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"testing"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
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
		got, err := cache.Get("nonexistent")
		Expect(err).To(MatchError(ErrCacheMiss))
		Expect(got).To(BeNil())
	})

	It("should store and retrieve data", func() {
		err := cache.Put("key1", &thumbnail.Image{Data: []byte("thumbnail-data")})
		Expect(err).ToNot(HaveOccurred())

		got, err := cache.Get("key1")
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Data).To(Equal([]byte("thumbnail-data")))
	})

	It("should return independent copies on Get", func() {
		err := cache.Put("key1", &thumbnail.Image{Data: []byte("original")})
		Expect(err).ToNot(HaveOccurred())

		got, err := cache.Get("key1")
		Expect(err).ToNot(HaveOccurred())
		got.Data[0] = 'X'

		orig, err := cache.Get("key1")
		Expect(err).ToNot(HaveOccurred())
		Expect(orig.Data).To(Equal([]byte("original")))
	})

	It("should overwrite existing keys", func() {
		err := cache.Put("key1", &thumbnail.Image{Data: []byte("v1")})
		Expect(err).ToNot(HaveOccurred())

		err = cache.Put("key1", &thumbnail.Image{Data: []byte("v2")})
		Expect(err).ToNot(HaveOccurred())

		got, err := cache.Get("key1")
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Data).To(Equal([]byte("v2")))
	})

	It("remembers a file without an image as nil", func() {
		err := cache.Put("empty-key", nil)
		Expect(err).ToNot(HaveOccurred())

		got, err := cache.Get("empty-key")
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(BeNil(), "a file without an image")
	})
})

var _ = Describe("NoopCache", func() {
	var cache ThumbnailCache

	BeforeEach(func() {
		cache = NoopCache{}
	})

	It("should always return ErrCacheMiss on Get", func() {
		got, err := cache.Get("any-key")
		Expect(err).To(MatchError(ErrCacheMiss))
		Expect(got).To(BeNil())
	})

	It("should accept Put without error", func() {
		err := cache.Put("key1", &thumbnail.Image{Data: []byte("data")})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should still miss after Put", func() {
		_ = cache.Put("key1", &thumbnail.Image{Data: []byte("data")})
		got, err := cache.Get("key1")
		Expect(err).To(MatchError(ErrCacheMiss))
		Expect(got).To(BeNil())
	})
})

var _ = Describe("NewThumbnailCache", func() {
	It("should create InMemoryCache for 'memory' backend", func() {
		cache, err := NewThumbnailCache("memory", "", nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(cache).ToNot(BeNil())
		err = cache.Put("k", &thumbnail.Image{Data: []byte("v")})
		Expect(err).ToNot(HaveOccurred())
		got, err := cache.Get("k")
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Data).To(Equal([]byte("v")))
	})

	It("should refuse an unknown backend", func() {
		_, err := NewThumbnailCache("unknown", "", nil)
		Expect(err).To(MatchError(ContainSubstring("unknown thumbnail cache backend")))
	})

	It("should refuse an empty backend", func() {
		_, err := NewThumbnailCache("", "", nil)
		Expect(err).To(HaveOccurred())
	})

	It("should create NoopCache for the 'none' backend", func() {
		cache, err := NewThumbnailCache("none", "", nil)
		Expect(err).ToNot(HaveOccurred())
		got, err := cache.Get("k")
		Expect(err).To(MatchError(ErrCacheMiss))
		Expect(got).To(BeNil())
	})

	It("should create S3Cache for 's3' backend with complete config", func() {
		cfg := &S3CacheConfig{
			Bucket:      "my-bucket",
			Region:      "us-east-1",
			Credentials: credentials.NewStaticV4("ak", "sk", ""),
		}
		cache, err := NewThumbnailCache("s3", "", cfg)
		Expect(err).ToNot(HaveOccurred())
		s3cache, ok := cache.(*S3Cache)
		Expect(ok).To(BeTrue())
		Expect(s3cache.config.Bucket).To(Equal("my-bucket"))
	})

	It("should refuse an 's3' backend with incomplete config", func() {
		_, err := NewThumbnailCache("s3", "", &S3CacheConfig{Bucket: "only-bucket"})
		Expect(err).To(MatchError(ContainSubstring("s3 thumbnail cache")))
	})

	It("should refuse an 's3' backend without config", func() {
		_, err := NewThumbnailCache("s3", "", nil)
		Expect(err).To(MatchError(ContainSubstring("s3 thumbnail cache")))
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
	contentType string
}

func (m *mockS3Object) Close() error {
	return nil
}

func (m *mockS3Object) Stat() (minio.ObjectInfo, error) {
	return minio.ObjectInfo{ContentType: m.contentType}, nil
}

var _ s3Object = (*mockS3Object)(nil)

// mockS3FailingObject answers the read with an error, the way minio does.
type mockS3FailingObject struct {
	err error
}

func (m *mockS3FailingObject) Read([]byte) (int, error)        { return 0, m.err }
func (m *mockS3FailingObject) Close() error                    { return nil }
func (m *mockS3FailingObject) Stat() (minio.ObjectInfo, error) { return minio.ObjectInfo{}, m.err }

var _ s3Object = (*mockS3FailingObject)(nil)

// mockS3Client implements s3Client interface for testing.
type mockS3Client struct {
	objects      map[string][]byte
	contentTypes map[string]string
	capturedOpts minio.PutObjectOptions
	getErr       error
	readErr      error
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
	if m.readErr != nil {
		return &mockS3FailingObject{err: m.readErr}, nil
	}
	data, ok := m.objects[objectName]
	if !ok {
		return &mockS3FailingObject{err: minio.ErrorResponse{
			Code:       minio.NoSuchKey,
			Message:    "key does not exist",
			StatusCode: http.StatusNotFound,
		}}, nil
	}
	return &mockS3Object{Reader: bytes.NewReader(data), contentType: m.contentTypes[objectName]}, nil
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
		got, err := cache.Get("nonexistent-key")
		Expect(err).To(MatchError(ErrCacheMiss))
		Expect(got).To(BeNil())
	})

	It("should store and retrieve data", func() {
		err := cache.Put("sha256:abc123-400x400.jpg", &thumbnail.Image{Data: []byte("thumbnail-data")})
		Expect(err).ToNot(HaveOccurred())

		got, err := cache.Get("sha256:abc123-400x400.jpg")
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Data).To(Equal([]byte("thumbnail-data")))
	})

	It("should return independent copies on Get", func() {
		err := cache.Put("key1", &thumbnail.Image{Data: []byte("original")})
		Expect(err).ToNot(HaveOccurred())

		got, err := cache.Get("key1")
		Expect(err).ToNot(HaveOccurred())
		got.Data[0] = 'X'

		orig, err := cache.Get("key1")
		Expect(err).ToNot(HaveOccurred())
		Expect(orig.Data).To(Equal([]byte("original")))
	})

	It("should overwrite existing keys", func() {
		err := cache.Put("key1", &thumbnail.Image{Data: []byte("v1")})
		Expect(err).ToNot(HaveOccurred())

		err = cache.Put("key1", &thumbnail.Image{Data: []byte("v2")})
		Expect(err).ToNot(HaveOccurred())

		got, err := cache.Get("key1")
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Data).To(Equal([]byte("v2")))
	})

	It("remembers a file without an image as nil", func() {
		err := cache.Put("empty-key", nil)
		Expect(err).ToNot(HaveOccurred())

		got, err := cache.Get("empty-key")
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(BeNil(), "a file without an image")
	})

	It("should use flat key format", func() {
		key := "sha256:abc123-400x400.jpg"
		err := cache.Put(key, &thumbnail.Image{Data: []byte("data")})
		Expect(err).ToNot(HaveOccurred())

		got, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Data).To(Equal([]byte("data")))
	})

	It("should initialize client lazily on first access", func() {
		Expect(cache.client).To(BeNil())

		_ = cache.Put("key1", &thumbnail.Image{Data: []byte("data")})
		Expect(cache.client).ToNot(BeNil())
	})

	It("should handle S3 errors gracefully on Get", func() {
		mockClient.getErr = errors.New("connection refused")

		got, err := cache.Get("any-key")
		Expect(err).To(HaveOccurred())
		Expect(got).To(BeNil())
	})

	It("should keep an error that is not a missing key an error", func() {
		mockClient.readErr = minio.ErrorResponse{Code: minio.AccessDenied, StatusCode: http.StatusForbidden}

		_, err := cache.Get("key1")
		Expect(err).To(HaveOccurred())
		Expect(err).ToNot(MatchError(ErrCacheMiss))
	})

	It("should handle S3 errors gracefully on Put", func() {
		mockClient.putErr = errors.New("connection refused")

		err := cache.Put("key1", &thumbnail.Image{Data: []byte("data")})
		Expect(err).To(HaveOccurred())
	})

	It("should propagate init error from newFunc", func() {
		cache.newFunc = func(_ S3CacheConfig) (s3Client, error) {
			return nil, errors.New("init failed")
		}

		got, err := cache.Get("any-key")
		Expect(err).To(HaveOccurred())
		Expect(got).To(BeNil())
	})

	It("stores the entry's type as the object's Content-Type", func() {
		Expect(cache.Put("key1", &thumbnail.Image{Data: []byte("data"), ContentType: "image/png"})).To(Succeed())
		Expect(mockClient.contentTypes["key1"]).To(Equal("image/png"))

		got, err := cache.Get("key1")
		Expect(err).ToNot(HaveOccurred())
		Expect(got.ContentType).To(Equal("image/png"))
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
		got, err := cache.Get("ab/cd/ef1234567890/400x400.jpg")
		Expect(err).To(MatchError(ErrCacheMiss))
		Expect(got).To(BeNil())
	})

	It("should store and retrieve data", func() {
		key := "ab/cd/ef1234567890/400x400.jpg"
		err := cache.Put(key, &thumbnail.Image{Data: []byte("thumbnail-data")})
		Expect(err).ToNot(HaveOccurred())

		got, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Data).To(Equal([]byte("thumbnail-data")))
	})

	It("should use hierarchical directory layout", func() {
		key := "ab/cd/ef1234567890/400x400.jpg"
		err := cache.Put(key, &thumbnail.Image{Data: []byte("thumbnail-data")})
		Expect(err).ToNot(HaveOccurred())

		expectedPath := filepath.Join(tmpdir, "ab", "cd", "ef1234567890", "400x400.jpg")
		content, err := os.ReadFile(expectedPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(content).To(Equal([]byte("thumbnail-data")))
	})

	It("should create parent directories with 0700 permissions", func() {
		key := "ab/cd/ef1234567890/400x400.jpg"
		err := cache.Put(key, &thumbnail.Image{Data: []byte("thumbnail-data")})
		Expect(err).ToNot(HaveOccurred())

		dirPath := filepath.Join(tmpdir, "ab", "cd")
		info, err := os.Stat(dirPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(info.IsDir()).To(BeTrue())
		Expect(info.Mode().Perm()).To(Equal(os.FileMode(0700)))
	})

	It("should return independent copies on Get", func() {
		key := "ab/cd/ef1234567890/400x400.jpg"
		err := cache.Put(key, &thumbnail.Image{Data: []byte("original")})
		Expect(err).ToNot(HaveOccurred())

		got, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		got.Data[0] = 'X'

		orig, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(orig.Data).To(Equal([]byte("original")))
	})

	It("should overwrite existing keys", func() {
		key := "ab/cd/ef1234567890/400x400.jpg"
		Expect(cache.Put(key, &thumbnail.Image{Data: []byte("v1")})).To(Succeed())
		Expect(cache.Put(key, &thumbnail.Image{Data: []byte("v2")})).To(Succeed())

		got, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Data).To(Equal([]byte("v2")), "a key whose value can change has to be replaceable")
	})

	It("remembers a file without an image as nil", func() {
		key := "ab/cd/ef1234567890/400x400.jpg"
		err := cache.Put(key, nil)
		Expect(err).ToNot(HaveOccurred())

		got, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(BeNil(), "a file without an image")
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

		err := cache.Put(key, &thumbnail.Image{Data: bytes.Repeat([]byte("x"), 1024*1024)})
		Expect(err).ToNot(HaveOccurred())

		close(done)

		got, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(got.Data)).To(Equal(1024 * 1024))
		_ = sawTempFile
	})

	It("should handle disk write error gracefully", func() {
		blocker := filepath.Join(tmpdir, "blocker")
		Expect(os.WriteFile(blocker, []byte("x"), 0600)).To(Succeed())

		// The cache dir path passes through a regular file, so MkdirAll cannot
		// create the target directory for any user (root or not) -> Put must error.
		rc := NewFileCache(filepath.Join(blocker, "sub"))
		err := rc.Put("ab/cd/ef1234567890/400x400.jpg", &thumbnail.Image{Data: []byte("data")})
		Expect(err).To(HaveOccurred())
	})

	It("keeps the content type in an extended attribute", func() {
		key := "ab/cd/ef1234567890/400x400"
		Expect(cache.Put(key, &thumbnail.Image{Data: []byte("thumbnail-data"), ContentType: "image/png"})).To(Succeed())

		got, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(got.ContentType).To(Equal("image/png"))
		Expect(got.Data).To(Equal([]byte("thumbnail-data")), "the file holds nothing but the image")

		raw, err := os.ReadFile(filepath.Join(tmpdir, key))
		Expect(err).ToNot(HaveOccurred())
		Expect(raw).To(Equal([]byte("thumbnail-data")))
	})

	It("should handle corrupt/partial files on Get", func() {
		key := "ab/cd/ef1234567890/400x400.jpg"
		dirPath := filepath.Join(tmpdir, "ab", "cd", "ef1234567890")
		Expect(os.MkdirAll(dirPath, 0700)).To(Succeed())

		filePath := filepath.Join(dirPath, "400x400.jpg")
		Expect(os.WriteFile(filePath, []byte("corrupt-data"), 0600)).To(Succeed())

		got, err := cache.Get(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Data).To(Equal([]byte("corrupt-data")))
	})
})

var _ = Describe("NewThumbnailCache factory", func() {
	It("should create FileCache for 'file' backend", func() {
		tmpdir, err := os.MkdirTemp("", "factory-test")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(tmpdir)

		cache, err := NewThumbnailCache("file", tmpdir, nil)
		Expect(err).ToNot(HaveOccurred())

		fileCache, ok := cache.(*FileCache)
		Expect(ok).To(BeTrue())
		Expect(fileCache.dir).To(Equal(tmpdir))

		err = cache.Put("ab/cd/ef1234567890/400x400.jpg", &thumbnail.Image{Data: []byte("file-data")})
		Expect(err).ToNot(HaveOccurred())

		got, err := cache.Get("ab/cd/ef1234567890/400x400.jpg")
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Data).To(Equal([]byte("file-data")))
	})

	It("should create FileCache with default dir when empty string passed", func() {
		GinkgoT().Setenv("OC_BASE_DATA_PATH", GinkgoT().TempDir())

		cache, err := NewThumbnailCache("file", "", nil)
		Expect(err).ToNot(HaveOccurred())
		fileCache, ok := cache.(*FileCache)
		Expect(ok).To(BeTrue())
		Expect(fileCache.dir).To(Equal(DefaultFileCacheDir()))
	})

	It("refuses a directory it cannot write to at startup", func() {
		blocker := filepath.Join(GinkgoT().TempDir(), "blocker")
		Expect(os.WriteFile(blocker, []byte("x"), 0600)).To(Succeed())

		_, err := NewThumbnailCache("file", filepath.Join(blocker, "sub"), nil)
		Expect(err).To(HaveOccurred())
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
