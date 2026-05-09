package thumbnail

import (
	"fmt"
	"image"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/preprocessor"
	"github.com/stretchr/testify/assert"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/opencloud-eu/opencloud/pkg/log"
	thumbnailconfig "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/config"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail/storage"
)

const (
	testThumbnailChecksum = "1872ade88f3013edeb33decd74a4f947"
	testMatchedKey        = "18/72/ade88f3013edeb33decd74a4f947/512x512-fit.jpeg"
	testRequestedKey      = "18/72/ade88f3013edeb33decd74a4f947/1000x1000-fit.jpeg"
)

type NoOpManager struct {
	storage.Storage
}

func (m NoOpManager) BuildKey(_ storage.Request) string {
	return ""
}

func (m NoOpManager) Set(_, _ string, _ []byte) error {
	return nil
}

type recordingGenerator struct {
	dimensions image.Rectangle
	generated  []image.Rectangle
}

func (g *recordingGenerator) Generate(size image.Rectangle, _ any) (any, error) {
	g.generated = append(g.generated, size)
	return size, nil
}

func (g *recordingGenerator) Dimensions(_ any) (image.Rectangle, error) {
	return g.dimensions, nil
}

func (g *recordingGenerator) ProcessorID() string {
	return "fit"
}

type recordingEncoder struct{}

func (e recordingEncoder) Encode(w io.Writer, img any) error {
	rect, ok := img.(image.Rectangle)
	if !ok {
		return fmt.Errorf("expected image.Rectangle")
	}
	_, err := fmt.Fprintf(w, "%dx%d", rect.Dx(), rect.Dy())
	return err
}

func (e recordingEncoder) Types() []string {
	return []string{typeJpeg}
}

func (e recordingEncoder) MimeType() string {
	return "image/jpeg"
}

func newManagerFixture(t *testing.T) (SimpleManager, storage.FileSystem, *recordingGenerator) {
	t.Helper()

	resolutions, err := ParseResolutions([]string{"64x64", "128x128", "256x256", "512x512"})
	assert.NoError(t, err)

	store := storage.NewFileSystemStorage(
		thumbnailconfig.FileSystemStorage{RootDirectory: t.TempDir()},
		log.NewLogger(),
	)
	generator := &recordingGenerator{dimensions: image.Rect(0, 0, 2000, 1500)}
	manager := NewSimpleManager(resolutions, store, log.NewLogger(), 7680, 7680)

	return manager, store, generator
}

func newTestThumbnailRequest(generator Generator, resolution image.Rectangle) Request {
	return Request{
		Resolution: resolution,
		Encoder:    recordingEncoder{},
		Generator:  generator,
		Checksum:   testThumbnailChecksum,
	}
}

func BenchmarkGet(b *testing.B) {

	sut := NewSimpleManager(
		Resolutions{},
		NoOpManager{},
		log.NewLogger(),
		6016,
		4000,
	)

	res, _ := ParseResolution("32x32")
	req := Request{
		Resolution: res,
		Checksum:   "1872ade88f3013edeb33decd74a4f947",
	}
	cwd, _ := os.Getwd()
	p := filepath.Join(cwd, "../../testdata/test.png")
	f, _ := os.Open(p)
	defer f.Close()
	img, ext, _ := image.Decode(f)
	req.Encoder, _ = EncoderForType(ext)
	for i := 0; i < b.N; i++ {
		_, _ = sut.Generate(req, img)
	}
}

