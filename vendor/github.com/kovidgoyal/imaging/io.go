package imaging

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kovidgoyal/imaging/prism/meta/autometa"
	"github.com/rwcarlsen/goexif/exif"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

type fileSystem interface {
	Create(string) (io.WriteCloser, error)
	Open(string) (io.ReadCloser, error)
}

type localFS struct{}

func (localFS) Create(name string) (io.WriteCloser, error) { return os.Create(name) }
func (localFS) Open(name string) (io.ReadCloser, error)    { return os.Open(name) }

var fs fileSystem = localFS{}

type decodeConfig struct {
	autoOrientation bool
}

var defaultDecodeConfig = decodeConfig{
	autoOrientation: true,
}

// DecodeOption sets an optional parameter for the Decode and Open functions.
type DecodeOption func(*decodeConfig)

// AutoOrientation returns a DecodeOption that sets the auto-orientation mode.
// If auto-orientation is enabled, the image will be transformed after decoding
// according to the EXIF orientation tag (if present). By default it's enabled.
func AutoOrientation(enabled bool) DecodeOption {
	return func(c *decodeConfig) {
		c.autoOrientation = enabled
	}
}

// Decode reads an image from r.
func Decode(r io.Reader, opts ...DecodeOption) (image.Image, error) {
	cfg := defaultDecodeConfig

	for _, option := range opts {
		option(&cfg)
	}

	if !cfg.autoOrientation {
		img, _, err := image.Decode(r)
		return img, err
	}
	md, r, err := autometa.Load(r)
	var oval orientation = orientationUnspecified
	if err == nil && md != nil && len(md.ExifData) > 6 {
		exif_data, err := exif.Decode(bytes.NewReader(md.ExifData))
		if err == nil {
			orient, err := exif_data.Get(exif.Orientation)
			if err == nil && orient != nil {
				x, err := strconv.ParseUint(orient.String(), 10, 0)
				if err == nil && x > 0 && x < 9 {
					oval = orientation(int(x))
				}
			}
		}
	}

	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}

	return fixOrientation(img, oval), nil
}

// Open loads an image from file.
//
// Examples:
//
//	// Load an image from file.
//	img, err := imaging.Open("test.jpg")
//
//	// Load an image and transform it depending on the EXIF orientation tag (if present).
//	img, err := imaging.Open("test.jpg", imaging.AutoOrientation(true))
func Open(filename string, opts ...DecodeOption) (image.Image, error) {
	file, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return Decode(file, opts...)
}

func OpenConfig(filename string) (ans image.Config, format_name string, err error) {
	file, err := fs.Open(filename)
	if err != nil {
		return ans, "", err
	}
	defer file.Close()
	return image.DecodeConfig(file)
}

// Format is an image file format.
type Format int

// Image file formats.
const (
	JPEG Format = iota
	PNG
	GIF
	TIFF
	BMP
	PBM
	PGM
	PPM
	PAM
)

var formatExts = map[string]Format{
	"jpg":  JPEG,
	"jpeg": JPEG,
	"png":  PNG,
	"gif":  GIF,
	"tif":  TIFF,
	"tiff": TIFF,
	"bmp":  BMP,
	"pbm":  PBM,
	"pgm":  PGM,
	"ppm":  PPM,
	"pam":  PAM,
}

var formatNames = map[Format]string{
	JPEG: "JPEG",
	PNG:  "PNG",
	GIF:  "GIF",
	TIFF: "TIFF",
	BMP:  "BMP",
	PBM:  "PBM",
	PGM:  "PGM",
	PAM:  "PAM",
}

func (f Format) String() string {
	return formatNames[f]
}

// ErrUnsupportedFormat means the given image format is not supported.
var ErrUnsupportedFormat = errors.New("imaging: unsupported image format")

// FormatFromExtension parses image format from filename extension:
// "jpg" (or "jpeg"), "png", "gif", "tif" (or "tiff") and "bmp" are supported.
func FormatFromExtension(ext string) (Format, error) {
	if f, ok := formatExts[strings.ToLower(strings.TrimPrefix(ext, "."))]; ok {
		return f, nil
	}
	return -1, ErrUnsupportedFormat
}

// FormatFromFilename parses image format from filename:
// "jpg" (or "jpeg"), "png", "gif", "tif" (or "tiff") and "bmp" are supported.
func FormatFromFilename(filename string) (Format, error) {
	ext := filepath.Ext(filename)
	return FormatFromExtension(ext)
}

type encodeConfig struct {
	jpegQuality         int
	gifNumColors        int
	gifQuantizer        draw.Quantizer
	gifDrawer           draw.Drawer
	pngCompressionLevel png.CompressionLevel
}

var defaultEncodeConfig = encodeConfig{
	jpegQuality:         95,
	gifNumColors:        256,
	gifQuantizer:        nil,
	gifDrawer:           nil,
	pngCompressionLevel: png.DefaultCompression,
}

