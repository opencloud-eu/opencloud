//go:build enable_vips

package svc

import (
	"image/color/palette"
	"image/gif"
	"io"

	"github.com/davidbyttow/govips/v2/vips"
)

// The backend this build resizes with. The implementations keep their own names
// so a symbol search finds both builds' versions side by side, the variables are
// what the shared handler calls.
//
// libvips is demand-driven: loading a buffer parses the header and computes
// pixels only when something pulls them, so the dimension guard has its say
// before any frame buffer exists. The resize works on the loaded image in place,
// so what an encoder gets is the same ImageRef, and the handler owns it until
// the encode is done.
var (
	loadSource         = loadSourceVips
	extForSourceFormat = extForSourceFormatVips
	resizeSource       = resizeSourceVips
	encodeJPEG         = encodeJPEGVips
	encodePNG          = encodePNGVips
	encodeGIF          = encodeGIFVips
	encodeWebP         = encodeWebPVips
)

// loadSourceVips reads the header through libvips, which this build also decodes
// with, so whatever the deployed libvips reads is what this service accepts.
func loadSourceVips(data []byte) (*vips.ImageRef, error) { return vips.NewImageFromBuffer(data) }

// extForSourceFormatVips is the output format for a request that asked for none: the
// input's own where this build can write it, a JPEG for the rest, which is how
// imagor answers a format it cannot write either.
func extForSourceFormatVips(src *vips.ImageRef) string {
	switch src.Format() {
	case vips.ImageTypePNG:
		return "png"
	case vips.ImageTypeGIF:
		return "gif"
	case vips.ImageTypeWEBP:
		return "webp"
	default: // jpeg, tiff, bmp, heif, ...
		return "jpg"
	}
}

// resizeSourceVips resizes with libvips. The operation selects the resize/crop mode:
// fill (center-crop to the box, upscaling by default like real imagor's default
// resize), fit-in (fit within the box without cropping and without ever enlarging
// the source, like imagor without its upscale() filter), or stretch (resize to
// the exact box). noUpscale caps the default fill at the source size, mirroring
// imagor's no_upscale() filter.
func resizeSourceVips(src *vips.ImageRef, req resizeRequest) (*vips.ImageRef, error) {
	if req.Width == 0 || req.Height == 0 {
		boxW, boxH, resize := autoAxis(src.Width(), src.Height(), req.Width, req.Height)
		if resize {
			if err := scaleTo(src, boxW, boxH); err != nil {
				return nil, err
			}
		}

		return src, src.RemoveMetadata()
	}

	switch req.Operation {
	case OpStretch:
		if err := scaleTo(src, req.Width, req.Height); err != nil {
			return nil, err
		}
	case OpFitIn:
		// imagor's fit-in only enlarges with the upscale() filter, which webdav
		// never sends, so it always caps at the source size.
		if err := src.ThumbnailWithSize(req.Width, req.Height, vips.InterestingNone, vips.SizeDown); err != nil {
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
		if err := src.ThumbnailWithSize(req.Width, req.Height, vips.InterestingAttention, size); err != nil {
			return nil, err
		}
	}

	return src, src.RemoveMetadata()
}

// scaleTo resizes to the exact box without preserving the aspect ratio.
func scaleTo(src *vips.ImageRef, width, height int) error {
	hScale := float64(width) / float64(src.Width())
	vScale := float64(height) / float64(src.Height())

	return src.ResizeWithVScale(hScale, vScale, vips.KernelLanczos3)
}

// The encoders keep the image in libvips form up to here, so a JPEG comes out of
// libvips (progressive, quality 80) instead of a stdlib re-encode.
func encodeJPEGVips(w io.Writer, img *vips.ImageRef) error {
	buf, _, err := img.ExportJpeg(vips.NewJpegExportParams())
	if err != nil {
		return err
	}
	_, err = w.Write(buf)

	return err
}

func encodePNGVips(w io.Writer, img *vips.ImageRef) error {
	buf, _, err := img.ExportPng(vips.NewPngExportParams())
	if err != nil {
		return err
	}
	_, err = w.Write(buf)

	return err
}

// encodeGIFVips goes through the same palette the animated path uses, as libvips has
// no gif export.
func encodeGIFVips(w io.Writer, img *vips.ImageRef) error {
	still, err := img.ToImage(vips.NewDefaultPNGExportParams())
	if err != nil {
		return err
	}

	return gif.Encode(w, palettedAtOrigin(still, palette.Plan9), nil)
}

func encodeWebPVips(w io.Writer, img *vips.ImageRef) error {
	buf, _, err := img.ExportWebp(vips.NewWebpExportParams())
	if err != nil {
		return err
	}
	_, err = w.Write(buf)

	return err
}
