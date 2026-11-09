package svc

import (
	"image"
	"image/draw"
	"image/gif"

	"github.com/kovidgoyal/imaging"
)

// resizeGIF resizes every frame of an animated gif, compositing each frame onto
// a running canvas to honour the gif disposal method and re-palletting with
// Floyd-Steinberg dithering. Both backends share it: the frames go through
// imaging either way, because neither libvips nor the pure Go path has anything
// better for a paletted animation. Code adapted from
// https://github.com/willnorris/gifresize. The only error is errBoxTooLarge.
func resizeGIF(m *gif.GIF, req resizeRequest) (*gif.GIF, error) {
	srcX, srcY := m.Config.Width, m.Config.Height
	width, height, operation, noUpscale := req.Width, req.Height, req.Operation, req.NoUpscale

	if width == 0 || height == 0 {
		resolvedW, resolvedH, resize, err := req.freeAxis(srcX, srcY)
		if err != nil {
			return nil, err
		}
		if !resize {
			return m, nil
		}
		width, height, operation = resolvedW, resolvedH, OpStretch
	}

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
			processed = fitIn(tmp, width, height)
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

	// the logical screen is what the frames are, or gif.EncodeAll refuses a frame
	// that extends past it
	if len(m.Image) > 0 {
		b := m.Image[0].Bounds()
		m.Config.Width, m.Config.Height = b.Dx(), b.Dy()
	}

	return m, nil
}
