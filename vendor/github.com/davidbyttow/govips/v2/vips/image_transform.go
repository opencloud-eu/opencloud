package vips

// #include "image.h"
import "C"

import (
	"errors"
	"math"
	"runtime"
	"unsafe"
)

// GetRotationAngleFromExif returns the angle which the image is currently rotated in.
// First returned value is the angle and second is a boolean indicating whether image is flipped.
// This is based on the EXIF orientation tag standard.
// If no proper orientation number is provided, the picture will be assumed to be upright.
func GetRotationAngleFromExif(orientation int) (Angle, bool) {
	switch orientation {
	case 0, 1, 2:
		return Angle0, orientation == 2
	case 3, 4:
		return Angle180, orientation == 4
	case 5, 8:
		return Angle90, orientation == 5
	case 6, 7:
		return Angle270, orientation == 7
	}

	return Angle0, false
}

// AutoRotate rotates the image upright based on the EXIF Orientation tag.
// It also resets the orientation information in the EXIF tag to be 1 (i.e. upright).
// N.B. libvips does not flip images currently (i.e. no support for orientations 2, 4, 5 and 7).
// N.B. due to the HEIF image standard, HEIF images are always autorotated by default on load.
// Thus, calling AutoRotate for HEIF images is not needed.
// todo: use https://www.libvips.org/API/current/libvips-conversion.html#vips-autorot-remove-angle
func (r *ImageRef) AutoRotate() error {
	defer runtime.KeepAlive(r)
	out, _, _, err := vipsGenAutorot(r.image)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// ExtractArea crops the image to a specified area
func (r *ImageRef) ExtractArea(left, top, width, height int) error {
	defer runtime.KeepAlive(r)
	if r.Height() > r.PageHeight() {
		// use animated extract area if more than 1 pages loaded
		out, err := vipsExtractAreaMultiPage(r.image, left, top, width, height)
		if err != nil {
			return err
		}
		r.setImage(out)
	} else {
		out, err := vipsExtractArea(r.image, left, top, width, height)
		if err != nil {
			return err
		}
		r.setImage(out)
	}
	return nil
}

// Resize resizes the image based on the scale, maintaining aspect ratio
func (r *ImageRef) Resize(scale float64, kernel Kernel) error {
	return r.ResizeWithVScale(scale, -1, kernel)
}

// ResizeWithVScale resizes the image with both horizontal and vertical scaling.
// The parameters are the scaling factors.
func (r *ImageRef) ResizeWithVScale(hScale, vScale float64, kernel Kernel) error {
	defer runtime.KeepAlive(r)
	if err := r.PremultiplyAlpha(); err != nil {
		return err
	}

	pages := r.Pages()
	pageHeight := r.PageHeight()

	out, err := vipsResizeWithVScale(r.image, hScale, vScale, kernel)
	if err != nil {
		return err
	}
	r.setImage(out)

	if pages > 1 {
		scale := hScale
		if vScale != -1 {
			scale = vScale
		}
		newPageHeight := int(math.Round(float64(pageHeight) * scale))
		if err := r.SetPageHeight(newPageHeight); err != nil {
			return err
		}
	}

	return r.UnpremultiplyAlpha()
}

// Thumbnail resizes the image to the given width and height.
// crop decides algorithm vips uses to shrink and crop to fill target,
func (r *ImageRef) Thumbnail(width, height int, crop Interesting) error {
	defer runtime.KeepAlive(r)
	out, err := vipsThumbnail(r.image, width, height, crop, SizeBoth)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// ThumbnailWithSize resizes the image to the given width and height.
// crop decides algorithm vips uses to shrink and crop to fill target,
// size controls upsize, downsize, both or force
func (r *ImageRef) ThumbnailWithSize(width, height int, crop Interesting, size Size) error {
	defer runtime.KeepAlive(r)
	out, err := vipsThumbnail(r.image, width, height, crop, size)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Embed embeds the given picture in a new one, i.e. the opposite of ExtractArea
func (r *ImageRef) Embed(left, top, width, height int, extend ExtendStrategy) error {
	defer runtime.KeepAlive(r)
	if r.Height() > r.PageHeight() {
		out, err := vipsEmbedMultiPage(r.image, left, top, width, height, extend)
		if err != nil {
			return err
		}
		r.setImage(out)
	} else {
		out, err := vipsEmbed(r.image, left, top, width, height, extend)
		if err != nil {
			return err
		}
		r.setImage(out)
	}
	return nil
}

// EmbedBackground embeds the given picture with a background color
func (r *ImageRef) EmbedBackground(left, top, width, height int, backgroundColor *Color) error {
	defer runtime.KeepAlive(r)
	c := &ColorRGBA{
		R: backgroundColor.R,
		G: backgroundColor.G,
		B: backgroundColor.B,
		A: 255,
	}
	if r.Height() > r.PageHeight() {
		out, err := vipsEmbedMultiPageBackground(r.image, left, top, width, height, c)
		if err != nil {
			return err
		}
		r.setImage(out)
	} else {
		out, err := vipsEmbedBackground(r.image, left, top, width, height, c)
		if err != nil {
			return err
		}
		r.setImage(out)
	}
	return nil
}

// EmbedBackgroundRGBA embeds the given picture with a background rgba color
func (r *ImageRef) EmbedBackgroundRGBA(left, top, width, height int, backgroundColor *ColorRGBA) error {
	defer runtime.KeepAlive(r)
	if r.Height() > r.PageHeight() {
		out, err := vipsEmbedMultiPageBackground(r.image, left, top, width, height, backgroundColor)
		if err != nil {
			return err
		}
		r.setImage(out)
	} else {
		out, err := vipsEmbedBackground(r.image, left, top, width, height, backgroundColor)
		if err != nil {
			return err
		}
		r.setImage(out)
	}
	return nil
}

// Zoom zooms the image by repeating pixels (fast nearest-neighbour)
func (r *ImageRef) Zoom(xFactor int, yFactor int) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenZoom(r.image, xFactor, yFactor)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

func (r *ImageRef) Gravity(gravity Gravity, width int, height int) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenGravity(r.image, gravity, width, height, nil)
	if err != nil {
		return err
	}

	r.setImage(out)
	return nil
}

// Flip flips the image either horizontally or vertically based on the parameter
func (r *ImageRef) Flip(direction Direction) error {
	defer runtime.KeepAlive(r)
	out, err := vipsFlip(r.image, direction)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Recomb recombines the image bands using the matrix provided
func (r *ImageRef) Recomb(matrix [][]float64) error {
	defer runtime.KeepAlive(r)
	numBands := r.Bands()
	// Ensure the provided matrix is 3x3
	if len(matrix) != 3 || len(matrix[0]) != 3 || len(matrix[1]) != 3 || len(matrix[2]) != 3 {
		return errors.New("Invalid recombination matrix")
	}
	// If the image is RGBA, expand the matrix to 4x4
	if numBands == 4 {
		matrix = append(matrix, []float64{0, 0, 0, 1})
		for i := 0; i < 3; i++ {
			matrix[i] = append(matrix[i], 0)
		}
	} else if numBands != 3 {
		return errors.New("Unsupported number of bands")
	}

	// Flatten the matrix
	matrixValues := make([]float64, 0, numBands*numBands)
	for _, row := range matrix {
		for _, value := range row {
			matrixValues = append(matrixValues, value)
		}
	}

	// Convert the Go slice to a C array and get its size
	matrixPtr := unsafe.Pointer(&matrixValues[0])
	matrixSize := C.size_t(len(matrixValues) * 8) // 8 bytes for each float64

	// Create a VipsImage from the matrix in memory
	matrixImage := C.vips_image_new_from_memory(matrixPtr, matrixSize, C.int(numBands), C.int(numBands), 1, C.VIPS_FORMAT_DOUBLE)
	defer clearImage(matrixImage)

	if matrixImage == nil {
		return handleVipsError()
	}

	// Recombine the image using the matrix
	out, err := vipsGenRecomb(r.image, matrixImage)
	runtime.KeepAlive(matrixValues)
	if err != nil {
		return err
	}

	r.setImage(out)
	return nil
}

// Rotate rotates the image by multiples of 90 degrees. To rotate by arbitrary angles use Similarity.
func (r *ImageRef) Rotate(angle Angle) error {
	defer runtime.KeepAlive(r)
	width := r.Width()

	if r.Pages() > 1 && (angle == Angle90 || angle == Angle270) {
		if angle == Angle270 {
			if err := r.Flip(DirectionHorizontal); err != nil {
				return err
			}
		}

		if err := r.Grid(r.PageHeight(), r.Pages(), 1); err != nil {
			return err
		}

		if angle == Angle270 {
			if err := r.Flip(DirectionHorizontal); err != nil {
				return err
			}
		}

	}

	out, err := vipsGenRot(r.image, angle)
	if err != nil {
		return err
	}
	r.setImage(out)

	if r.Pages() > 1 && (angle == Angle90 || angle == Angle270) {
		if err := r.SetPageHeight(width); err != nil {
			return err
		}
	}
	return nil
}

// Similarity lets you scale, offset and rotate images by arbitrary angles in a single operation while defining the
// color of new background pixels. If the input image has no alpha channel, the alpha on `backgroundColor` will be
// ignored. You can add an alpha channel to an image with `BandJoinConst` (e.g. `img.BandJoinConst([]float64{255})`) or
// AddAlpha.
func (r *ImageRef) Similarity(scale float64, angle float64, backgroundColor *ColorRGBA,
	idx float64, idy float64, odx float64, ody float64) error {
	defer runtime.KeepAlive(r)
	out, err := vipsSimilarity(r.image, scale, angle, backgroundColor, idx, idy, odx, ody)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Grid tiles the image pages into a matrix across*down
func (r *ImageRef) Grid(tileHeight, across, down int) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenGrid(r.image, tileHeight, across, down)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// SmartCrop will crop the image based on interesting factor
func (r *ImageRef) SmartCrop(width int, height int, interesting Interesting) error {
	defer runtime.KeepAlive(r)
	out, err := vipsSmartCrop(r.image, width, height, interesting)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Crop will crop the image based on coordinate and box size
func (r *ImageRef) Crop(left int, top int, width int, height int) error {
	defer runtime.KeepAlive(r)
	out, err := vipsCrop(r.image, left, top, width, height)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Label overlays a label on top of the image
func (r *ImageRef) Label(labelParams *LabelParams) error {
	defer runtime.KeepAlive(r)
	out, err := labelImage(r.image, labelParams)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Replicate repeats an image many times across and down
func (r *ImageRef) Replicate(across int, down int) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenReplicate(r.image, across, down)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Pixelate applies a simple pixelate filter to the image
func Pixelate(imageRef *ImageRef, factor float64) (err error) {
	if factor < 1 {
		return errors.New("factor must be greater then 1")
	}

	width := imageRef.Width()
	height := imageRef.Height()

	if err = imageRef.Resize(1/factor, KernelAuto); err != nil {
		return
	}

	hScale := float64(width) / float64(imageRef.Width())
	vScale := float64(height) / float64(imageRef.Height())
	if err = imageRef.ResizeWithVScale(hScale, vScale, KernelNearest); err != nil {
		return
	}

	return
}
