package vips

// #include "image.h"
import "C"

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	_ "golang.org/x/image/webp"
)

const GaussBlurDefaultMinAMpl = 0.2

// PreMultiplicationState stores the pre-multiplication band format of the image
type PreMultiplicationState struct {
	bandFormat BandFormat
}

// ImageRef contains a libvips image and manages its lifecycle.
type ImageRef struct {
	// NOTE: We keep a reference to this so that the input buffer is
	// never garbage collected during processing. Some image loaders use random
	// access transcoding and therefore need the original buffer to be in memory.
	buf                 []byte
	image               *C.VipsImage
	format              ImageType
	originalFormat      ImageType
	lock                sync.Mutex
	preMultiplication   *PreMultiplicationState
	optimizedIccProfile string
}

// ImageMetadata is a data structure holding the width, height, orientation and other metadata of the picture.
type ImageMetadata struct {
	Format      ImageType
	Width       int
	Height      int
	Colorspace  Interpretation
	Orientation int
	Pages       int
}

type Parameter struct {
	value interface{}
	isSet bool
}

func (p *Parameter) IsSet() bool {
	return p.isSet
}

func (p *Parameter) set(v interface{}) {
	p.value = v
	p.isSet = true
}

type BoolParameter struct {
	Parameter
}

func (p *BoolParameter) Set(v bool) {
	p.set(v)
}

func (p *BoolParameter) Get() bool {
	return p.value.(bool)
}

type IntParameter struct {
	Parameter
}

func (p *IntParameter) Set(v int) {
	p.set(v)
}

func (p *IntParameter) Get() int {
	return p.value.(int)
}

type Float64Parameter struct {
	Parameter
}

func (p *Float64Parameter) Set(v float64) {
	p.set(v)
}

func (p *Float64Parameter) Get() float64 {
	return p.value.(float64)
}

// ImportParams are options for loading an image. Some are type-specific.
// For default loading, use NewImportParams() or specify nil
type ImportParams struct {
	AutoRotate  BoolParameter
	FailOnError BoolParameter
	Page        IntParameter
	NumPages    IntParameter
	Density     IntParameter

	JpegShrinkFactor IntParameter
	WebpScaleFactor  Float64Parameter
	HeifThumbnail    BoolParameter
	SvgUnlimited     BoolParameter
	Access           IntParameter
}

// NewImportParams creates default ImportParams
func NewImportParams() *ImportParams {
	p := &ImportParams{}
	p.FailOnError.Set(true)
	return p
}

// OptionString convert import params to option_string
func (i *ImportParams) OptionString() string {
	var values []string
	if v := i.NumPages; v.IsSet() {
		values = append(values, "n="+strconv.Itoa(v.Get()))
	}
	if v := i.Page; v.IsSet() {
		values = append(values, "page="+strconv.Itoa(v.Get()))
	}
	if v := i.Density; v.IsSet() {
		values = append(values, "dpi="+strconv.Itoa(v.Get()))
	}
	if v := i.FailOnError; v.IsSet() {
		values = append(values, "fail="+boolToStr(v.Get()))
	}
	if v := i.JpegShrinkFactor; v.IsSet() {
		values = append(values, "shrink="+strconv.Itoa(v.Get()))
	}
	if v := i.WebpScaleFactor; v.IsSet() {
		values = append(values, "scale="+strconv.FormatFloat(v.Get(), 'f', -1, 64))
	}
	if v := i.AutoRotate; v.IsSet() {
		values = append(values, "autorotate="+boolToStr(v.Get()))
	}
	if v := i.SvgUnlimited; v.IsSet() {
		values = append(values, "unlimited="+boolToStr(v.Get()))
	}
	if v := i.HeifThumbnail; v.IsSet() {
		values = append(values, "thumbnail="+boolToStr(v.Get()))
	}
	if v := i.Access; v.IsSet() {
		values = append(values, "access="+strconv.Itoa(v.Get()))
	}
	return strings.Join(values, ",")
}

func boolToStr(v bool) string {
	if v {
		return "TRUE"
	}
	return "FALSE"
}

