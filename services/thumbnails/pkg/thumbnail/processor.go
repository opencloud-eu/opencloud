package thumbnail

import (
	"image"
	"strings"

	"github.com/kovidgoyal/imaging"
)

// Processor processes the thumbnail by applying different transformations to it.
type Processor interface {
	ID() string
	Process(img image.Image, width, height int, filter imaging.ResampleFilter) image.Image
}

// DefinableProcessor is the simplest processor, it holds a replaceable image converter function.
type DefinableProcessor struct {
	Slug      string
	Converter func(img image.Image, width, height int, filter imaging.ResampleFilter) image.Image
}

// ID returns the processor identification.
func (p DefinableProcessor) ID() string { return p.Slug }

// Process transforms the given image.
func (p DefinableProcessor) Process(img image.Image, width, height int, filter imaging.ResampleFilter) image.Image {
	return p.Converter(img, width, height, filter)
}

// ProcessorFor returns a matching Processor
func ProcessorFor(id, fileType string) (DefinableProcessor, error) {
	switch strings.ToLower(id) {
	case "fit":
		return DefinableProcessor{Slug: strings.ToLower(id), Converter: imaging.Fit}, nil
	case "resize":
		return DefinableProcessor{Slug: strings.ToLower(id), Converter: imaging.Resize}, nil
	case "fill":
		return DefinableProcessor{Slug: strings.ToLower(id), Converter: func(img image.Image, width, height int, filter imaging.ResampleFilter) image.Image {
			return imaging.Fill(img, width, height, imaging.Center, filter)
		}}, nil
	case "thumbnail":
		return DefinableProcessor{Slug: strings.ToLower(id), Converter: imaging.Thumbnail}, nil
	default:
		switch strings.ToLower(fileType) {
		case typeGif:
			return DefinableProcessor{Converter: imaging.Resize}, nil
		default:
			return DefinableProcessor{Converter: imaging.Thumbnail}, nil
		}
	}
}
