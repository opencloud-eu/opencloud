package netpbmmeta

import (
	"fmt"
	"io"
	"strings"

	"github.com/kovidgoyal/imaging/netpbm"
	"github.com/kovidgoyal/imaging/prism/meta"
	"github.com/kovidgoyal/imaging/prism/meta/tiffmeta"
)

var _ = fmt.Print

func ExtractMetadata(r io.Reader) (md *meta.Data, err error) {
	c, fmt, err := netpbm.DecodeConfigAndFormat(r)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported netPBM format") {
			err = nil
		}
		return nil, err
	}
	md = &meta.Data{
		Format: fmt, PixelWidth: uint32(c.Width), PixelHeight: uint32(c.Height),
		BitsPerComponent: tiffmeta.BitsPerComponent(c.ColorModel),
	}
	return md, nil
}
