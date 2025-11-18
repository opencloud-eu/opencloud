package imaging

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/kovidgoyal/imaging/apng"
	myjpeg "github.com/kovidgoyal/imaging/jpeg"
	"github.com/kovidgoyal/imaging/magick"
	_ "github.com/kovidgoyal/imaging/netpbm"
	"github.com/kovidgoyal/imaging/prism/meta"
	"github.com/kovidgoyal/imaging/prism/meta/autometa"
	"github.com/kovidgoyal/imaging/prism/meta/icc"
	"github.com/kovidgoyal/imaging/prism/meta/tiffmeta"
	"github.com/kovidgoyal/imaging/streams"
	"github.com/kovidgoyal/imaging/types"
	"github.com/kovidgoyal/imaging/webp"

	"github.com/rwcarlsen/goexif/exif"
	exif_tiff "github.com/rwcarlsen/goexif/tiff"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

type fileSystem interface {
	Create(string) (io.WriteCloser, error)
	Open(string) (*os.File, error)
}

type localFS struct{}

func (localFS) Create(name string) (io.WriteCloser, error) { return os.Create(name) }
func (localFS) Open(name string) (*os.File, error)         { return os.Open(name) }

var mockable_fs fileSystem = localFS{}

type Backend int

const (
	GO_IMAGE Backend = iota
	MAGICK_IMAGE
)

func (b Backend) String() string {
	switch b {
	case GO_IMAGE:
		return "GO_IMAGE"
	case MAGICK_IMAGE:
		return "MAGICK_IMAGE"
	}
	return fmt.Sprintf("UNKNOWN_IMAGE_TYPE_%d", b)
}

type ColorSpaceType int

const (
	NO_CHANGE_OF_COLORSPACE ColorSpaceType = iota
	SRGB_COLORSPACE
)
const (
	Relative   icc.RenderingIntent = icc.RelativeColorimetricRenderingIntent
	Perceptual                     = icc.PerceptualRenderingIntent
	Saturation                     = icc.SaturationRenderingIntent
	Absolute                       = icc.AbsoluteColorimetricRenderingIntent
)

type ResizeCallbackFunction func(w, h int) (nw, nh int)

