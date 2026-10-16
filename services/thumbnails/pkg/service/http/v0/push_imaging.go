//go:build !enable_vips

package svc

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"

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
// like real imagor's default resize), fit-in (fit within the box without cropping,
// upscaling small sources by default like real imagor), or stretch (resize to the
// exact box). noUpscale caps the
// default fill at the source size, mirroring imagor's no_upscale() filter.
func processImageImaging(r io.Reader, width, height int, operation string, noUpscale, keepAnimation bool) (any, error) {
	if keepAnimation && isGifReader(r) {
		g, err := gif.DecodeAll(r)
		if err == nil && len(g.Image) > 0 {
			return resizeGIF(g, width, height, operation, noUpscale), nil
		}
	}

	img, err := imaging.Decode(r, imaging.AutoOrientation(true))
	if err != nil {
		return nil, err
	}

	srcBounds := img.Bounds()

	switch operation {
	case OpStretch:
		return imaging.Resize(img, width, height, imaging.Lanczos), nil
	case OpFitIn:
		// imagor's fit-in only enlarges with the upscale() filter, which webdav
		// never sends, so a source smaller than the box is returned as it is.
		// Not imaging.Fit: it truncates the scaled side, vips and imagor round it.
		w, h := fitInBox(srcBounds.Dx(), srcBounds.Dy(), width, height)
		if w >= srcBounds.Dx() && h >= srcBounds.Dy() {
			return img, nil
		}
		return imaging.Resize(img, w, h, imaging.Lanczos), nil
	default: // OpFill
		if noUpscale && srcBounds.Dx() <= width && srcBounds.Dy() <= height {
			// imagor no_upscale(): never enlarge the source.
			return img, nil
		}
		return imaging.Thumbnail(img, width, height, imaging.Lanczos), nil
	}
}

// fitInBox is the size an image of srcW x srcH takes inside the box, aspect
// ratio preserved and the scaled side rounded the way vips and imagor round it.
func fitInBox(srcW, srcH, boxW, boxH int) (int, int) {
	scale := math.Min(float64(boxW)/float64(srcW), float64(boxH)/float64(srcH))

	return max(int(math.Round(float64(srcW)*scale)), 1), max(int(math.Round(float64(srcH)*scale)), 1)
}

// resizeGIF resizes every frame of an animated gif while preserving the
// animation. It composites each frame onto a running canvas honoring the gif
// disposal method, resizes with the requested processor, and re-pallettes the
// result using Floyd-Steinberg dithering. Code adapted from
// https://github.com/willnorris/gifresize. noUpscale caps the default fill at
// the source size, mirroring imagor's no_upscale() filter.
func resizeGIF(m *gif.GIF, width, height int, operation string, noUpscale bool) *gif.GIF {
	srcX, srcY := m.Config.Width, m.Config.Height
	b := image.Rect(0, 0, srcX, srcY)
	tmp := image.NewRGBA(b)

	for i, frame := range m.Image {
		frameBounds := frame.Bounds()
		prev := tmp
		draw.Draw(tmp, frameBounds, frame, frameBounds.Min, draw.Over)

		var processed image.Image
		switch operation {
		case OpStretch:
			processed = imaging.Resize(tmp, width, height, imaging.Lanczos)
		case OpFitIn:
			if noUpscale && srcX <= width && srcY <= height {
				processed = tmp
			} else if srcX <= width && srcY <= height {
				srcAR := float64(srcX) / float64(srcY)
				boxAR := float64(width) / float64(height)
				if srcAR > boxAR {
					processed = imaging.Resize(tmp, width, int(float64(width)/srcAR), imaging.Lanczos)
				} else {
					processed = imaging.Resize(tmp, int(float64(height)*srcAR), height, imaging.Lanczos)
				}
			} else {
				processed = imaging.Fit(tmp, width, height, imaging.Lanczos)
			}
		default: // OpFill
			if noUpscale && srcX <= width && srcY <= height {
				processed = tmp
			} else {
				processed = imaging.Fill(tmp, width, height, imaging.Center, imaging.Lanczos)
			}
		}

		m.Image[i] = palettedAtOrigin(processed, frame.Palette)

		switch m.Disposal[i] {
		case gif.DisposalBackground:
			tmp = image.NewRGBA(b)
		case gif.DisposalPrevious:
			tmp = prev
		}
	}

	// stretch resizes every frame to exactly width x height, so the logical
	// screen must always match the box even when that is larger than the source
	// (webdav sends no_upscale(), which would otherwise skip the update and leave
	// frames extending past the screen, making gif.EncodeAll fail with "image
	// block is out of bounds"). fill/fit-in keep the source size when noUpscale
	// caps them at the source, so they only grow the screen when actually resized.
	if operation == OpStretch || !noUpscale || srcX > width || srcY > height {
		m.Config.Width = width
		m.Config.Height = height
	}

	return m
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
