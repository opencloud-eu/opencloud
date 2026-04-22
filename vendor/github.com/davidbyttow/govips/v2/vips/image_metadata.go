package vips

// #include "image.h"
import "C"

import (
	"runtime"
	"strings"
)

// Format returns the current format of the vips image.
func (r *ImageRef) Format() ImageType {
	return r.format
}

// OriginalFormat returns the original format of the image when loaded.
// In some cases the loaded image is converted on load, for example, a BMP is automatically converted to PNG
// This method returns the format of the original buffer, as opposed to Format() with will return the format of the
// currently held buffer content.
func (r *ImageRef) OriginalFormat() ImageType {
	return r.originalFormat
}

// Width returns the width of this image.
func (r *ImageRef) Width() int {
	return int(r.image.Xsize)
}

// Height returns the height of this image.
func (r *ImageRef) Height() int {
	return int(r.image.Ysize)
}

// Bands returns the number of bands for this image.
func (r *ImageRef) Bands() int {
	return int(r.image.Bands)
}

// HasProfile returns if the image has an ICC profile embedded.
func (r *ImageRef) HasProfile() bool {
	defer runtime.KeepAlive(r)
	return vipsHasICCProfile(r.image)
}

// HasICCProfile checks whether the image has an ICC profile embedded. Alias to HasProfile
func (r *ImageRef) HasICCProfile() bool {
	return r.HasProfile()
}

// HasIPTC returns a boolean whether the image in question has IPTC data associated with it.
func (r *ImageRef) HasIPTC() bool {
	defer runtime.KeepAlive(r)
	return vipsHasIPTC(r.image)
}

// HasAlpha returns if the image has an alpha layer.
func (r *ImageRef) HasAlpha() bool {
	defer runtime.KeepAlive(r)
	return vipsHasAlpha(r.image)
}

// Orientation returns the orientation number as it appears in the EXIF, if present
func (r *ImageRef) Orientation() int {
	defer runtime.KeepAlive(r)
	return vipsGetMetaOrientation(r.image)
}

// Deprecated: use Orientation() instead
func (r *ImageRef) GetOrientation() int {
	return r.Orientation()
}

// SetOrientation sets the orientation in the EXIF header of the associated image.
func (r *ImageRef) SetOrientation(orientation int) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenCopy(r.image, nil)
	if err != nil {
		return err
	}

	vipsSetMetaOrientation(out, orientation)

	r.setImage(out)
	return nil
}

// RemoveOrientation removes the EXIF orientation information of the image.
func (r *ImageRef) RemoveOrientation() error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenCopy(r.image, nil)
	if err != nil {
		return err
	}

	vipsRemoveMetaOrientation(out)

	r.setImage(out)
	return nil
}

// ResX returns the X resolution
func (r *ImageRef) ResX() float64 {
	return float64(r.image.Xres)
}

// ResY returns the Y resolution
func (r *ImageRef) ResY() float64 {
	return float64(r.image.Yres)
}

// OffsetX returns the X offset
func (r *ImageRef) OffsetX() int {
	return int(r.image.Xoffset)
}

// OffsetY returns the Y offset
func (r *ImageRef) OffsetY() int {
	return int(r.image.Yoffset)
}

// BandFormat returns the current band format
func (r *ImageRef) BandFormat() BandFormat {
	return BandFormat(int(r.image.BandFmt))
}

// Coding returns the image coding
func (r *ImageRef) Coding() Coding {
	return Coding(int(r.image.Coding))
}

// Interpretation returns the current interpretation of the color space of the image.
func (r *ImageRef) Interpretation() Interpretation {
	return Interpretation(int(r.image.Type))
}

// ColorSpace returns the interpretation of the current color space. Alias to Interpretation().
func (r *ImageRef) ColorSpace() Interpretation {
	return r.Interpretation()
}

// IsColorSpaceSupported returns a boolean whether the image's color space is supported by libvips.
func (r *ImageRef) IsColorSpaceSupported() bool {
	defer runtime.KeepAlive(r)
	return vipsIsColorSpaceSupported(r.image)
}

// Pages returns the number of pages in the Image
// For animated images this corresponds to the number of frames
func (r *ImageRef) Pages() int {
	// libvips uses the same attribute (n_pages) to represent the number of pyramid layers in JP2K
	// as we interpret the attribute as frames and JP2K does not support animation we override this with 1
	if r.format == ImageTypeJP2K {
		return 1
	}

	defer runtime.KeepAlive(r)
	return vipsGetImageNPages(r.image)
}

// Deprecated: use Pages() instead
func (r *ImageRef) GetPages() int {
	return r.Pages()
}

