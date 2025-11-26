package autometa

import (
	"io"

	"github.com/kovidgoyal/imaging/prism/meta"
	"github.com/kovidgoyal/imaging/prism/meta/gifmeta"
	"github.com/kovidgoyal/imaging/prism/meta/jpegmeta"
	"github.com/kovidgoyal/imaging/prism/meta/netpbmmeta"
	"github.com/kovidgoyal/imaging/prism/meta/pngmeta"
	"github.com/kovidgoyal/imaging/prism/meta/tiffmeta"
	"github.com/kovidgoyal/imaging/prism/meta/webpmeta"
	"github.com/kovidgoyal/imaging/streams"
)

var loaders = []func(io.Reader) (*meta.Data, error){
	jpegmeta.ExtractMetadata,
	pngmeta.ExtractMetadata,
	gifmeta.ExtractMetadata,
	webpmeta.ExtractMetadata,
	tiffmeta.ExtractMetadata,
	netpbmmeta.ExtractMetadata,
}

// Load loads the metadata for an image stream, which may be one of the
// supported image formats.
//
// Only as much of the stream is consumed as necessary to extract the metadata;
// the returned stream contains a buffered copy of the consumed data such that
// reading from it will produce the same results as fully reading the input
// stream. This provides a convenient way to load the full image after loading
// the metadata.
//
// Returns nil if the no image format was recognized. Returns an error if there
// was an error decoding metadata.
func Load(r io.Reader) (md *meta.Data, imgStream io.Reader, err error) {
	for _, loader := range loaders {
		r, err = streams.CallbackWithSeekable(r, func(r io.Reader) (err error) {
			md, err = loader(r)
			return
		})
		switch {
		case err != nil:
			return nil, r, err
		case md != nil:
			return md, r, nil
		}
	}
	return nil, r, nil
}