// ExportParams are options when exporting an image to file or buffer.
// Deprecated: Use format-specific params
type ExportParams struct {
	Format             ImageType
	Quality            int
	Compression        int
	Interlaced         bool
	Lossless           bool
	Effort             int
	StripMetadata      bool
	OptimizeCoding     bool          // jpeg param
	SubsampleMode      SubsampleMode // jpeg param
	TrellisQuant       bool          // jpeg param
	OvershootDeringing bool          // jpeg param
	OptimizeScans      bool          // jpeg param
	QuantTable         int           // jpeg param
	Speed              int           // avif param
}

// NewDefaultExportParams creates default values for an export when image type is not JPEG, PNG or WEBP.
// By default, govips creates interlaced, lossy images with a quality of 80/100 and compression of 6/10.
// As these are default values for a wide variety of image formats, their application varies.
// Some formats use the quality parameters, some compression, etc.
// Deprecated: Use format-specific params
func NewDefaultExportParams() *ExportParams {
	return &ExportParams{
		Format:      ImageTypeUnknown, // defaults to the starting encoder
		Quality:     80,
		Compression: 6,
		Interlaced:  true,
		Lossless:    false,
		Effort:      4,
	}
}

// NewDefaultJPEGExportParams creates default values for an export of a JPEG image.
// By default, govips creates interlaced JPEGs with a quality of 80/100.
// Deprecated: Use NewJpegExportParams
func NewDefaultJPEGExportParams() *ExportParams {
	return &ExportParams{
		Format:     ImageTypeJPEG,
		Quality:    80,
		Interlaced: true,
	}
}

// NewDefaultPNGExportParams creates default values for an export of a PNG image.
// By default, govips creates non-interlaced PNGs with a compression of 6/10.
// Deprecated: Use NewPngExportParams
func NewDefaultPNGExportParams() *ExportParams {
	return &ExportParams{
		Format:      ImageTypePNG,
		Compression: 6,
		Interlaced:  false,
	}
}

// NewDefaultWEBPExportParams creates default values for an export of a WEBP image.
// By default, govips creates lossy images with a quality of 75/100.
// Deprecated: Use NewWebpExportParams
func NewDefaultWEBPExportParams() *ExportParams {
	return &ExportParams{
		Format:   ImageTypeWEBP,
		Quality:  75,
		Lossless: false,
		Effort:   4,
	}
}

// JpegExportParams are options when exporting a JPEG to file or buffer
type JpegExportParams struct {
	StripMetadata      bool
	Quality            int
	Interlace          bool
	OptimizeCoding     bool
	SubsampleMode      SubsampleMode
	TrellisQuant       bool
	OvershootDeringing bool
	OptimizeScans      bool
	QuantTable         int
}

// NewJpegExportParams creates default values for an export of a JPEG image.
// By default, govips creates interlaced JPEGs with a quality of 80/100.
func NewJpegExportParams() *JpegExportParams {
	return &JpegExportParams{
		Quality:   80,
		Interlace: true,
	}
}

// PngExportParams are options when exporting a PNG to file or buffer
type PngExportParams struct {
	StripMetadata bool
	Compression   int
	Filter        PngFilter
	Interlace     bool
	Quality       int
	Palette       bool
	Dither        float64
	Bitdepth      int
	Profile       string
}

// NewPngExportParams creates default values for an export of a PNG image.
// By default, govips creates non-interlaced PNGs with a compression of 6/10.
func NewPngExportParams() *PngExportParams {
	return &PngExportParams{
		Compression: 6,
		Filter:      PngFilterNone,
		Interlace:   false,
		Palette:     false,
	}
}

// WebpExportParams are options when exporting a WEBP to file or buffer
// see https://www.libvips.org/API/current/VipsForeignSave.html#vips-webpsave
// for details on each parameter
type WebpExportParams struct {
	StripMetadata   bool
	Quality         int
	Lossless        bool
	NearLossless    bool
	ReductionEffort int
	IccProfile      string
	MinSize         bool
	MinKeyFrames    int
	MaxKeyFrames    int
}

