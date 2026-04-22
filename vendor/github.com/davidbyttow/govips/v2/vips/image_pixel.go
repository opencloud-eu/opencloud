package vips

// #include "image.h"
import "C"

import (
	"errors"
	"runtime"
)

// GaussianBlur blurs the image
// add support minAmpl
func (r *ImageRef) GaussianBlur(sigmas ...float64) error {
	defer runtime.KeepAlive(r)
	var (
		sigma   = sigmas[0]
		minAmpl = GaussBlurDefaultMinAMpl
	)
	if len(sigmas) >= 2 {
		minAmpl = sigmas[1]
	}
	out, err := vipsGenGaussblur(r.image, sigma, &GaussblurOptions{MinAmpl: &minAmpl})
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Sharpen sharpens the image
// sigma: sigma of the gaussian
// x1: flat/jaggy threshold
// m2: slope for jaggy areas
func (r *ImageRef) Sharpen(sigma float64, x1 float64, m2 float64) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenSharpen(r.image, &SharpenOptions{Sigma: &sigma, X1: &x1, M2: &m2})
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Apply Sobel edge detector to the image.
func (r *ImageRef) Sobel() error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenSobel(r.image)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Rank does rank filtering on an image. A window of size width by height is passed over the image.
// At each position, the pixels inside the window are sorted into ascending order and the pixel at position
// index is output. index numbers from 0.
func (r *ImageRef) Rank(width int, height int, index int) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenRank(r.image, width, height, index)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Mapim resamples an image using index to look up pixels
func (r *ImageRef) Mapim(index *ImageRef) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenMapim(r.image, index.image, nil)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Maplut maps an image through another image acting as a LUT (Look Up Table)
func (r *ImageRef) Maplut(lut *ImageRef) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenMaplut(r.image, lut.image, nil)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// ExtractBand extracts one or more bands out of the image (replacing the associated ImageRef)
func (r *ImageRef) ExtractBand(band int, num int) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenExtractBand(r.image, band, &ExtractBandOptions{N: &num})
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// ExtractBandToImage extracts one or more bands out of the image to a new image
func (r *ImageRef) ExtractBandToImage(band int, num int) (*ImageRef, error) {
	defer runtime.KeepAlive(r)
	out, err := vipsGenExtractBand(r.image, band, &ExtractBandOptions{N: &num})
	if err != nil {
		return nil, err
	}
	return newImageRef(out, ImageTypeUnknown, ImageTypeUnknown, nil), nil
}

// BandJoin joins a set of images together, bandwise.
func (r *ImageRef) BandJoin(images ...*ImageRef) error {
	defer runtime.KeepAlive(r)
	vipsImages := []*C.VipsImage{r.image}
	for _, vipsImage := range images {
		vipsImages = append(vipsImages, vipsImage.image)
	}

	out, err := vipsGenBandjoin(vipsImages)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// BandSplit split an n-band image into n separate images..
func (r *ImageRef) BandSplit() ([]*ImageRef, error) {
	defer runtime.KeepAlive(r)
	var out []*ImageRef
	n := 1
	for i := 0; i < r.Bands(); i++ {
		img, err := vipsGenExtractBand(r.image, i, &ExtractBandOptions{N: &n})
		if err != nil {
			return out, err
		}
		out = append(out, &ImageRef{image: img})
	}
	return out, nil
}

// BandJoinConst appends a set of constant bands to an image.
func (r *ImageRef) BandJoinConst(constants []float64) error {
	defer runtime.KeepAlive(r)
	if len(constants) == 0 {
		return errors.New("BandJoinConst: empty constants slice")
	}
	out, err := vipsGenBandjoinConst(r.image, constants)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// AddAlpha adds an alpha channel to the associated image.
func (r *ImageRef) AddAlpha() error {
	defer runtime.KeepAlive(r)
	if vipsHasAlpha(r.image) {
		return nil
	}

	out, err := vipsAddAlpha(r.image)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// PremultiplyAlpha premultiplies the alpha channel.
// See https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-premultiply
func (r *ImageRef) PremultiplyAlpha() error {
	defer runtime.KeepAlive(r)
	if r.preMultiplication != nil || !vipsHasAlpha(r.image) {
		return nil
	}

	band := r.BandFormat()

	out, err := vipsGenPremultiply(r.image, nil)
	if err != nil {
		return err
	}
	r.preMultiplication = &PreMultiplicationState{
		bandFormat: band,
	}
	r.setImage(out)
	return nil
}

// UnpremultiplyAlpha unpremultiplies any alpha channel.
// See https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-unpremultiply
func (r *ImageRef) UnpremultiplyAlpha() error {
	defer runtime.KeepAlive(r)
	if r.preMultiplication == nil {
		return nil
	}

	unpremultiplied, err := vipsGenUnpremultiply(r.image, nil)
	if err != nil {
		return err
	}
	defer clearImage(unpremultiplied)

	out, err := vipsGenCast(unpremultiplied, r.preMultiplication.bandFormat, nil)
	if err != nil {
		return err
	}

	r.preMultiplication = nil
	r.setImage(out)
	return nil
}

// Add calculates a sum of the image + addend and stores it back in the image
func (r *ImageRef) Add(addend *ImageRef) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenAdd(r.image, addend.image)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Multiply calculates the product of the image * multiplier and stores it back in the image
func (r *ImageRef) Multiply(multiplier *ImageRef) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenMultiply(r.image, multiplier.image)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Divide calculates the product of the image / denominator and stores it back in the image
func (r *ImageRef) Divide(denominator *ImageRef) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenDivide(r.image, denominator.image)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// Average finds the average value in an image
func (r *ImageRef) Average() (float64, error) {
	defer runtime.KeepAlive(r)
	out, err := vipsGenAvg(r.image)
	if err != nil {
		return 0, err
	}
	return out, nil
}

// FindTrim returns the bounding box of the non-border part of the image
// Returned values are left, top, width, height
func (r *ImageRef) FindTrim(threshold float64, backgroundColor *Color) (int, int, int, int, error) {
	defer runtime.KeepAlive(r)
	return vipsFindTrim(r.image, threshold, backgroundColor)
}

// GetPoint reads a single pixel on an image.
// The pixel values are returned in a slice of length n.
func (r *ImageRef) GetPoint(x int, y int) ([]float64, error) {
	defer runtime.KeepAlive(r)
	n := 3
	if vipsHasAlpha(r.image) {
		n = 4
	}
	return vipsGetPoint(r.image, n, x, y)
}

// Stats find many image statistics in a single pass through the data. Image is changed into a one-band
// `BandFormatDouble` image of at least 10 columns by n + 1 (where n is number of bands in image in)
// rows. Columns are statistics, and are, in order: minimum, maximum, sum, sum of squares, mean,
// standard deviation, x coordinate of minimum, y coordinate of minimum, x coordinate of maximum,
// y coordinate of maximum.
//
// Row 0 has statistics for all bands together, row 1 has stats for band 1, and so on.
//
// If there is more than one maxima or minima, one of them will be chosen at random.
func (r *ImageRef) Stats() error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenStats(r.image)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// HistogramFind find the histogram the image.
// Find the histogram for all bands (producing a one-band histogram).
// char and uchar images are cast to uchar before histogramming, all other image types are cast to ushort.
func (r *ImageRef) HistogramFind() error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenHistFind(r.image, nil)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// HistogramCumulative form cumulative histogram.
func (r *ImageRef) HistogramCumulative() error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenHistCum(r.image)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// HistogramNormalise
// The maximum of each band becomes equal to the maximum index, so for example the max for a uchar
// image becomes 255. Normalise each band separately.
func (r *ImageRef) HistogramNormalise() error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenHistNorm(r.image)
	if err != nil {
		return err
	}
	r.setImage(out)
	return nil
}

// HistogramEntropy estimate image entropy from a histogram. Entropy is calculated as:
// `-sum(p * log2(p))`
// where p is histogram-value / sum-of-histogram-values.
func (r *ImageRef) HistogramEntropy() (float64, error) {
	defer runtime.KeepAlive(r)
	return vipsGenHistEntropy(r.image)
}

// DrawRect draws an (optionally filled) rectangle with a single colour
func (r *ImageRef) DrawRect(ink ColorRGBA, left int, top int, width int, height int, fill bool) error {
	defer runtime.KeepAlive(r)
	err := vipsDrawRect(r.image, ink, left, top, width, height, fill)
	if err != nil {
		return err
	}
	return nil
}

// Subtract calculate subtract operation between two images.
func (r *ImageRef) Subtract(in2 *ImageRef) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenSubtract(r.image, in2.image)
	if err != nil {
		return err
	}

	r.setImage(out)
	return nil
}

// Abs calculate abs operation.
func (r *ImageRef) Abs() error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenAbs(r.image)
	if err != nil {
		return err
	}

	r.setImage(out)
	return nil
}

// Project calculate project operation.
func (r *ImageRef) Project() (*ImageRef, *ImageRef, error) {
	defer runtime.KeepAlive(r)
	col, row, err := vipsGenProject(r.image)
	if err != nil {
		return nil, nil, err
	}

	return newImageRef(col, r.format, r.originalFormat, nil), newImageRef(row, r.format, r.originalFormat, nil), nil
}

// Min finds the minimum value in an image.
func (r *ImageRef) Min() (float64, int, int, error) {
	defer runtime.KeepAlive(r)
	return vipsMin(r.image)
}