func TestPrepareRequest(t *testing.T) {
	type args struct {
		width    int
		height   int
		tType    string
		checksum string
	}
	tests := []struct {
		name    string
		args    args
		want    Request
		wantErr bool
	}{
		{
			name: "Test successful prepare the request for jpg",
			args: args{
				width:    32,
				height:   32,
				tType:    "jpg",
				checksum: "1872ade88f3013edeb33decd74a4f947",
			},
			want: Request{
				Resolution: image.Rect(0, 0, 32, 32),
				Encoder:    JpegEncoder{},
				Generator:  SimpleGenerator{},
				Checksum:   "1872ade88f3013edeb33decd74a4f947",
			},
		},
		{
			name: "Test successful prepare the request for png",
			args: args{
				width:    32,
				height:   32,
				tType:    "png",
				checksum: "1872ade88f3013edeb33decd74a4f947",
			},
			want: Request{
				Resolution: image.Rect(0, 0, 32, 32),
				Encoder:    PngEncoder{},
				Generator:  SimpleGenerator{},
				Checksum:   "1872ade88f3013edeb33decd74a4f947",
			},
		},
		{
			name: "Test successful prepare the request for gif",
			args: args{
				width:    32,
				height:   32,
				tType:    "gif",
				checksum: "1872ade88f3013edeb33decd74a4f947",
			},
			want: Request{
				Resolution: image.Rect(0, 0, 32, 32),
				Encoder:    GifEncoder{},
				Generator:  GifGenerator{},
				Checksum:   "1872ade88f3013edeb33decd74a4f947",
			},
		},
		{
			name: "Test error when prepare the request for bmp",
			args: args{
				width:    32,
				height:   32,
				tType:    "bmp",
				checksum: "1872ade88f3013edeb33decd74a4f947",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PrepareRequest(tt.args.width, tt.args.height, tt.args.tType, tt.args.checksum, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("PrepareRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// funcs are not reflactable, ignore
			if diff := cmp.Diff(tt.want, got, cmpopts.IgnoreFields(Request{}, "Generator")); diff != "" {
				t.Errorf("PrepareRequest(): %v", diff)
			}
			if reflect.TypeOf(got.Generator) != reflect.TypeOf(tt.want.Generator) {
				t.Errorf("PrepareRequest() = %v, want %v", reflect.TypeOf(got.Generator), reflect.TypeOf(tt.want.Generator))
			}
		})
	}
}

func TestSimpleManagerGenerateUsesClosestMatchForStorageKey(t *testing.T) {
	sut, store, generator := newManagerFixture(t)
	req := newTestThumbnailRequest(generator, image.Rect(0, 0, 1000, 1000))

	key, err := sut.Generate(req, image.Rect(0, 0, 2000, 1500))

	assert.NoError(t, err)
	assert.Equal(t, testMatchedKey, key)
	assert.Equal(t, []image.Rectangle{image.Rect(0, 0, 512, 512)}, generator.generated)
	assert.False(t, store.Stat(testRequestedKey))

	content, err := store.Get(key)
	assert.NoError(t, err)
	assert.Equal(t, []byte("512x512"), content)
}

func TestSimpleManagerGenerateUsesClosestMatchForCacheHit(t *testing.T) {
	sut, store, generator := newManagerFixture(t)
	req := newTestThumbnailRequest(generator, image.Rect(0, 0, 1000, 1000))

	assert.NoError(t, store.Put(testMatchedKey, []byte("cached")))

	key, err := sut.Generate(req, image.Rect(0, 0, 2000, 1500))

	assert.NoError(t, err)
	assert.Equal(t, testMatchedKey, key)
	assert.Empty(t, generator.generated)
}

func TestSimpleManagerCheckThumbnailIgnoresNonConfiguredRequestSize(t *testing.T) {
	sut, store, generator := newManagerFixture(t)
	req := newTestThumbnailRequest(generator, image.Rect(0, 0, 1000, 1000))

	assert.NoError(t, store.Put(testRequestedKey, []byte("stale")))
	assert.NoError(t, store.Put(testMatchedKey, []byte("cached")))

	key, exists := sut.CheckThumbnail(req)

	assert.Empty(t, key)
	assert.False(t, exists)

	req.Resolution = image.Rect(0, 0, 512, 512)
	key, exists = sut.CheckThumbnail(req)

	assert.Equal(t, testMatchedKey, key)
	assert.True(t, exists)
}

func TestPreviewGenerationTooBigImage(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		mimeType string
	}{
		{name: "png", mimeType: "image/png", fileName: "../../testdata/test.png"},
		{name: "jpg", mimeType: "image/jpeg", fileName: "../../testdata/test.jpg"},
		{name: "ggs", mimeType: "application/vnd.geogebra.slides", fileName: "../../testdata/test.ggs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sut := NewSimpleManager(
				Resolutions{},
				NoOpManager{},
				log.NewLogger(),
				1024,
				768,
			)

			res, _ := ParseResolution("32x32")
			req := Request{
				Resolution: res,
				Checksum:   "1872ade88f3013edeb33decd74a4f947",
			}
			cwd, _ := os.Getwd()
			p := filepath.Join(cwd, tt.fileName)
			f, _ := os.Open(p)
			defer f.Close()

			preproc := preprocessor.ForType(tt.mimeType, nil)
			convert, err := preproc.Convert(f)
			if err != nil {
				return
			}

			ext := path.Ext(tt.fileName)
			req.Encoder, _ = EncoderForType(ext)
			req.Generator, err = GeneratorFor(ext, "fit")
			if err != nil {
				return
			}
			generate, err := sut.Generate(req, convert)
			if err != nil {
				return
			}
			assert.ErrorIs(t, err, errors.ErrImageTooLarge)
			assert.Equal(t, "", generate)
		})
	}
}