// NewWebpExportParams creates default values for an export of a WEBP image.
// By default, govips creates lossy images with a quality of 75/100.
func NewWebpExportParams() *WebpExportParams {
	return &WebpExportParams{
		Quality:         75,
		Lossless:        false,
		NearLossless:    false,
		ReductionEffort: 4,
	}
}

// TiffExportParams are options when exporting a TIFF to file or buffer
type TiffExportParams struct {
	StripMetadata bool
	Quality       int
	Compression   TiffCompression
	Predictor     TiffPredictor
	Pyramid       bool
	Tile          bool
	TileHeight    int
	TileWidth     int
}

// NewTiffExportParams creates default values for an export of a TIFF image.
func NewTiffExportParams() *TiffExportParams {
	return &TiffExportParams{
		Quality:     80,
		Compression: TiffCompressionLzw,
		Predictor:   TiffPredictorHorizontal,
		Pyramid:     false,
		Tile:        false,
		TileHeight:  256,
		TileWidth:   256,
	}
}

// GifExportParams are options when exporting a GIF to file or buffer.
//
// For vips 8.12+, native gifsave is used. The relevant parameters are Dither,
// Effort, and Bitdepth. Quality is ignored because native gifsave does not
// support a quality parameter.
//
// For vips below 8.12, magicksave is used as a fallback. The relevant
// parameters are Quality and Bitdepth.
//
// StripMetadata has no effect on GIF images.
type GifExportParams struct {
	StripMetadata bool
	// Quality is only used with vips < 8.12 (magicksave fallback).
	// Ignored by native gifsave in vips 8.12+.
	Quality  int
	Dither   float64
	Effort   int
	Bitdepth int
}

// NewGifExportParams creates default values for an export of a GIF image.
func NewGifExportParams() *GifExportParams {
	return &GifExportParams{
		Quality:  75,
		Effort:   7,
		Bitdepth: 8,
	}
}

// HeifExportParams are options when exporting a HEIF to file or buffer
type HeifExportParams struct {
	Quality  int
	Bitdepth int
	Effort   int
	Lossless bool
}

// NewHeifExportParams creates default values for an export of a HEIF image.
func NewHeifExportParams() *HeifExportParams {
	return &HeifExportParams{
		Quality:  80,
		Bitdepth: 8,
		Effort:   5,
		Lossless: false,
	}
}

// AvifExportParams are options when exporting an AVIF to file or buffer.
type AvifExportParams struct {
	StripMetadata bool
	Quality       int
	Bitdepth      int
	Effort        int
	Lossless      bool

	// DEPRECATED - Use Effort instead.
	Speed int
}

// NewAvifExportParams creates default values for an export of an AVIF image.
func NewAvifExportParams() *AvifExportParams {
	return &AvifExportParams{
		Quality:  80,
		Bitdepth: 8,
		Effort:   5,
		Lossless: false,
	}
}

// Jp2kExportParams are options when exporting an JPEG2000 to file or buffer.
type Jp2kExportParams struct {
	Quality       int
	Lossless      bool
	TileWidth     int
	TileHeight    int
	SubsampleMode SubsampleMode
}

// NewJp2kExportParams creates default values for an export of an JPEG2000 image.
func NewJp2kExportParams() *Jp2kExportParams {
	return &Jp2kExportParams{
		Quality:    80,
		Lossless:   false,
		TileWidth:  512,
		TileHeight: 512,
	}
}

// JxlExportParams are options when exporting an JXL to file or buffer.
type JxlExportParams struct {
	Quality  int
	Lossless bool
	Tier     int
	Distance float64
	Effort   int
}

// NewJxlExportParams creates default values for an export of an JXL image.
func NewJxlExportParams() *JxlExportParams {
	return &JxlExportParams{
		Quality:  75,
		Lossless: false,
		Effort:   7,
		Distance: 1.0,
	}
}

// MagickExportParams are options when exporting an image to file or buffer by ImageMagick.
type MagickExportParams struct {
	Quality                 int
	Format                  string
	OptimizeGifFrames       bool
	OptimizeGifTransparency bool
	BitDepth                int
}

