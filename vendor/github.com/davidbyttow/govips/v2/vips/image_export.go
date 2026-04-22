package vips

// #include "image.h"
import "C"

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"runtime"
	"unsafe"
)

// Export creates a byte array of the image for use.
// The function returns a byte array that can be written to a file e.g. via os.WriteFile().
// N.B. govips does not currently have built-in support for directly exporting to a file.
// The function also returns a copy of the image metadata as well as an error.
// Deprecated: Use ExportNative or format-specific Export methods
func (r *ImageRef) Export(params *ExportParams) ([]byte, *ImageMetadata, error) {
	if params == nil || params.Format == ImageTypeUnknown {
		return r.ExportNative()
	}

	format := params.Format

	if !IsTypeSupported(format) {
		return nil, r.newMetadata(ImageTypeUnknown), fmt.Errorf("cannot save to %#v", ImageTypes[format])
	}

	switch format {
	case ImageTypeGIF:
		return r.ExportGIF(&GifExportParams{
			Quality: params.Quality,
		})
	case ImageTypeWEBP:
		return r.ExportWebp(&WebpExportParams{
			StripMetadata:   params.StripMetadata,
			Quality:         params.Quality,
			Lossless:        params.Lossless,
			ReductionEffort: params.Effort,
		})
	case ImageTypePNG:
		return r.ExportPng(&PngExportParams{
			StripMetadata: params.StripMetadata,
			Compression:   params.Compression,
			Interlace:     params.Interlaced,
		})
	case ImageTypeTIFF:
		compression := TiffCompressionLzw
		if params.Lossless {
			compression = TiffCompressionNone
		}
		return r.ExportTiff(&TiffExportParams{
			StripMetadata: params.StripMetadata,
			Quality:       params.Quality,
			Compression:   compression,
		})
	case ImageTypeHEIF:
		return r.ExportHeif(&HeifExportParams{
			Quality:  params.Quality,
			Lossless: params.Lossless,
		})
	case ImageTypeAVIF:
		return r.ExportAvif(&AvifExportParams{
			StripMetadata: params.StripMetadata,
			Quality:       params.Quality,
			Lossless:      params.Lossless,
			Speed:         params.Speed,
		})
	case ImageTypeJXL:
		return r.ExportJxl(&JxlExportParams{
			Quality:  params.Quality,
			Lossless: params.Lossless,
			Effort:   params.Effort,
		})
	default:
		format = ImageTypeJPEG
		return r.ExportJpeg(&JpegExportParams{
			Quality:            params.Quality,
			StripMetadata:      params.StripMetadata,
			Interlace:          params.Interlaced,
			OptimizeCoding:     params.OptimizeCoding,
			SubsampleMode:      params.SubsampleMode,
			TrellisQuant:       params.TrellisQuant,
			OvershootDeringing: params.OvershootDeringing,
			OptimizeScans:      params.OptimizeScans,
			QuantTable:         params.QuantTable,
		})
	}
}

// ExportNative exports the image to a buffer based on its native format with default parameters.
func (r *ImageRef) ExportNative() ([]byte, *ImageMetadata, error) {
	switch r.format {
	case ImageTypeJPEG:
		return r.ExportJpeg(NewJpegExportParams())
	case ImageTypePNG:
		return r.ExportPng(NewPngExportParams())
	case ImageTypeWEBP:
		return r.ExportWebp(NewWebpExportParams())
	case ImageTypeHEIF:
		return r.ExportHeif(NewHeifExportParams())
	case ImageTypeTIFF:
		return r.ExportTiff(NewTiffExportParams())
	case ImageTypeAVIF:
		return r.ExportAvif(NewAvifExportParams())
	case ImageTypeJP2K:
		return r.ExportJp2k(NewJp2kExportParams())
	case ImageTypeGIF:
		return r.ExportGIF(NewGifExportParams())
	case ImageTypeJXL:
		return r.ExportJxl(NewJxlExportParams())
	default:
		return r.ExportJpeg(NewJpegExportParams())
	}
}