type decodeConfig struct {
	autoOrientation             bool
	outputColorspace            ColorSpaceType
	transform                   types.TransformType
	resize                      ResizeCallbackFunction
	background                  *color.RGBA64
	backends                    []Backend
	rendering_intent            icc.RenderingIntent
	use_blackpoint_compensation bool
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

// ColorSpace returns a DecodeOption that sets the colorspace that the
// opened image will be in. Defaults to sRGB. If the image has an embedded ICC
// color profile it is automatically used to convert colors to sRGB if needed.
func ColorSpace(cs ColorSpaceType) DecodeOption {
	return func(c *decodeConfig) {
		c.outputColorspace = cs
	}
}

// Specify a transform to perform on the image when loading it
func Transform(t types.TransformType) DecodeOption {
	return func(c *decodeConfig) {
		c.transform = t
	}
}

// Specify a background color onto which the image should be composed when loading it
func Background(bg color.Color) DecodeOption {
	return func(c *decodeConfig) {
		nbg := color.RGBA64Model.Convert(bg).(color.RGBA64)
		c.background = &nbg
	}
}

// Specify a callback function to decide a new size for the image when loading it
func ResizeCallback(f ResizeCallbackFunction) DecodeOption {
	return func(c *decodeConfig) {
		c.resize = f
	}
}

// Specify which backends to use to try to load the image, successively. If no backends are specified, the default
// set are used.
func Backends(backends ...Backend) DecodeOption {
	return func(c *decodeConfig) {
		c.backends = backends
	}
}

// Set the rendering intent to use for ICC profile based color conversions
func RenderingIntent(intent icc.RenderingIntent) DecodeOption {
	return func(c *decodeConfig) {
		c.rendering_intent = intent
	}
}

// Set whether to use blackpoint compensation during ICC profile color conversions
func BlackpointCompensation(enable bool) DecodeOption {
	return func(c *decodeConfig) {
		c.use_blackpoint_compensation = enable
	}
}

func NewDecodeConfig(opts ...DecodeOption) (cfg *decodeConfig) {
	cfg = &decodeConfig{
		autoOrientation:  true,
		outputColorspace: SRGB_COLORSPACE,
		transform:        types.NoTransform,
		backends:         []Backend{GO_IMAGE, MAGICK_IMAGE},
		// These settings match ImageMagick defaults as of v6
		rendering_intent:            Relative,
		use_blackpoint_compensation: true,
	}
	default_backends := cfg.backends
	for _, option := range opts {
		option(cfg)
	}
	if len(cfg.backends) == 0 {
		cfg.backends = default_backends
	}
	return
}

func (cfg *decodeConfig) magick_callback(w, h int) (ro magick.RenderOptions) {
	ro.AutoOrient = cfg.autoOrientation
	if cfg.resize != nil {
		nw, nh := cfg.resize(w, h)
		if nw != w || nh != h {
			ro.ResizeTo.X, ro.ResizeTo.Y = nw, nh
		}
	}
	ro.Background = cfg.background
	ro.ToSRGB = cfg.outputColorspace == SRGB_COLORSPACE
	ro.Transform = cfg.transform
	ro.RenderingIntent = cfg.rendering_intent
	ro.BlackpointCompensation = cfg.use_blackpoint_compensation
	return
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

func fix_colors(images []*Frame, md *meta.Data, cfg *decodeConfig) error {
	var err error
	if md == nil || cfg.outputColorspace != SRGB_COLORSPACE {
		return nil
	}
	if md.CICP.IsSet && !md.CICP.IsSRGB() {
		p := md.CICP.PipelineToSRGB()
		if p == nil {
			return fmt.Errorf("cannot convert colorspace, unknown %s", md.CICP)
		}
		for _, f := range images {
			if f.Image, err = convert(p, f.Image); err != nil {
				return err
			}
		}
		return nil
	}
	profile, err := md.ICCProfile()
	if err != nil {
		return err
	}
	if profile != nil {
		for _, f := range images {
			if f.Image, err = ConvertToSRGB(profile, cfg.rendering_intent, cfg.use_blackpoint_compensation, f.Image); err != nil {
				return err
			}
		}
	}
	return nil
}

func fix_orientation(ans *Image, md *meta.Data, cfg *decodeConfig) error {
	if md == nil || !cfg.autoOrientation {
		return nil
	}
	exif_data, err := md.Exif()
	if err != nil {
		return err
	}
	var oval orientation = orientationUnspecified
	if exif_data != nil {
		orient, err := exif_data.Get(exif.Orientation)
		if err == nil && orient != nil && orient.Format() == exif_tiff.IntVal {
			if x, err := orient.Int(0); err == nil && x > 0 && x < 9 {
				oval = orientation(x)
			}
		}
	}
	switch oval {
	case orientationNormal, orientationUnspecified:
	case orientationFlipH:
		ans.FlipH()
	case orientationFlipV:
		ans.FlipV()
	case orientationRotate90:
		ans.Rotate90()
	case orientationRotate180:
		ans.Rotate180()
	case orientationRotate270:
		ans.Rotate270()
	case orientationTranspose:
		ans.Transpose()
	case orientationTransverse:
		ans.Transverse()
	}
	return nil
}

func (img *Image) Transform(t types.TransformType) {
	switch t {
	case types.TransverseTransform:
		img.Transverse()
	case types.TransposeTransform:
		img.Transpose()
	case types.FlipHTransform:
		img.FlipH()
	case types.FlipVTransform:
		img.FlipV()
	case types.Rotate90Transform:
		img.Rotate90()
	case types.Rotate180Transform:
		img.Rotate180()
	case types.Rotate270Transform:
		img.Rotate270()
	}
}

const (
	NoTransform         = types.NoTransform
	FlipHTransform      = types.FlipHTransform
	FlipVTransform      = types.FlipVTransform
	Rotate90Transform   = types.Rotate90Transform
	Rotate180Transform  = types.Rotate180Transform
	Rotate270Transform  = types.Rotate270Transform
	TransverseTransform = types.TransverseTransform
	TransposeTransform  = types.TransposeTransform
)

func format_from_decode_result(x string) Format {
	switch x {
	case "BMP":
		return BMP
	case "TIFF", "TIF":
		return TIFF
	}
	return UNKNOWN
}

func decode_all_go(r io.Reader, md *meta.Data, cfg *decodeConfig) (ans *Image, err error) {
	defer func() {
		if ans == nil || err != nil || ans.Metadata == nil {
			return
		}
		if cfg.outputColorspace != NO_CHANGE_OF_COLORSPACE {
			if err = fix_colors(ans.Frames, ans.Metadata, cfg); err != nil {
				return
			}
		}
		if cfg.autoOrientation {
			if err = fix_orientation(ans, ans.Metadata, cfg); err != nil {
				return
			}
		}
		if cfg.background != nil {
			ans.PasteOntoBackground(*cfg.background)
		}
		if cfg.transform != types.NoTransform {
			ans.Transform(cfg.transform)
		}
		if cfg.resize != nil {
			w, h := ans.Bounds().Dx(), ans.Bounds().Dy()
			nw, nh := cfg.resize(w, h)
			if nw != w || nh != h {
				ans.Resize(nw, nh, Lanczos)
			}
		}
		ans.Metadata.PixelWidth = uint32(ans.Bounds().Dx())
		ans.Metadata.PixelHeight = uint32(ans.Bounds().Dy())
	}()
	if md == nil {
		img, imgf, err := image.Decode(r)
		if err != nil {
			return nil, err
		}
		m := meta.Data{
			PixelWidth:       uint32(img.Bounds().Dx()),
			PixelHeight:      uint32(img.Bounds().Dy()),
			Format:           format_from_decode_result(imgf),
			BitsPerComponent: tiffmeta.BitsPerComponent(img.ColorModel()),
		}
		f := Frame{Image: img}
		return &Image{Metadata: &m, Frames: []*Frame{&f}}, nil
	}
	ans = &Image{Metadata: md}
	if md.HasFrames {
		switch md.Format {
		case GIF:
			g, err := gif.DecodeAll(r)
			if err != nil {
				return nil, err
			}
			ans.populate_from_gif(g)
		case PNG:
			png, err := apng.DecodeAll(r)
			if err != nil {
				return nil, err
			}
			ans.populate_from_apng(&png)
		case WEBP:
			wp, err := webp.DecodeAnimated(r)
			if err != nil {
				return nil, err
			}
			ans.populate_from_webp(wp)
		}
		ans.Metadata.NumFrames = len(ans.Frames)
		ans.Metadata.NumPlays = int(ans.LoopCount)
	} else {
		var img image.Image
		switch md.Format {
		case JPEG:
			img, err = myjpeg.Decode(r)
		case PNG:
			img, err = apng.Decode(r)
		default:
			img, _, err = image.Decode(r)
		}
		if err != nil {
			return nil, err
		}
		ans.Metadata.PixelWidth = uint32(img.Bounds().Dx())
		ans.Metadata.PixelHeight = uint32(img.Bounds().Dy())
		ans.Frames = append(ans.Frames, &Frame{Image: img})
	}
	return
}

func decode_all_magick(inp *types.Input, md *meta.Data, cfg *decodeConfig) (ans *Image, err error) {
	mi, err := magick.OpenAll(inp, md, cfg.magick_callback)
	if err != nil {
		return nil, err
	}
	ans = &Image{Metadata: md}
	for _, f := range mi.Frames {
		fr := &Frame{
			Number: uint(f.Number), TopLeft: image.Pt(f.Left, f.Top), Image: f.Img,
			Delay: time.Millisecond * time.Duration(f.Delay_ms), ComposeOnto: uint(f.Compose_onto),
			Replace: f.Replace,
		}
		ans.Frames = append(ans.Frames, fr)
	}
	if md != nil {
		// in case of transforms/auto-orient
		b := ans.Bounds()
		md.PixelWidth, md.PixelHeight = uint32(b.Dx()), uint32(b.Dy())
	}
	return
}

func decode_all(inp *types.Input, opts []DecodeOption) (ans *Image, err error) {
	cfg := NewDecodeConfig(opts...)
	if !magick.HasMagick() {
		cfg.backends = slices.DeleteFunc(cfg.backends, func(b Backend) bool { return b == MAGICK_IMAGE })
	}
	if len(cfg.backends) == 0 {
		return nil, fmt.Errorf("the magick command was not found in PATH")
	}

	if inp.Reader == nil {
		var f *os.File
		f, err = mockable_fs.Open(inp.Path)
		if err != nil {
			return
		}
		defer f.Close()
		inp.Reader = f
	}
	var md *meta.Data
	md, inp.Reader, err = autometa.Load(inp.Reader)
	if err != nil {
		return
	}
	var backend_err error
	for _, backend := range cfg.backends {
		switch backend {
		case GO_IMAGE:
			inp.Reader, backend_err = streams.CallbackWithSeekable(inp.Reader, func(r io.Reader) (err error) {
				ans, err = decode_all_go(r, md, cfg)
				return
			})
		case MAGICK_IMAGE:
			inp.Reader, backend_err = streams.CallbackWithSeekable(inp.Reader, func(r io.Reader) (err error) {
				i := *inp
				i.Reader = r
				ans, err = decode_all_magick(&i, md, cfg)
				return
			})
		}
		if backend_err == nil && ans != nil {
			return
		}
	}
	if ans == nil && backend_err == nil {
		backend_err = fmt.Errorf("unrecognised image format")
	}
	return ans, backend_err
}

// Decode image from r including all animation frames if its an animated image.
// Returns nil with no error when no supported image is found in r.
// Also returns a reader that will yield all bytes from r so that this API does
// not exhaust r.
func DecodeAll(r io.Reader, opts ...DecodeOption) (ans *Image, s io.Reader, err error) {
	inp := &types.Input{Reader: r}
	ans, err = decode_all(inp, opts)
	s = inp.Reader
	return
}

func (ans *Image) SingleFrame() image.Image {
	if ans.DefaultImage != nil {
		return ans.DefaultImage
	}
	return ans.Frames[0].Image

}

// Decode reads an image from r.
func Decode(r io.Reader, opts ...DecodeOption) (image.Image, error) {
	ans, _, err := DecodeAll(r, opts...)
	if err != nil {
		return nil, err
	}
	return ans.SingleFrame(), nil
}

// Open loads an image from file.
//
// Examples:
//
//	// Load an image from file.
//	img, err := imaging.Open("test.jpg")
func Open(filename string, opts ...DecodeOption) (image.Image, error) {
	ans, err := OpenAll(filename, opts...)
	if err != nil {
		return nil, err
	}
	return ans.SingleFrame(), nil
}

func OpenAll(filename string, opts ...DecodeOption) (*Image, error) {
	return decode_all(&types.Input{Path: filename}, opts)
}

func OpenConfig(filename string) (ans image.Config, format_name string, err error) {
	file, err := mockable_fs.Open(filename)
	if err != nil {
		return ans, "", err
	}
	defer file.Close()
	return image.DecodeConfig(file)
}

type Format = types.Format

const (
	UNKNOWN = types.UNKNOWN
	JPEG    = types.JPEG
	PNG     = types.PNG
	GIF     = types.GIF
	TIFF    = types.TIFF
	WEBP    = types.WEBP
	BMP     = types.BMP
	PBM     = types.PBM
	PGM     = types.PGM
	PPM     = types.PPM
	PAM     = types.PAM
)

// ErrUnsupportedFormat means the given image format is not supported.
var ErrUnsupportedFormat = errors.New("imaging: unsupported image format")

// FormatFromExtension parses image format from filename extension:
// "jpg" (or "jpeg"), "png", "gif", "tif" (or "tiff") and "bmp" are supported.
func FormatFromExtension(ext string) (Format, error) {
	if f, ok := types.FormatExts[strings.ToLower(strings.TrimPrefix(ext, "."))]; ok {
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
		if nrgba, ok := img.(*image.NRGBA); ok && IsOpaque(nrgba) {
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
	file, err := mockable_fs.Create(filename)
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