// NewMagickExportParams creates default values for an export of an image by ImageMagick.
func NewMagickExportParams() *MagickExportParams {
	return &MagickExportParams{
		Quality: 75,
	}
}

// NewImageFromReader loads an ImageRef from the given reader
func NewImageFromReader(r io.Reader) (*ImageRef, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return NewImageFromBuffer(buf)
}

// NewImageFromFile loads an image from file and creates a new ImageRef
func NewImageFromFile(file string) (*ImageRef, error) {
	return LoadImageFromFile(file, nil)
}

// LoadImageFromFile loads an image from file and creates a new ImageRef
func LoadImageFromFile(file string, params *ImportParams) (*ImageRef, error) {
	buf, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	govipsLog("govips", LogLevelDebug, fmt.Sprintf("creating imageRef from file %s", file))
	return LoadImageFromBuffer(buf, params)
}

// NewImageFromBuffer loads an image buffer and creates a new Image
func NewImageFromBuffer(buf []byte) (*ImageRef, error) {
	return LoadImageFromBuffer(buf, nil)
}

// LoadImageFromBuffer loads an image buffer and creates a new Image
func LoadImageFromBuffer(buf []byte, params *ImportParams) (*ImageRef, error) {
	if err := startupIfNeeded(); err != nil {
		return nil, err
	}

	if params == nil {
		params = NewImportParams()
	}

	vipsImage, currentFormat, originalFormat, err := vipsLoadFromBuffer(buf, params)
	if err != nil {
		return nil, err
	}

	ref := newImageRef(vipsImage, currentFormat, originalFormat, buf)

	govipsLog("govips", LogLevelDebug, fmt.Sprintf("created imageRef %p", ref))
	return ref, nil
}

// NewThumbnailFromFile loads an image from file and creates a new ImageRef with thumbnail crop
func NewThumbnailFromFile(file string, width, height int, crop Interesting) (*ImageRef, error) {
	return LoadThumbnailFromFile(file, width, height, crop, SizeBoth, nil)
}

// NewThumbnailFromBuffer loads an image buffer and creates a new Image with thumbnail crop
func NewThumbnailFromBuffer(buf []byte, width, height int, crop Interesting) (*ImageRef, error) {
	return LoadThumbnailFromBuffer(buf, width, height, crop, SizeBoth, nil)
}

// NewThumbnailWithSizeFromFile loads an image from file and creates a new ImageRef with thumbnail crop and size
func NewThumbnailWithSizeFromFile(file string, width, height int, crop Interesting, size Size) (*ImageRef, error) {
	return LoadThumbnailFromFile(file, width, height, crop, size, nil)
}

// LoadThumbnailFromFile loads an image from file and creates a new ImageRef with thumbnail crop and size
func LoadThumbnailFromFile(file string, width, height int, crop Interesting, size Size, params *ImportParams) (*ImageRef, error) {
	if err := startupIfNeeded(); err != nil {
		return nil, err
	}

	vipsImage, format, err := vipsThumbnailFromFile(file, width, height, crop, size, params)
	if err != nil {
		return nil, err
	}

	ref := newImageRef(vipsImage, format, format, nil)

	govipsLog("govips", LogLevelDebug, fmt.Sprintf("created imageref %p", ref))
	return ref, nil
}

// NewThumbnailWithSizeFromBuffer loads an image buffer and creates a new Image with thumbnail crop and size
func NewThumbnailWithSizeFromBuffer(buf []byte, width, height int, crop Interesting, size Size) (*ImageRef, error) {
	return LoadThumbnailFromBuffer(buf, width, height, crop, size, nil)
}

// LoadThumbnailFromBuffer loads an image buffer and creates a new Image with thumbnail crop and size
func LoadThumbnailFromBuffer(buf []byte, width, height int, crop Interesting, size Size, params *ImportParams) (*ImageRef, error) {
	if err := startupIfNeeded(); err != nil {
		return nil, err
	}

	vipsImage, format, err := vipsThumbnailFromBuffer(buf, width, height, crop, size, params)
	if err != nil {
		return nil, err
	}

	ref := newImageRef(vipsImage, format, format, buf)

	govipsLog("govips", LogLevelDebug, fmt.Sprintf("created imageref %p", ref))
	return ref, nil
}

