package autometa

import (
	"fmt"
	"io"

	"github.com/kovidgoyal/imaging/prism/meta"
	"github.com/kovidgoyal/imaging/prism/meta/jpegmeta"
	"github.com/kovidgoyal/imaging/prism/meta/pngmeta"
	"github.com/kovidgoyal/imaging/prism/meta/webpmeta"
	"github.com/kovidgoyal/imaging/streams"
)

// Load loads the metadata for an image stream, which may be one of the
// supported image formats.
//
// Only as much of the stream is consumed as necessary to extract the metadata;
// the returned stream contains a buffered copy of the consumed data such that
// reading from it will produce the same results as fully reading the input
// stream. This provides a convenient way to load the full image after loading
// the metadata.
//
// An error is returned if basic metadata could not be extracted. The returned
// stream still provides the full image data.
func Load(r io.Reader) (md *meta.Data, imgStream io.Reader, err error) {
	loaders := []func(io.Reader) (*meta.Data, error){
		pngmeta.ExtractMetadata,
		jpegmeta.ExtractMetadata,
		webpmeta.ExtractMetadata,
	}
	for _, loader := range loaders {
		r, err = streams.CallbackWithSeekable(r, func(r io.Reader) (err error) {
			md, err = loader(r)
			return
		})
		if err == nil {
			return md, r, nil
		}
	}
	return nil, r, fmt.Errorf("unrecognised image format")
}
