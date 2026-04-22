//go:build !enable_vips

package thumbnail

import (
	"image"

	"github.com/kovidgoyal/imaging"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
)

// SimpleGenerator is the default image generator and is used for all image types expect gif.
type SimpleGenerator struct {
	processor Processor
}

func NewSimpleGenerator(filetype, process string) (SimpleGenerator, error) {
	processor, err := ProcessorFor(process, filetype)
	if err != nil {
		return SimpleGenerator{}, err
	}
	return SimpleGenerator{processor: processor}, nil
}

// ProcessorID returns the processor identification.
func (g SimpleGenerator) ProcessorID() string {
	return g.processor.ID()
}

// Generate generates a alternative image version.
func (g SimpleGenerator) Generate(size image.Rectangle, img any) (any, error) {
	m, ok := img.(image.Image)
	if !ok {
		return nil, errors.ErrInvalidType
	}

	return g.processor.Process(m, size.Dx(), size.Dy(), imaging.Lanczos), nil
}

func (g SimpleGenerator) Dimensions(img any) (image.Rectangle, error) {
	m, ok := img.(image.Image)
	if !ok {
		return image.Rectangle{}, errors.ErrInvalidType
	}
	return m.Bounds(), nil
}