// Metadata returns the metadata (ImageMetadata struct) of the associated ImageRef
func (r *ImageRef) Metadata() *ImageMetadata {
	return &ImageMetadata{
		Format:      r.Format(),
		Width:       r.Width(),
		Height:      r.Height(),
		Orientation: r.Orientation(),
		Colorspace:  r.ColorSpace(),
		Pages:       r.Pages(),
	}
}

// Copy creates a new copy of the given image.
func (r *ImageRef) Copy() (*ImageRef, error) {
	defer runtime.KeepAlive(r)
	out, err := vipsGenCopy(r.image, nil)
	if err != nil {
		return nil, err
	}

	return newImageRef(out, r.format, r.originalFormat, r.buf), nil
}

// Copy creates a new copy of the given image with the new X and Y resolution (PPI).
func (r *ImageRef) CopyChangingResolution(xres, yres float64) (*ImageRef, error) {
	defer runtime.KeepAlive(r)
	out, err := vipsGenCopy(r.image, &CopyOptions{Xres: &xres, Yres: &yres})
	if err != nil {
		return nil, err
	}

	return newImageRef(out, r.format, r.originalFormat, r.buf), nil
}

// Copy creates a new copy of the given image with the interpretation.
func (r *ImageRef) CopyChangingInterpretation(interpretation Interpretation) (*ImageRef, error) {
	defer runtime.KeepAlive(r)
	out, err := vipsGenCopy(r.image, &CopyOptions{Interpretation: &interpretation})
	if err != nil {
		return nil, err
	}

	return newImageRef(out, r.format, r.originalFormat, r.buf), nil
}

// XYZ creates a two-band uint32 image where the elements in the first band have the value of their x coordinate
// and elements in the second band have their y coordinate.
func XYZ(width, height int) (*ImageRef, error) {
	vipsImage, err := vipsGenXyz(width, height, nil)
	return newImageRef(vipsImage, ImageTypeUnknown, ImageTypeUnknown, nil), err
}

// Identity creates an identity lookup table, which will leave an image unchanged when applied with Maplut.
// Each entry in the table has a value equal to its position.
func Identity(ushort bool) (*ImageRef, error) {
	img, err := vipsGenIdentity(&IdentityOptions{Ushort: &ushort})
	return newImageRef(img, ImageTypeUnknown, ImageTypeUnknown, nil), err
}

// Black creates a new black image of the specified size
func Black(width, height int) (*ImageRef, error) {
	vipsImage, err := vipsGenBlack(width, height, nil)
	if err != nil {
		return nil, err
	}
	return newImageRef(vipsImage, ImageTypeUnknown, ImageTypeUnknown, nil), nil
}

// Grey creates a horizontal gradient image (ramp from black to white).
// When uchar is true, pixel values are 0-255 uint8; when false, 0.0-1.0 float.
// Useful for creating gradient overlays when combined with rotation, BandJoin, and Composite.
func Grey(width, height int, uchar bool) (*ImageRef, error) {
	img, err := vipsGenGrey(width, height, &GreyOptions{Uchar: &uchar})
	if err != nil {
		return nil, err
	}
	return newImageRef(img, ImageTypeUnknown, ImageTypeUnknown, nil), nil
}

// NewTransparentCanvas creates a fully transparent RGBA image of the given dimensions.
// The image is in sRGB color space with 4 bands (RGBA), all channels set to 0.
func NewTransparentCanvas(width, height int) (*ImageRef, error) {
	ref, err := Black(width, height)
	if err != nil {
		return nil, err
	}

	if err := ref.ToColorSpace(InterpretationSRGB); err != nil {
		ref.Close()
		return nil, err
	}

	if err := ref.BandJoinConst([]float64{0}); err != nil {
		ref.Close()
		return nil, err
	}

	return ref, nil
}

