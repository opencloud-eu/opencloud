package thumbnail

import (
	"context"
	"image"
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
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail/storage"
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
			convert, err := preproc.Convert(context.TODO(), f)
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