// ExportJpeg exports the image as JPEG to a buffer.
func (r *ImageRef) ExportJpeg(params *JpegExportParams) ([]byte, *ImageMetadata, error) {
	defer runtime.KeepAlive(r)
	if params == nil {
		params = NewJpegExportParams()
	}

	buf, err := vipsSaveJPEGToBuffer(r.image, *params)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.newMetadata(ImageTypeJPEG), nil
}

// ExportPng exports the image as PNG to a buffer.
func (r *ImageRef) ExportPng(params *PngExportParams) ([]byte, *ImageMetadata, error) {
	defer runtime.KeepAlive(r)
	if params == nil {
		params = NewPngExportParams()
	}

	buf, err := vipsSavePNGToBuffer(r.image, *params)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.newMetadata(ImageTypePNG), nil
}

// ExportWebp exports the image as WEBP to a buffer.
func (r *ImageRef) ExportWebp(params *WebpExportParams) ([]byte, *ImageMetadata, error) {
	defer runtime.KeepAlive(r)
	if params == nil {
		params = NewWebpExportParams()
	}

	paramsWithIccProfile := *params
	paramsWithIccProfile.IccProfile = r.optimizedIccProfile

	buf, err := vipsSaveWebPToBuffer(r.image, paramsWithIccProfile)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.newMetadata(ImageTypeWEBP), nil
}

// ExportHeif exports the image as HEIF to a buffer.
func (r *ImageRef) ExportHeif(params *HeifExportParams) ([]byte, *ImageMetadata, error) {
	defer runtime.KeepAlive(r)
	if params == nil {
		params = NewHeifExportParams()
	}

	buf, err := vipsSaveHEIFToBuffer(r.image, *params)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.newMetadata(ImageTypeHEIF), nil
}

// ExportTiff exports the image as TIFF to a buffer.
func (r *ImageRef) ExportTiff(params *TiffExportParams) ([]byte, *ImageMetadata, error) {
	defer runtime.KeepAlive(r)
	if params == nil {
		params = NewTiffExportParams()
	}

	buf, err := vipsSaveTIFFToBuffer(r.image, *params)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.newMetadata(ImageTypeTIFF), nil
}

// ExportGIF exports the image as GIF to a buffer.
func (r *ImageRef) ExportGIF(params *GifExportParams) ([]byte, *ImageMetadata, error) {
	defer runtime.KeepAlive(r)
	if params == nil {
		params = NewGifExportParams()
	}

	buf, err := vipsSaveGIFToBuffer(r.image, *params)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.newMetadata(ImageTypeGIF), nil
}

// ExportAvif exports the image as AVIF to a buffer.
func (r *ImageRef) ExportAvif(params *AvifExportParams) ([]byte, *ImageMetadata, error) {
	defer runtime.KeepAlive(r)
	if params == nil {
		params = NewAvifExportParams()
	}

	buf, err := vipsSaveAVIFToBuffer(r.image, *params)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.newMetadata(ImageTypeAVIF), nil
}

// ExportJp2k exports the image as JPEG2000 to a buffer.
func (r *ImageRef) ExportJp2k(params *Jp2kExportParams) ([]byte, *ImageMetadata, error) {
	defer runtime.KeepAlive(r)
	if params == nil {
		params = NewJp2kExportParams()
	}

	buf, err := vipsSaveJP2KToBuffer(r.image, *params)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.newMetadata(ImageTypeJP2K), nil
}

// ExportJxl exports the image as JPEG XL to a buffer.
func (r *ImageRef) ExportJxl(params *JxlExportParams) ([]byte, *ImageMetadata, error) {
	defer runtime.KeepAlive(r)
	if params == nil {
		params = NewJxlExportParams()
	}

	buf, err := vipsSaveJxlToBuffer(r.image, *params)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.newMetadata(ImageTypeJXL), nil
}

// ExportMagick exports the image as Format set in param to a buffer.
func (r *ImageRef) ExportMagick(params *MagickExportParams) ([]byte, *ImageMetadata, error) {
	defer runtime.KeepAlive(r)
	if params == nil {
		params = NewMagickExportParams()
		params.Format = "JPG"
	}

	buf, err := vipsSaveMagickToBuffer(r.image, *params)
	if err != nil {
		return nil, nil, err
	}

	return buf, r.newMetadata(ImageTypeMagick), nil
}