// Text draws the string text to an image.
func Text(params *TextParams) (*ImageRef, error) {
	img, err := vipsText(params)
	return newImageRef(img, ImageTypeUnknown, ImageTypeUnknown, nil), err
}

// NewImageFromGoImage creates a new ImageRef from a Go image.Image.
// The image is normalized to NRGBA (non-premultiplied RGBA, 8-bit) and
// imported into libvips in sRGB color space.
func NewImageFromGoImage(img image.Image) (*ImageRef, error) {
	if err := startupIfNeeded(); err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width == 0 || height == 0 {
		return nil, errors.New("image has zero dimensions")
	}

	// Normalize to NRGBA
	var nrgba *image.NRGBA
	if n, ok := img.(*image.NRGBA); ok && n.Rect.Min.X == 0 && n.Rect.Min.Y == 0 && n.Stride == width*4 {
		nrgba = n
	} else {
		nrgba = image.NewNRGBA(image.Rect(0, 0, width, height))
		draw.Draw(nrgba, nrgba.Bounds(), img, bounds.Min, draw.Src)
	}

	// Create vips image from pixel data (copies data, safe for Go GC)
	vipsImage := C.create_image_from_memory_copy(
		unsafe.Pointer(&nrgba.Pix[0]),
		C.size_t(len(nrgba.Pix)),
		C.int(width),
		C.int(height),
		4,
		C.VIPS_FORMAT_UCHAR,
	)
	runtime.KeepAlive(nrgba)
	if vipsImage == nil {
		return nil, errors.New("failed to create vips image from memory")
	}

	// Set interpretation to sRGB
	vipsImage.Type = C.VIPS_INTERPRETATION_sRGB

	return newImageRef(vipsImage, ImageTypeUnknown, ImageTypeUnknown, nil), nil
}

func newImageRef(vipsImage *C.VipsImage, currentFormat ImageType, originalFormat ImageType, buf []byte) *ImageRef {
	imageRef := &ImageRef{
		image:          vipsImage,
		format:         currentFormat,
		originalFormat: originalFormat,
		buf:            buf,
	}
	openImageRefs.Add(1)
	runtime.SetFinalizer(imageRef, finalizeImage)

	return imageRef
}

func finalizeImage(ref *ImageRef) {
	govipsLog("govips", LogLevelDebug, fmt.Sprintf("closing image %p", ref))
	ref.Close()
}

// Close manually closes the image and frees the memory. Calling Close() is optional.
// Images are automatically closed by GC. However, in high volume applications the GC
// can't keep up with the amount of memory, so you might want to manually close the images.
func (r *ImageRef) Close() {
	r.lock.Lock()

	if r.image != nil {
		clearImage(r.image)
		r.image = nil
		openImageRefs.Add(-1)
	}

	r.buf = nil

	r.lock.Unlock()
}

// setImage resets the image for this image and frees the previous one
func (r *ImageRef) setImage(image *C.VipsImage) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.image == image {
		return
	}

	if r.image != nil {
		clearImage(r.image)
	}

	r.image = image
}

func vipsHasAlpha(in *C.VipsImage) bool {
	return int(C.has_alpha_channel(in)) > 0
}

func clearImage(ref *C.VipsImage) {
	C.clear_image(&ref)
}

// Coding represents VIPS_CODING type
type Coding int

// Coding enum
//
//goland:noinspection GoUnusedConst
const (
	CodingError Coding = C.VIPS_CODING_ERROR
	CodingNone  Coding = C.VIPS_CODING_NONE
	CodingLABQ  Coding = C.VIPS_CODING_LABQ
	CodingRAD   Coding = C.VIPS_CODING_RAD
)

func (r *ImageRef) newMetadata(format ImageType) *ImageMetadata {
	return &ImageMetadata{
		Format:      format,
		Width:       r.Width(),
		Height:      r.Height(),
		Colorspace:  r.ColorSpace(),
		Orientation: r.Orientation(),
		Pages:       r.Pages(),
	}
}
