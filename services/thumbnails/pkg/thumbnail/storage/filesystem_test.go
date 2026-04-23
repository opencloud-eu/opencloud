package storage_test

import (
	"image"
	"testing"

	tAssert "github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail/storage"
)

func TestFileSystem_BuildKey(t *testing.T) {
	tests := []struct {
		r    storage.Request
		want string
	}{
		{
			r: storage.Request{
				Checksum: "120EA8A25E5D487BF68B5F7096440019",
				Types:    []string{"png", "jpg"},
				Resolution: image.Rectangle{
					Min: image.Point{
						X: 1,
						Y: 2,
					},
					Max: image.Point{
						X: 3,
						Y: 4,
					},
				},
				Characteristic: "",
			},
			want: "12/0E/A8A25E5D487BF68B5F7096440019/2x2.png",
		},
		{
			r: storage.Request{
				Checksum: "120EA8A25E5D487BF68B5F7096440019",
				Types:    []string{"png", "jpg"},
				Resolution: image.Rectangle{
					Min: image.Point{
						X: 1,
						Y: 2,
					},
					Max: image.Point{
						X: 3,
						Y: 4,
					},
				},
				Characteristic: "fill",
			},
			want: "12/0E/A8A25E5D487BF68B5F7096440019/2x2-fill.png",
		},
	}

	s := storage.FileSystem{}
	assert := tAssert.New(t)

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			assert.Equal(s.BuildKey(tt.r), tt.want)
		})
	}

}