// ToBytes writes the image to memory in VIPs format and returns the raw bytes, useful for storage.
func (r *ImageRef) ToBytes() ([]byte, error) {
	defer runtime.KeepAlive(r)
	var cSize C.size_t
	cData := C.vips_image_write_to_memory(r.image, &cSize)
	if cData == nil {
		return nil, errors.New("failed to write image to memory")
	}
	defer C.free(cData)

	data := C.GoBytes(unsafe.Pointer(cData), C.int(cSize))
	return data, nil
}

// ToImage converts a VIPs image to a golang image.Image object, useful for interoperability with other golang libraries.
// Deprecated: Use ToGoImage for a faster, direct conversion without encoding round-trip.
func (r *ImageRef) ToImage(params *ExportParams) (image.Image, error) {
	imageBytes, _, err := r.ExportNative()
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(imageBytes)
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// ToGoImage converts a vips image directly to a Go image.Image without encoding.
// This is significantly faster than ToImage() which round-trips through JPEG/PNG.
// The resulting image will be in sRGB color space with 8-bit depth.
func (r *ImageRef) ToGoImage() (image.Image, error) {
	defer runtime.KeepAlive(r)

	// Work on a copy to avoid mutating the receiver
	tmp, err := vipsGenCopy(r.image, nil)
	if err != nil {
		return nil, err
	}
	defer clearImage(tmp)

	// Convert to sRGB if needed (keep B_W for grayscale)
	interp := Interpretation(int(tmp.Type))
	if interp != InterpretationSRGB && interp != InterpretationBW {
		out, err := vipsToColorSpace(tmp, InterpretationSRGB)
		if err != nil {
			return nil, err
		}
		clearImage(tmp)
		tmp = out
	}

	// Cast to uchar if needed
	if BandFormat(int(tmp.BandFmt)) != BandFormatUchar {
		out, err := vipsGenCast(tmp, BandFormatUchar, nil)
		if err != nil {
			return nil, err
		}
		clearImage(tmp)
		tmp = out
	}

	// Extract raw pixel data
	var cSize C.size_t
	cData := C.vips_image_write_to_memory(tmp, &cSize)
	if cData == nil {
		return nil, errors.New("failed to write image to memory")
	}
	defer C.free(cData)

	width := int(tmp.Xsize)
	height := int(tmp.Ysize)
	bands := int(tmp.Bands)
	pixels := C.GoBytes(unsafe.Pointer(cData), C.int(cSize))

	switch bands {
	case 1:
		img := image.NewGray(image.Rect(0, 0, width, height))
		copy(img.Pix, pixels)
		return img, nil
	case 2:
		// Grayscale + alpha
		img := image.NewNRGBA(image.Rect(0, 0, width, height))
		srcIdx := 0
		dstIdx := 0
		for srcIdx+1 < len(pixels) {
			v := pixels[srcIdx]
			img.Pix[dstIdx] = v
			img.Pix[dstIdx+1] = v
			img.Pix[dstIdx+2] = v
			img.Pix[dstIdx+3] = pixels[srcIdx+1]
			srcIdx += 2
			dstIdx += 4
		}
		return img, nil
	case 3:
		// RGB, add opaque alpha
		img := image.NewNRGBA(image.Rect(0, 0, width, height))
		srcIdx := 0
		dstIdx := 0
		for srcIdx+2 < len(pixels) {
			img.Pix[dstIdx] = pixels[srcIdx]
			img.Pix[dstIdx+1] = pixels[srcIdx+1]
			img.Pix[dstIdx+2] = pixels[srcIdx+2]
			img.Pix[dstIdx+3] = 255
			srcIdx += 3
			dstIdx += 4
		}
		return img, nil
	case 4:
		img := image.NewNRGBA(image.Rect(0, 0, width, height))
		copy(img.Pix, pixels)
		return img, nil
	default:
		return nil, fmt.Errorf("unsupported number of bands: %d", bands)
	}
}
