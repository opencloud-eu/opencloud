//go:build !enable_vips

package svc

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/kovidgoyal/imaging"
)

func init() {
	processImage = processImageImaging
	encodeJPEG = encodeJPEGImaging
	encodePNG = encodePNGImaging
	encodeGIF = encodeGIFImaging
	encodeWebP = encodeWebPImaging
}

// processImageImaging resizes the input using the imaging backend. The operation
// selects the resize/crop mode: fill (center-crop to the box, upscaling by default
// like real imagor's default resize), fit-in (fit within the box without cropping
// and without ever enlarging the source, like imagor without its upscale()
// filter), or stretch (resize to the exact box). noUpscale caps the default fill
// at the source size, mirroring imagor's no_upscale() filter.
func processImageImaging(r io.Reader, req resizeRequest) (any, error) {
	if req.KeepAnimation && isGifReader(r) {
		g, err := gif.DecodeAll(r)
		if err == nil && len(g.Image) > 0 {
			return resizeGIF(g, req.Width, req.Height, req.Operation, req.NoUpscale), nil
		}
	}

	img, err := imaging.Decode(r, imaging.AutoOrientation(true))
	if err != nil {
		return nil, err
	}

	srcBounds := img.Bounds()

	if req.Width == 0 || req.Height == 0 {
		boxW, boxH, resize := autoAxis(srcBounds.Dx(), srcBounds.Dy(), req.Width, req.Height)
		if !resize {
			return img, nil
		}

		return imaging.Resize(img, boxW, boxH, imaging.Lanczos), nil
	}

	switch req.Operation {
	case OpStretch:
		return imaging.Resize(img, req.Width, req.Height, imaging.Lanczos), nil
	case OpFitIn:
		// imagor's fit-in only enlarges with the upscale() filter, which webdav
		// never sends, so a source smaller than the box is returned as it is.
		// Not imaging.Fit: it truncates the scaled side, vips and imagor round it.
		w, h := fitInBox(srcBounds.Dx(), srcBounds.Dy(), req.Width, req.Height)
		if w >= srcBounds.Dx() && h >= srcBounds.Dy() {
			return img, nil
		}
		return imaging.Resize(img, w, h, imaging.Lanczos), nil
	default: // OpFill
		if req.NoUpscale && srcBounds.Dx() <= req.Width && srcBounds.Dy() <= req.Height {
			// imagor no_upscale(): never enlarge the source.
			return img, nil
		}
		return imaging.Thumbnail(img, req.Width, req.Height, imaging.Lanczos), nil
	}
}

// encodeJPEGImaging encodes the processed image as JPEG using the stdlib encoder.
func encodeJPEGImaging(w io.Writer, processed any) error {
	img, ok := processed.(image.Image)
	if !ok {
		return fmt.Errorf("cannot encode %T as jpeg", processed)
	}
	return jpeg.Encode(w, img, nil)
}

// encodePNGImaging encodes the processed image as PNG using the stdlib encoder.
func encodePNGImaging(w io.Writer, processed any) error {
	img, ok := processed.(image.Image)
	if !ok {
		return fmt.Errorf("cannot encode %T as png", processed)
	}
	return png.Encode(w, img)
}

// encodeGIFImaging writes an animated gif as it is and palettes a still image
// with the same dithering the animated path uses.
func encodeGIFImaging(w io.Writer, processed any) error {
	if g, ok := processed.(*gif.GIF); ok {
		return gif.EncodeAll(w, g)
	}

	img, ok := processed.(image.Image)
	if !ok {
		return fmt.Errorf("cannot encode %T as gif", processed)
	}

	return gif.Encode(w, palettedAtOrigin(img, palette.Plan9), nil)
}

// encodeWebPImaging refuses: there is no webp encoder in the pure Go build, so a
// deployment that needs webp output runs the libvips build or a real imagor.
func encodeWebPImaging(io.Writer, any) error {
	return errFormatUnsupported
}
