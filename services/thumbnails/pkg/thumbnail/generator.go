package thumbnail

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"strings"

	"github.com/kovidgoyal/imaging"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
)

// Generator generates a web friendly file version.
type Generator interface {
	Generate(size image.Rectangle, img interface{}) (interface{}, error)
	Dimensions(img interface{}) (image.Rectangle, error)
	ProcessorID() string
}

// GifGenerator is used to create a web friendly version of the provided gif image.
type GifGenerator struct {
	processor Processor
}

func NewGifGenerator(filetype, process string) (GifGenerator, error) {
	processor, err := ProcessorFor(process, filetype)
	if err != nil {
		return GifGenerator{}, err
	}
	return GifGenerator{processor: processor}, nil

}

// ProcessorID returns the processor identification.
func (g GifGenerator) ProcessorID() string {
	return g.processor.ID()
}

// Generate generates a alternative gif version.
func (g GifGenerator) Generate(size image.Rectangle, img interface{}) (interface{}, error) {
	// Code inspired by https://github.com/willnorris/gifresize/blob/db93a7e1dcb1c279f7eeb99cc6d90b9e2e23e871/gifresize.go

	m, ok := img.(*gif.GIF)
	if !ok {
		return nil, errors.ErrInvalidType
	}
	// Create a new RGBA image to hold the incremental frames.
	srcX, srcY := m.Config.Width, m.Config.Height
	b := image.Rect(0, 0, srcX, srcY)
	tmp := image.NewRGBA(b)

	for i, frame := range m.Image {
		bounds := frame.Bounds()
		prev := tmp
		draw.Draw(tmp, bounds, frame, bounds.Min, draw.Over)
		processed := g.processor.Process(tmp, size.Dx(), size.Dy(), imaging.Lanczos)
		m.Image[i] = g.imageToPaletted(processed, frame.Palette)

		switch m.Disposal[i] {
		case gif.DisposalBackground:
			tmp = image.NewRGBA(b)
		case gif.DisposalPrevious:
			tmp = prev
		}
	}
	m.Config.Width = size.Dx()
	m.Config.Height = size.Dy()

	return m, nil
}

func (g GifGenerator) Dimensions(img interface{}) (image.Rectangle, error) {
	m, ok := img.(*gif.GIF)
	if !ok {
		return image.Rectangle{}, errors.ErrInvalidType
	}
	return m.Image[0].Bounds(), nil
}

func (g GifGenerator) imageToPaletted(img image.Image, p color.Palette) *image.Paletted {
	b := img.Bounds()
	pm := image.NewPaletted(b, p)
	draw.FloydSteinberg.Draw(pm, b, img, image.Point{})
	return pm
}

// GeneratorFor returns the generator for a given file type
// or nil if the type is not supported.
func GeneratorFor(fileType, processorID string) (Generator, error) {
	switch strings.ToLower(fileType) {
	case typePng, typeJpg, typeJpeg, typeGgs, typeGgp, typeWebp:
		return NewSimpleGenerator(fileType, processorID)
	case typeGif:
		return NewGifGenerator(fileType, processorID)
	default:
		return nil, errors.ErrNoGeneratorForType
	}
}
