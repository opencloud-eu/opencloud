package thumbnail_test

import (
	"testing"

	"github.com/kovidgoyal/imaging"
	tAssert "github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail"
)

func TestProcessorFor(t *testing.T) {
	tests := []struct {
		id       string
		fileType string
		wantP    thumbnail.Processor
		wantE    error
	}{
		{
			id:       "fit",
			fileType: "",
			wantP:    thumbnail.DefinableProcessor{Slug: "fit", Converter: imaging.Fit},
			wantE:    nil,
		},
		{
			id:       "fit",
			fileType: "jpg",
			wantP:    thumbnail.DefinableProcessor{Slug: "fit"},
			wantE:    nil,
		},
		{
			id:       "FIT",
			fileType: "jpg",
			wantP:    thumbnail.DefinableProcessor{Slug: "fit"},
			wantE:    nil,
		},
		{
			id:       "resize",
			fileType: "jpg",
			wantP:    thumbnail.DefinableProcessor{Slug: "resize"},
			wantE:    nil,
		},
		{
			id:       "RESIZE",
			fileType: "jpg",
			wantP:    thumbnail.DefinableProcessor{Slug: "resize"},
			wantE:    nil,
		},
		{
			id:       "fill",
			fileType: "jpg",
			wantP:    thumbnail.DefinableProcessor{Slug: "fill"},
			wantE:    nil,
		},
		{
			id:       "FILL",
			fileType: "jpg",
			wantP:    thumbnail.DefinableProcessor{Slug: "fill"},
			wantE:    nil,
		},
		{
			id:       "thumbnail",
			fileType: "jpg",
			wantP:    thumbnail.DefinableProcessor{Slug: "thumbnail"},
			wantE:    nil,
		},
		{
			id:       "THUMBNAIL",
			fileType: "jpg",
			wantP:    thumbnail.DefinableProcessor{Slug: "thumbnail"},
			wantE:    nil,
		},
		{
			id:       "",
			fileType: "jpg",
			wantP:    thumbnail.DefinableProcessor{},
			wantE:    nil,
		},
		{
			id:       "",
			fileType: "gif",
			wantP:    thumbnail.DefinableProcessor{},
			wantE:    nil,
		},
	}

	assert := tAssert.New(t)

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			p, e := thumbnail.ProcessorFor(tt.id, tt.fileType)
			assert.Equal(p.ID(), tt.wantP.ID())
			assert.Equal(e, tt.wantE)
		})
	}

}