// SetPages sets the number of pages in the Image
// For animated images this corresponds to the number of frames
func (r *ImageRef) SetPages(pages int) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenCopy(r.image, nil)
	if err != nil {
		return err
	}

	vipsSetImageNPages(out, pages)

	r.setImage(out)
	return nil
}

// PageHeight return the height of a single page
func (r *ImageRef) PageHeight() int {
	defer runtime.KeepAlive(r)
	return vipsGetPageHeight(r.image)
}

// GetPageHeight return the height of a single page
// Deprecated use PageHeight() instead
func (r *ImageRef) GetPageHeight() int {
	defer runtime.KeepAlive(r)
	return vipsGetPageHeight(r.image)
}

// SetPageHeight set the height of a page
// For animated images this is used when "unrolling" back to frames
func (r *ImageRef) SetPageHeight(height int) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenCopy(r.image, nil)
	if err != nil {
		return err
	}

	vipsSetPageHeight(out, height)

	r.setImage(out)
	return nil
}

// PageDelay get the page delay array for animation
func (r *ImageRef) PageDelay() ([]int, error) {
	defer runtime.KeepAlive(r)
	n := vipsGetImageNPages(r.image)
	if n <= 1 {
		// should not call if not multi page
		return nil, nil
	}
	return vipsImageGetDelay(r.image, n)
}

// SetPageDelay set the page delay array for animation
func (r *ImageRef) SetPageDelay(delay []int) error {
	defer runtime.KeepAlive(r)
	var data []C.int
	for _, d := range delay {
		data = append(data, C.int(d))
	}
	return vipsImageSetDelay(r.image, data)
}

// Loop returns the loop count for animated images.
// A value of 0 means infinite looping.
func (r *ImageRef) Loop() int {
	defer runtime.KeepAlive(r)
	return vipsImageGetLoop(r.image)
}

// SetLoop sets the loop count for animated images.
// A value of 0 means infinite looping.
func (r *ImageRef) SetLoop(loop int) error {
	defer runtime.KeepAlive(r)
	vipsImageSetLoop(r.image, loop)
	return nil
}

// Background get the background of image.
func (r *ImageRef) Background() ([]float64, error) {
	defer runtime.KeepAlive(r)
	out, err := vipsImageGetBackground(r.image)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *ImageRef) ImageFields() []string {
	return r.GetFields()
}

func (r *ImageRef) GetFields() []string {
	defer runtime.KeepAlive(r)
	return vipsImageGetFields(r.image)
}

func (r *ImageRef) SetBlob(name string, data []byte) {
	defer runtime.KeepAlive(r)
	vipsImageSetBlob(r.image, name, data)
}

func (r *ImageRef) GetBlob(name string) []byte {
	defer runtime.KeepAlive(r)
	return vipsImageGetBlob(r.image, name)
}

func (r *ImageRef) SetDouble(name string, f float64) {
	defer runtime.KeepAlive(r)
	vipsImageSetDouble(r.image, name, f)
}

func (r *ImageRef) GetDouble(name string) float64 {
	defer runtime.KeepAlive(r)
	return vipsImageGetDouble(r.image, name)
}

func (r *ImageRef) SetInt(name string, i int) {
	defer runtime.KeepAlive(r)
	vipsImageSetInt(r.image, name, i)
}

func (r *ImageRef) GetInt(name string) int {
	defer runtime.KeepAlive(r)
	return vipsImageGetInt(r.image, name)
}

func (r *ImageRef) SetString(name string, str string) {
	defer runtime.KeepAlive(r)
	vipsImageSetString(r.image, name, str)
}

func (r *ImageRef) GetString(name string) string {
	defer runtime.KeepAlive(r)
	return vipsImageGetString(r.image, name)
}

func (r *ImageRef) GetAsString(name string) string {
	defer runtime.KeepAlive(r)
	return vipsImageGetAsString(r.image, name)
}

func (r *ImageRef) HasExif() bool {
	for _, field := range r.ImageFields() {
		if strings.HasPrefix(field, "exif-") {
			return true
		}
	}

	return false
}

func (r *ImageRef) GetExif() map[string]string {
	defer runtime.KeepAlive(r)
	return vipsImageGetExifData(r.image)
}

// RemoveMetadata removes the EXIF metadata from the image.
// N.B. this function won't remove the ICC profile, orientation and pages metadata
// because govips needs it to correctly display the image.
func (r *ImageRef) RemoveMetadata(keep ...string) error {
	defer runtime.KeepAlive(r)
	out, err := vipsGenCopy(r.image, nil)
	if err != nil {
		return err
	}

	vipsRemoveMetadata(out, keep...)

	r.setImage(out)

	return nil
}
