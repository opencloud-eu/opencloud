//go:build !enable_vips

package svc

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/kovidgoyal/imaging"
)

func init() {
	processImage = processImageImaging
	encodeJPEG   = encodeJPEGImaging
	encodePNG    = encodePNGImaging
}

// processImageImaging resizes the input using the imaging backend. The operation
// selects the resize/crop mode: fill (center-crop to the box), fit-in (fit within
// the box without cropping, never upscaling), or stretch (resize to the exact box).
func processImageImaging(r io.Reader, width, height int, operation string) (any, image.Rectangle, error) {
	if isGifReader(r) {
		g, err := gif.DecodeAll(r)
		if err == nil && len(g.Image) > 0 {
			srcBounds := g.Image[0].Bounds()
			return resizeGIF(g, width, height, operation), srcBounds, nil
		}
	}

	img, err := imaging.Decode(r, imaging.AutoOrientation(true))
	if err != nil {
		return nil, image.Rectangle{}, err
	}

	srcBounds := img.Bounds()

	switch operation {
	case OpStretch:
		return imaging.Resize(img, width, height, imaging.Lanczos), srcBounds, nil
	case OpFitIn:
		if srcBounds.Dx() > width || srcBounds.Dy() > height {
			return imaging.Fit(img, width, height, imaging.Lanczos), srcBounds, nil
		}
		return img, srcBounds, nil
	default: // OpFill
		return imaging.Thumbnail(img, width, height, imaging.Lanczos), srcBounds, nil
	}
}

// resizeGIF resizes every frame of an animated gif while preserving the
// animation. It composites each frame onto a running canvas honoring the gif
// disposal method, resizes with the requested processor, and re-pallettes the
// result using Floyd-Steinberg dithering. Code adapted from
// https://github.com/willnorris/gifresize.
func resizeGIF(m *gif.GIF, width, height int, operation string) *gif.GIF {
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
			if srcX > width || srcY > height {
				processed = imaging.Fit(tmp, width, height, imaging.Lanczos)
			} else {
				processed = tmp
			}
		default: // OpFill
			processed = imaging.Fill(tmp, width, height, imaging.Center, imaging.Lanczos)
		}

		m.Image[i] = imageToPaletted(processed, frame.Palette)

		switch m.Disposal[i] {
		case gif.DisposalBackground:
			tmp = image.NewRGBA(b)
		case gif.DisposalPrevious:
			tmp = prev
		}
	}

	m.Config.Width = width
	m.Config.Height = height

	return m
}

func imageToPaletted(img image.Image, p color.Palette) *image.Paletted {
	b := img.Bounds()
	pm := image.NewPaletted(b, p)
	draw.FloydSteinberg.Draw(pm, b, img, image.Point{})
	return pm
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
