//go:build !enable_vips

package svc

import (
	"bytes"
	"image"
	"image/color/palette"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/kovidgoyal/imaging"
	"github.com/kovidgoyal/imaging/prism/meta/autometa"
)

// The backend this build resizes with. The implementations keep their own names
// so a symbol search finds both builds' versions side by side, the variables are
// what the shared handler calls.
var (
	loadSource         = loadSourceImaging
	extForSourceFormat = extForSourceFormatImaging
	resizeSource       = resizeSourceImaging
	encodeJPEG         = encodeJPEGImaging
	encodePNG          = encodePNGImaging
	encodeGIF          = encodeGIFImaging
	encodeWebP         = encodeWebPImaging
)

// imagingSource is a request's body as this build holds it between the header
// read and the resize: the bytes plus what the header said about them. The
// pixels are decoded in resizeSourceImaging, so the dimension guard has its say
// before anything allocates a frame buffer.
type imagingSource struct {
	data   []byte
	config image.Config
	format string
}

// loadSourceImaging reads the header through the stdlib decoder registry, which this
// build also decodes with: png, jpeg and gif from the stdlib, bmp, tiff and webp
// from the imaging library.
func loadSourceImaging(data []byte) (*imagingSource, error) {
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return &imagingSource{data: data, config: config, format: format}, nil
}

// Width and Height are what the header declared, so the dimension guard can ask
// before the pixels exist. Close is there because the libvips build's image has
// one; this build holds nothing that needs releasing.
func (s *imagingSource) Width() int  { return s.config.Width }
func (s *imagingSource) Height() int { return s.config.Height }
func (s *imagingSource) Close()      {}

// extForSourceFormatImaging is the output format for a request that asked for none: the
// input's own where this build can write it, a JPEG for the rest, which is how
// imagor answers a format it cannot write either.
func extForSourceFormatImaging(src *imagingSource) string {
	switch src.format {
	case "png":
		return "png"
	case "gif":
		return "gif"
	default: // jpeg, webp, tiff, bmp, ...
		return "jpg"
	}
}

// decodeStill reads the pixels of the one frame that is resized here.
// imaging.Decode is DecodeAll plus SingleFrame, so for an animation it holds
// every frame in memory to return the first, and the frame guard only sees a
// gif. The stdlib decoders read the first frame alone: gif.Decode stops after
// it, png.Decode reads IDAT and ignores an animation's later fdAT chunks. The
// rest keeps the imaging path, which is also what reads a JPEG's orientation
// and its color profile.
func decodeStill(src *imagingSource) (image.Image, error) {
	switch src.format {
	case "gif":
		img, err := gif.Decode(bytes.NewReader(src.data))
		if err != nil {
			return nil, err
		}

		return toSRGB(src.data, img), nil
	case "png":
		img, err := png.Decode(bytes.NewReader(src.data))
		if err != nil {
			return nil, err
		}

		return toSRGB(src.data, img), nil
	}

	return imaging.Decode(bytes.NewReader(src.data), imaging.AutoOrientation(true))
}

// toSRGB converts the colors through an embedded ICC profile, which the stdlib
// decoders above hand back untouched: a wide gamut source would be read as sRGB
// and come out oversaturated. imaging.Decode does this for the formats it
// reads, with the same intent and blackpoint setting. A profile that cannot be
// read leaves the colors alone, an uncorrected thumbnail beats none.
func toSRGB(data []byte, img image.Image) image.Image {
	md, _, err := autometa.Load(bytes.NewReader(data))
	if err != nil || md == nil {
		return img
	}

	profile, err := md.ICCProfile()
	if err != nil || profile == nil {
		return img
	}

	converted, err := imaging.ConvertToSRGB(profile, imaging.Relative, true, img)
	if err != nil {
		return img
	}

	return converted
}

// resizeSourceImaging resizes with the imaging backend. The operation selects the
// resize/crop mode: fill (center-crop to the box, upscaling by default like real
// imagor's default resize), fit-in (fit within the box without cropping and
// without ever enlarging the source, like imagor without its upscale() filter),
// or stretch (resize to the exact box). noUpscale caps the default fill at the
// source size, mirroring imagor's no_upscale() filter.
func resizeSourceImaging(src *imagingSource, req resizeRequest) (image.Image, error) {
	img, err := decodeStill(src)
	if err != nil {
		return nil, err
	}

	srcBounds := img.Bounds()

	if req.Width == 0 || req.Height == 0 {
		boxW, boxH, resize, err := req.freeAxis(srcBounds.Dx(), srcBounds.Dy())
		if err != nil {
			return nil, err
		}
		if !resize {
			return img, nil
		}

		return imaging.Resize(img, boxW, boxH, imaging.Lanczos), nil
	}

	switch req.Operation {
	case OpStretch:
		return imaging.Resize(img, req.Width, req.Height, imaging.Lanczos), nil
	case OpFitIn:
		return fitIn(img, req.Width, req.Height), nil
	default: // OpFill
		if req.NoUpscale && srcBounds.Dx() <= req.Width && srcBounds.Dy() <= req.Height {
			// imagor no_upscale(): never enlarge the source.
			return img, nil
		}

		// smart() asks for the most interesting window, which needs libvips;
		// this build crops the centre for it, the way main always did

		return imaging.Thumbnail(img, req.Width, req.Height, imaging.Lanczos), nil
	}
}

func encodeJPEGImaging(w io.Writer, img image.Image) error { return jpeg.Encode(w, img, nil) }

func encodePNGImaging(w io.Writer, img image.Image) error { return png.Encode(w, img) }

// encodeGIFImaging palettes a still image with the same dithering the animated path uses.
func encodeGIFImaging(w io.Writer, img image.Image) error {
	return gif.Encode(w, palettedAtOrigin(img, palette.Plan9), nil)
}

// encodeWebPImaging refuses: there is no webp encoder in the pure Go build, so a
// deployment that needs webp output runs the libvips build or a real imagor.
func encodeWebPImaging(io.Writer, image.Image) error { return errFormatUnsupported }
