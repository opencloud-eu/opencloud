//go:build enable_vips

package svc

import (
	"fmt"
	"image/color/palette"
	"image/gif"
	"io"

	"github.com/davidbyttow/govips/v2/vips"
)

func init() {
	processImage = processImageVips
	encodeJPEG = encodeJPEGVips
	encodePNG = encodePNGVips
	encodeGIF = encodeGIFVips
	encodeWebP = encodeWebPVips
}

// processImageVips resizes the input using libvips. The operation selects the
// resize/crop mode: fill (center-crop to the box, upscaling by default like real
// imagor's default resize), fit-in (fit within the box without cropping, upscaling
// small sources by default like real imagor), or stretch (resize to the exact box).
// noUpscale caps the upscaling at the source size, mirroring imagor's no_upscale()
// filter.
func processImageVips(r io.Reader, req resizeRequest) (any, error) {
	if req.KeepAnimation && isGifReader(r) {
		g, err := gif.DecodeAll(r)
		if err == nil && len(g.Image) > 0 {
			return resizeGIF(g, req.Width, req.Height, req.Operation, req.NoUpscale), nil
		}
	}

	imgData, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	m, err := vips.NewImageFromBuffer(imgData)
	if err != nil {
		return nil, err
	}

	// Note: no defer m.Close() here. The ImageRef is returned to the caller and
	// encoded by encodeJPEGVips/encodePNGVips, which own its lifetime and close it
	// after exporting. Closing it here would free the underlying C image before the
	// export runs (use-after-free).

	switch req.Operation {
	case OpStretch:
		// Resize to the exact box without preserving aspect ratio.
		hScale := float64(req.Width) / float64(m.Width())
		vScale := float64(req.Height) / float64(m.Height())
		if err := m.ResizeWithVScale(hScale, vScale, vips.KernelLanczos3); err != nil {
			return nil, err
		}
	case OpFitIn:
		// imagor's fit-in only enlarges with the upscale() filter, which webdav
		// never sends, so it always caps at the source size.
		if err := m.ThumbnailWithSize(req.Width, req.Height, vips.InterestingNone, vips.SizeDown); err != nil {
			return nil, err
		}
	default: // OpFill
		// Center-crop to fill the box exactly. SizeBoth matches real imagor's
		// default resize (which upscales small sources); no_upscale() caps it
		// at the source size via SizeDown.
		size := vips.SizeBoth
		if req.NoUpscale {
			size = vips.SizeDown
		}
		if err := m.ThumbnailWithSize(req.Width, req.Height, vips.InterestingAttention, size); err != nil {
			return nil, err
		}
	}

	if err := m.RemoveMetadata(); err != nil {
		return nil, err
	}

	// Keep the image in libvips form so the encode hooks can export it via
	// ExportJpeg/ExportPng (matching the legacy pipeline's progressive JPEG and
	// quality-80 output) instead of a lossy PNG round-trip + stdlib re-encode.
	return m, nil
}

// encodeJPEGVips exports the processed libvips image as JPEG using libvips's
// default export parameters (progressive/interlaced, quality 80), matching the
// legacy pipeline's output byte-for-byte where the resize operation is unchanged.
func encodeJPEGVips(w io.Writer, processed any) error {
	m, ok := processed.(*vips.ImageRef)
	if !ok {
		return fmt.Errorf("cannot encode %T as jpeg", processed)
	}
	defer m.Close()
	buf, _, err := m.ExportJpeg(vips.NewJpegExportParams())
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}

// encodePNGVips exports the processed libvips image as PNG via libvips.
func encodePNGVips(w io.Writer, processed any) error {
	m, ok := processed.(*vips.ImageRef)
	if !ok {
		return fmt.Errorf("cannot encode %T as png", processed)
	}
	defer m.Close()
	buf, _, err := m.ExportPng(vips.NewPngExportParams())
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}

// encodeGIFVips writes an animated gif as it is; a still libvips image goes
// through the same palette the animated path uses, as libvips has no gif export.
func encodeGIFVips(w io.Writer, processed any) error {
	if g, ok := processed.(*gif.GIF); ok {
		return gif.EncodeAll(w, g)
	}

	m, ok := processed.(*vips.ImageRef)
	if !ok {
		return fmt.Errorf("cannot encode %T as gif", processed)
	}
	defer m.Close()

	img, err := m.ToImage(vips.NewDefaultPNGExportParams())
	if err != nil {
		return err
	}

	return gif.Encode(w, palettedAtOrigin(img, palette.Plan9), nil)
}

// encodeWebPVips exports the processed libvips image as webp.
func encodeWebPVips(w io.Writer, processed any) error {
	m, ok := processed.(*vips.ImageRef)
	if !ok {
		return fmt.Errorf("cannot encode %T as webp", processed)
	}
	defer m.Close()

	buf, _, err := m.ExportWebp(vips.NewWebpExportParams())
	if err != nil {
		return err
	}

	_, err = w.Write(buf)

	return err
}