// EncodeOption sets an optional parameter for the Encode and Save functions.
type EncodeOption func(*encodeConfig)

// JPEGQuality returns an EncodeOption that sets the output JPEG quality.
// Quality ranges from 1 to 100 inclusive, higher is better. Default is 95.
func JPEGQuality(quality int) EncodeOption {
	return func(c *encodeConfig) {
		c.jpegQuality = quality
	}
}

// GIFNumColors returns an EncodeOption that sets the maximum number of colors
// used in the GIF-encoded image. It ranges from 1 to 256.  Default is 256.
func GIFNumColors(numColors int) EncodeOption {
	return func(c *encodeConfig) {
		c.gifNumColors = numColors
	}
}

// GIFQuantizer returns an EncodeOption that sets the quantizer that is used to produce
// a palette of the GIF-encoded image.
func GIFQuantizer(quantizer draw.Quantizer) EncodeOption {
	return func(c *encodeConfig) {
		c.gifQuantizer = quantizer
	}
}

// GIFDrawer returns an EncodeOption that sets the drawer that is used to convert
// the source image to the desired palette of the GIF-encoded image.
func GIFDrawer(drawer draw.Drawer) EncodeOption {
	return func(c *encodeConfig) {
		c.gifDrawer = drawer
	}
}

// PNGCompressionLevel returns an EncodeOption that sets the compression level
// of the PNG-encoded image. Default is png.DefaultCompression.
func PNGCompressionLevel(level png.CompressionLevel) EncodeOption {
	return func(c *encodeConfig) {
		c.pngCompressionLevel = level
	}
}

// Encode writes the image img to w in the specified format (JPEG, PNG, GIF, TIFF or BMP).
func Encode(w io.Writer, img image.Image, format Format, opts ...EncodeOption) error {
	cfg := defaultEncodeConfig
	for _, option := range opts {
		option(&cfg)
	}

	switch format {
	case JPEG:
		if nrgba, ok := img.(*image.NRGBA); ok && nrgba.Opaque() {
			rgba := &image.RGBA{
				Pix:    nrgba.Pix,
				Stride: nrgba.Stride,
				Rect:   nrgba.Rect,
			}
			return jpeg.Encode(w, rgba, &jpeg.Options{Quality: cfg.jpegQuality})
		}
		return jpeg.Encode(w, img, &jpeg.Options{Quality: cfg.jpegQuality})

	case PNG:
		encoder := png.Encoder{CompressionLevel: cfg.pngCompressionLevel}
		return encoder.Encode(w, img)

	case GIF:
		return gif.Encode(w, img, &gif.Options{
			NumColors: cfg.gifNumColors,
			Quantizer: cfg.gifQuantizer,
			Drawer:    cfg.gifDrawer,
		})

	case TIFF:
		return tiff.Encode(w, img, &tiff.Options{Compression: tiff.Deflate, Predictor: true})

	case BMP:
		return bmp.Encode(w, img)
	}

	return ErrUnsupportedFormat
}

// Save saves the image to file with the specified filename.
// The format is determined from the filename extension:
// "jpg" (or "jpeg"), "png", "gif", "tif" (or "tiff") and "bmp" are supported.
//
// Examples:
//
//	// Save the image as PNG.
//	err := imaging.Save(img, "out.png")
//
//	// Save the image as JPEG with optional quality parameter set to 80.
//	err := imaging.Save(img, "out.jpg", imaging.JPEGQuality(80))
func Save(img image.Image, filename string, opts ...EncodeOption) (err error) {
	f, err := FormatFromFilename(filename)
	if err != nil {
		return err
	}
	file, err := fs.Create(filename)
	if err != nil {
		return err
	}
	err = Encode(file, img, f, opts...)
	errc := file.Close()
	if err == nil {
		err = errc
	}
	return err
}

// orientation is an EXIF flag that specifies the transformation
// that should be applied to image to display it correctly.
type orientation int

const (
	orientationUnspecified = 0
	orientationNormal      = 1
	orientationFlipH       = 2
	orientationRotate180   = 3
	orientationFlipV       = 4
	orientationTranspose   = 5
	orientationRotate270   = 6
	orientationTransverse  = 7
	orientationRotate90    = 8
)

// fixOrientation applies a transform to img corresponding to the given orientation flag.
func fixOrientation(img image.Image, o orientation) image.Image {
	switch o {
	case orientationNormal:
	case orientationFlipH:
		img = FlipH(img)
	case orientationFlipV:
		img = FlipV(img)
	case orientationRotate90:
		img = Rotate90(img)
	case orientationRotate180:
		img = Rotate180(img)
	case orientationRotate270:
		img = Rotate270(img)
	case orientationTranspose:
		img = Transpose(img)
	case orientationTransverse:
		img = Transverse(img)
	}
	return img
}
