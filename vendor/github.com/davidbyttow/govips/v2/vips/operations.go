package vips

// #cgo CFLAGS: -std=c99
// #include "operations.h"
import "C"
import (
	"errors"
	"os"
	"runtime"
	"strings"
	"unsafe"
)

// Arithmetic

// https://libvips.github.io/libvips/API/current/libvips-arithmetic.html#vips-find-trim
func vipsFindTrim(in *C.VipsImage, threshold float64, backgroundColor *Color) (int, int, int, int, error) {
	incOpCounter("findTrim")
	var left, top, width, height C.int

	if err := C.find_trim(in, &left, &top, &width, &height, C.double(threshold), C.double(backgroundColor.R),
		C.double(backgroundColor.G), C.double(backgroundColor.B)); err != 0 {
		return -1, -1, -1, -1, handleVipsError()
	}

	return int(left), int(top), int(width), int(height), nil
}

// https://libvips.github.io/libvips/API/current/libvips-arithmetic.html#vips-getpoint
func vipsGetPoint(in *C.VipsImage, n int, x int, y int) ([]float64, error) {
	incOpCounter("getpoint")
	var out *C.double
	defer gFreePointer(unsafe.Pointer(out))

	if err := C.getpoint(in, &out, C.int(n), C.int(x), C.int(y)); err != 0 {
		return nil, handleVipsError()
	}

	// maximum n is 4
	return (*[4]float64)(unsafe.Pointer(out))[:n:n], nil
}

// https://www.libvips.org/API/current/libvips-arithmetic.html#vips-min
func vipsMin(in *C.VipsImage) (float64, int, int, error) {
	incOpCounter("min")
	var out C.double
	var x, y C.int

	if err := C.minOp(in, &out, &x, &y, C.int(1)); err != 0 {
		return 0, 0, 0, handleVipsError()
	}

	return float64(out), int(x), int(y), nil
}

// Color

// Color represents an RGB
type Color struct {
	R, G, B uint8
}

// ColorRGBA represents an RGB with alpha channel (A)
type ColorRGBA struct {
	R, G, B, A uint8
}

// Interpretation represents VIPS_INTERPRETATION type
type Interpretation int

// Interpretation enum
const (
	InterpretationError     Interpretation = C.VIPS_INTERPRETATION_ERROR
	InterpretationMultiband Interpretation = C.VIPS_INTERPRETATION_MULTIBAND
	InterpretationBW        Interpretation = C.VIPS_INTERPRETATION_B_W
	InterpretationHistogram Interpretation = C.VIPS_INTERPRETATION_HISTOGRAM
	InterpretationXYZ       Interpretation = C.VIPS_INTERPRETATION_XYZ
	InterpretationLAB       Interpretation = C.VIPS_INTERPRETATION_LAB
	InterpretationCMYK      Interpretation = C.VIPS_INTERPRETATION_CMYK
	InterpretationLABQ      Interpretation = C.VIPS_INTERPRETATION_LABQ
	InterpretationRGB       Interpretation = C.VIPS_INTERPRETATION_RGB
	InterpretationRGB16     Interpretation = C.VIPS_INTERPRETATION_RGB16
	InterpretationCMC       Interpretation = C.VIPS_INTERPRETATION_CMC
	InterpretationLCH       Interpretation = C.VIPS_INTERPRETATION_LCH
	InterpretationLABS      Interpretation = C.VIPS_INTERPRETATION_LABS
	InterpretationSRGB      Interpretation = C.VIPS_INTERPRETATION_sRGB
	InterpretationYXY       Interpretation = C.VIPS_INTERPRETATION_YXY
	InterpretationFourier   Interpretation = C.VIPS_INTERPRETATION_FOURIER
	InterpretationGrey16    Interpretation = C.VIPS_INTERPRETATION_GREY16
	InterpretationMatrix    Interpretation = C.VIPS_INTERPRETATION_MATRIX
	InterpretationScRGB     Interpretation = C.VIPS_INTERPRETATION_scRGB
	InterpretationHSV       Interpretation = C.VIPS_INTERPRETATION_HSV
)

// Intent represents VIPS_INTENT type
type Intent int

// Intent enum
const (
	IntentPerceptual Intent = C.VIPS_INTENT_PERCEPTUAL
	IntentRelative   Intent = C.VIPS_INTENT_RELATIVE
	IntentSaturation Intent = C.VIPS_INTENT_SATURATION
	IntentAbsolute   Intent = C.VIPS_INTENT_ABSOLUTE
	IntentLast       Intent = C.VIPS_INTENT_LAST
)

func vipsIsColorSpaceSupported(in *C.VipsImage) bool {
	return C.is_colorspace_supported(in) == 1
}

// https://libvips.github.io/libvips/API/current/libvips-colour.html#vips-colourspace
func vipsToColorSpace(in *C.VipsImage, interpretation Interpretation) (*C.VipsImage, error) {
	incOpCounter("to_colorspace")
	var out *C.VipsImage

	inter := C.VipsInterpretation(interpretation)

	if err := C.to_colorspace(in, &out, inter); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

func vipsICCTransform(in *C.VipsImage, outputProfile string, inputProfile string, intent Intent, depth int,
	embedded bool) (*C.VipsImage, error) {
	var out *C.VipsImage
	var cInputProfile *C.char
	var cEmbedded C.gboolean

	cOutputProfile := C.CString(outputProfile)
	defer freeCString(cOutputProfile)

	if inputProfile != "" {
		cInputProfile = C.CString(inputProfile)
		defer freeCString(cInputProfile)
	}

	if embedded {
		cEmbedded = C.TRUE
	}

	if res := C.icc_transform(in, &out, cOutputProfile, cInputProfile, C.VipsIntent(intent), C.int(depth), cEmbedded); res != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// Composite

// ImageComposite image to composite param
type ImageComposite struct {
	Image     *ImageRef
	BlendMode BlendMode
	X, Y      int
}

func toVipsCompositeStructs(r *ImageRef, datas []*ImageComposite) ([]*C.VipsImage, []C.int, []C.int, []C.int) {
	ins := []*C.VipsImage{r.image}
	modes := []C.int{}
	xs := []C.int{}
	ys := []C.int{}

	for _, image := range datas {
		ins = append(ins, image.Image.image)
		modes = append(modes, C.int(image.BlendMode))
		xs = append(xs, C.int(image.X))
		ys = append(ys, C.int(image.Y))
	}

	return ins, modes, xs, ys
}

// Conversion

// BandFormat represents VIPS_FORMAT type
type BandFormat int

// BandFormat enum
const (
	BandFormatNotSet    BandFormat = C.VIPS_FORMAT_NOTSET
	BandFormatUchar     BandFormat = C.VIPS_FORMAT_UCHAR
	BandFormatChar      BandFormat = C.VIPS_FORMAT_CHAR
	BandFormatUshort    BandFormat = C.VIPS_FORMAT_USHORT
	BandFormatShort     BandFormat = C.VIPS_FORMAT_SHORT
	BandFormatUint      BandFormat = C.VIPS_FORMAT_UINT
	BandFormatInt       BandFormat = C.VIPS_FORMAT_INT
	BandFormatFloat     BandFormat = C.VIPS_FORMAT_FLOAT
	BandFormatComplex   BandFormat = C.VIPS_FORMAT_COMPLEX
	BandFormatDouble    BandFormat = C.VIPS_FORMAT_DOUBLE
	BandFormatDpComplex BandFormat = C.VIPS_FORMAT_DPCOMPLEX
)

// BlendMode gives the various Porter-Duff and PDF blend modes.
// See https://libvips.github.io/libvips/API/current/libvips-conversion.html#VipsBlendMode
type BlendMode int

// Constants define the various Porter-Duff and PDF blend modes.
// See https://libvips.github.io/libvips/API/current/libvips-conversion.html#VipsBlendMode
const (
	BlendModeClear      BlendMode = C.VIPS_BLEND_MODE_CLEAR
	BlendModeSource     BlendMode = C.VIPS_BLEND_MODE_SOURCE
	BlendModeOver       BlendMode = C.VIPS_BLEND_MODE_OVER
	BlendModeIn         BlendMode = C.VIPS_BLEND_MODE_IN
	BlendModeOut        BlendMode = C.VIPS_BLEND_MODE_OUT
	BlendModeAtop       BlendMode = C.VIPS_BLEND_MODE_ATOP
	BlendModeDest       BlendMode = C.VIPS_BLEND_MODE_DEST
	BlendModeDestOver   BlendMode = C.VIPS_BLEND_MODE_DEST_OVER
	BlendModeDestIn     BlendMode = C.VIPS_BLEND_MODE_DEST_IN
	BlendModeDestOut    BlendMode = C.VIPS_BLEND_MODE_DEST_OUT
	BlendModeDestAtop   BlendMode = C.VIPS_BLEND_MODE_DEST_ATOP
	BlendModeXOR        BlendMode = C.VIPS_BLEND_MODE_XOR
	BlendModeAdd        BlendMode = C.VIPS_BLEND_MODE_ADD
	BlendModeSaturate   BlendMode = C.VIPS_BLEND_MODE_SATURATE
	BlendModeMultiply   BlendMode = C.VIPS_BLEND_MODE_MULTIPLY
	BlendModeScreen     BlendMode = C.VIPS_BLEND_MODE_SCREEN
	BlendModeOverlay    BlendMode = C.VIPS_BLEND_MODE_OVERLAY
	BlendModeDarken     BlendMode = C.VIPS_BLEND_MODE_DARKEN
	BlendModeLighten    BlendMode = C.VIPS_BLEND_MODE_LIGHTEN
	BlendModeColorDodge BlendMode = C.VIPS_BLEND_MODE_COLOUR_DODGE
	BlendModeColorBurn  BlendMode = C.VIPS_BLEND_MODE_COLOUR_BURN
	BlendModeHardLight  BlendMode = C.VIPS_BLEND_MODE_HARD_LIGHT
	BlendModeSoftLight  BlendMode = C.VIPS_BLEND_MODE_SOFT_LIGHT
	BlendModeDifference BlendMode = C.VIPS_BLEND_MODE_DIFFERENCE
	BlendModeExclusion  BlendMode = C.VIPS_BLEND_MODE_EXCLUSION
)

// Gravity represents VIPS_GRAVITY type
type Gravity int

// Gravity enum
const (
	GravityCentre    Gravity = C.VIPS_COMPASS_DIRECTION_CENTRE
	GravityNorth     Gravity = C.VIPS_COMPASS_DIRECTION_NORTH
	GravityEast      Gravity = C.VIPS_COMPASS_DIRECTION_EAST
	GravitySouth     Gravity = C.VIPS_COMPASS_DIRECTION_SOUTH
	GravityWest      Gravity = C.VIPS_COMPASS_DIRECTION_WEST
	GravityNorthEast Gravity = C.VIPS_COMPASS_DIRECTION_NORTH_EAST
	GravityNorthWest Gravity = C.VIPS_COMPASS_DIRECTION_NORTH_WEST
	GravitySouthEast Gravity = C.VIPS_COMPASS_DIRECTION_SOUTH_EAST
	GravitySouthWest Gravity = C.VIPS_COMPASS_DIRECTION_SOUTH_WEST
)

// Direction represents VIPS_DIRECTION type
type Direction int

// Direction enum
const (
	DirectionHorizontal Direction = C.VIPS_DIRECTION_HORIZONTAL
	DirectionVertical   Direction = C.VIPS_DIRECTION_VERTICAL
)

// Angle represents VIPS_ANGLE type
type Angle int

// Angle enum
const (
	Angle0   Angle = C.VIPS_ANGLE_D0
	Angle90  Angle = C.VIPS_ANGLE_D90
	Angle180 Angle = C.VIPS_ANGLE_D180
	Angle270 Angle = C.VIPS_ANGLE_D270
)

// Angle45 represents VIPS_ANGLE45 type
type Angle45 int

// Angle45 enum
const (
	Angle45_0   Angle45 = C.VIPS_ANGLE45_D0
	Angle45_45  Angle45 = C.VIPS_ANGLE45_D45
	Angle45_90  Angle45 = C.VIPS_ANGLE45_D90
	Angle45_135 Angle45 = C.VIPS_ANGLE45_D135
	Angle45_180 Angle45 = C.VIPS_ANGLE45_D180
	Angle45_225 Angle45 = C.VIPS_ANGLE45_D225
	Angle45_270 Angle45 = C.VIPS_ANGLE45_D270
	Angle45_315 Angle45 = C.VIPS_ANGLE45_D315
)

// ExtendStrategy represents VIPS_EXTEND type
type ExtendStrategy int

// ExtendStrategy enum
const (
	ExtendBlack      ExtendStrategy = C.VIPS_EXTEND_BLACK
	ExtendCopy       ExtendStrategy = C.VIPS_EXTEND_COPY
	ExtendRepeat     ExtendStrategy = C.VIPS_EXTEND_REPEAT
	ExtendMirror     ExtendStrategy = C.VIPS_EXTEND_MIRROR
	ExtendWhite      ExtendStrategy = C.VIPS_EXTEND_WHITE
	ExtendBackground ExtendStrategy = C.VIPS_EXTEND_BACKGROUND
)

// Interesting represents VIPS_INTERESTING type
// https://libvips.github.io/libvips/API/current/libvips-conversion.html#VipsInteresting
type Interesting int

// Interesting constants represent areas of interest which smart cropping will crop based on.
const (
	InterestingNone      Interesting = C.VIPS_INTERESTING_NONE
	InterestingCentre    Interesting = C.VIPS_INTERESTING_CENTRE
	InterestingEntropy   Interesting = C.VIPS_INTERESTING_ENTROPY
	InterestingAttention Interesting = C.VIPS_INTERESTING_ATTENTION
	InterestingLow       Interesting = C.VIPS_INTERESTING_LOW
	InterestingHigh      Interesting = C.VIPS_INTERESTING_HIGH
	InterestingAll       Interesting = C.VIPS_INTERESTING_ALL
	InterestingLast      Interesting = C.VIPS_INTERESTING_LAST
)

// https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-embed
func vipsEmbed(in *C.VipsImage, left, top, width, height int, extend ExtendStrategy) (*C.VipsImage, error) {
	incOpCounter("embed")
	var out *C.VipsImage

	if err := C.embed_image(in, &out, C.int(left), C.int(top), C.int(width), C.int(height), C.int(extend)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-embed
func vipsEmbedBackground(in *C.VipsImage, left, top, width, height int, backgroundColor *ColorRGBA) (*C.VipsImage, error) {
	incOpCounter("embed")
	var out *C.VipsImage

	if err := C.embed_image_background(in, &out, C.int(left), C.int(top), C.int(width),
		C.int(height), C.double(backgroundColor.R),
		C.double(backgroundColor.G), C.double(backgroundColor.B), C.double(backgroundColor.A)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

func vipsEmbedMultiPage(in *C.VipsImage, left, top, width, height int, extend ExtendStrategy) (*C.VipsImage, error) {
	incOpCounter("embedMultiPage")
	var out *C.VipsImage

	if err := C.embed_multi_page_image(in, &out, C.int(left), C.int(top), C.int(width), C.int(height), C.int(extend)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

func vipsEmbedMultiPageBackground(in *C.VipsImage, left, top, width, height int, backgroundColor *ColorRGBA) (*C.VipsImage, error) {
	incOpCounter("embedMultiPageBackground")
	var out *C.VipsImage

	if err := C.embed_multi_page_image_background(in, &out, C.int(left), C.int(top), C.int(width),
		C.int(height), C.double(backgroundColor.R),
		C.double(backgroundColor.G), C.double(backgroundColor.B), C.double(backgroundColor.A)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-flip
func vipsFlip(in *C.VipsImage, direction Direction) (*C.VipsImage, error) {
	return vipsGenFlip(in, direction)
}

// https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-extract-area
func vipsExtractArea(in *C.VipsImage, left, top, width, height int) (*C.VipsImage, error) {
	return vipsGenExtractArea(in, left, top, width, height)
}

func vipsExtractAreaMultiPage(in *C.VipsImage, left, top, width, height int) (*C.VipsImage, error) {
	incOpCounter("extractAreaMultiPage")

	pageHeight := vipsGetPageHeight(in)
	nPages := int(in.Ysize) / pageHeight

	pages := make([]*C.VipsImage, nPages)
	for i := 0; i < nPages; i++ {
		page, err := vipsGenExtractArea(in, left, pageHeight*i+top, width, height)
		if err != nil {
			for j := 0; j < i; j++ {
				clearImage(pages[j])
			}
			return nil, err
		}
		pages[i] = page
	}

	across := 1
	joined, err := vipsGenArrayjoin(pages, &ArrayjoinOptions{Across: &across})
	for _, p := range pages {
		clearImage(p)
	}
	if err != nil {
		return nil, err
	}

	out, err := vipsGenCopy(joined, nil)
	clearImage(joined)
	if err != nil {
		return nil, err
	}

	vipsSetPageHeight(out, height)
	return out, nil
}

// http://libvips.github.io/libvips/API/current/libvips-resample.html#vips-similarity
func vipsSimilarity(in *C.VipsImage, scale float64, angle float64, color *ColorRGBA,
	idx float64, idy float64, odx float64, ody float64) (*C.VipsImage, error) {
	incOpCounter("similarity")
	var out *C.VipsImage

	if err := C.similarity(in, &out, C.double(scale), C.double(angle),
		C.double(color.R), C.double(color.G), C.double(color.B), C.double(color.A),
		C.double(idx), C.double(idy), C.double(odx), C.double(ody)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// http://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-smartcrop
func vipsSmartCrop(in *C.VipsImage, width int, height int, interesting Interesting) (*C.VipsImage, error) {
	_, out, _, err := vipsGenSmartcrop(in, width, height, &SmartcropOptions{Interesting: &interesting})
	return out, err
}

// http://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-crop
func vipsCrop(in *C.VipsImage, left int, top int, width int, height int) (*C.VipsImage, error) {
	incOpCounter("crop")
	var out *C.VipsImage

	if err := C.crop(in, &out, C.int(left), C.int(top), C.int(width), C.int(height)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-composite
func vipsComposite(ins []*C.VipsImage, modes []C.int, xs, ys []C.int) (*C.VipsImage, error) {
	if len(ins) == 0 || len(modes) == 0 || len(xs) == 0 || len(ys) == 0 {
		return nil, errors.New("vipsComposite: empty input slice")
	}
	incOpCounter("composite_multi")
	var out *C.VipsImage

	if err := C.composite_image(&ins[0], &out, C.int(len(ins)), &modes[0], &xs[0], &ys[0]); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://libvips.github.io/libvips/API/current/libvips-conversion.html#vips-join
func vipsJoin(input1 *C.VipsImage, input2 *C.VipsImage, dir Direction) (*C.VipsImage, error) {
	incOpCounter("join")
	var out *C.VipsImage

	defer C.g_object_unref(C.gpointer(input1))
	defer C.g_object_unref(C.gpointer(input2))
	if err := C.join(input1, input2, &out, C.int(dir)); err != 0 {
		return nil, handleVipsError()
	}
	return out, nil
}

func vipsAddAlpha(in *C.VipsImage) (*C.VipsImage, error) {
	incOpCounter("addalpha")
	var out *C.VipsImage

	if err := C.add_alpha(in, &out); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// Create

type TextWrap int

type TextParams struct {
	Text      string
	Font      string
	Width     int
	Height    int
	Alignment Align
	DPI       int
	RGBA      bool
	Justify   bool
	Spacing   int
	Wrap      TextWrap
}

type vipsTextOptions struct {
	Text      *C.char
	Font      *C.char
	Width     C.int
	Height    C.int
	DPI       C.int
	RGBA      C.gboolean
	Justify   C.gboolean
	Spacing   C.int
	Alignment C.VipsAlign
	Wrap      C.VipsTextWrap
}

// TextWrap enum
const (
	TextWrapWord     TextWrap = C.VIPS_TEXT_WRAP_WORD
	TextWrapChar     TextWrap = C.VIPS_TEXT_WRAP_CHAR
	TextWrapWordChar TextWrap = C.VIPS_TEXT_WRAP_WORD_CHAR
	TextWrapNone     TextWrap = C.VIPS_TEXT_WRAP_NONE
)

// https://libvips.github.io/libvips/API/current/libvips-create.html#vips-text
func vipsText(params *TextParams) (*C.VipsImage, error) {
	var out *C.VipsImage

	text := C.CString(params.Text)
	defer freeCString(text)

	font := C.CString(params.Font)
	defer freeCString(font)

	opts := vipsTextOptions{
		Text:      text,
		Font:      font,
		Width:     C.int(params.Width),
		Height:    C.int(params.Height),
		DPI:       C.int(params.DPI),
		Alignment: C.VipsAlign(params.Alignment),
		Spacing:   C.int(params.Spacing),
		Wrap:      C.VipsTextWrap(params.Wrap),
	}

	if params.RGBA {
		opts.RGBA = C.TRUE
	}

	if params.Justify {
		opts.Justify = C.TRUE
	}

	err := C.text(&out, (*C.TextOptions)(unsafe.Pointer(&opts)))
	if err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// Draw

// https://libvips.github.io/libvips/API/current/libvips-draw.html#vips-draw-rect
func vipsDrawRect(in *C.VipsImage, color ColorRGBA, left int, top int, width int, height int, fill bool) error {
	incOpCounter("draw_rect")

	fillBit := 0
	if fill {
		fillBit = 1
	}

	if err := C.draw_rect(in, C.double(color.R), C.double(color.G), C.double(color.B), C.double(color.A),
		C.int(left), C.int(top), C.int(width), C.int(height), C.int(fillBit)); err != 0 {
		return handleImageError(in)
	}

	return nil
}

// Header

func vipsHasICCProfile(in *C.VipsImage) bool {
	return int(C.has_icc_profile(in)) != 0
}

func vipsGetICCProfile(in *C.VipsImage) ([]byte, bool) {
	var bufPtr unsafe.Pointer
	var dataLength C.size_t

	if int(C.get_icc_profile(in, &bufPtr, &dataLength)) != 0 {
		return nil, false
	}

	buf := C.GoBytes(bufPtr, C.int(dataLength))
	return buf, true
}

func vipsRemoveICCProfile(in *C.VipsImage) bool {
	return fromGboolean(C.remove_icc_profile(in))
}

func vipsHasIPTC(in *C.VipsImage) bool {
	return int(C.has_iptc(in)) != 0
}

func vipsImageGetFields(in *C.VipsImage) (fields []string) {
	const maxFields = 256

	rawFields := C.image_get_fields(in)
	defer C.g_strfreev(rawFields)

	cFields := (*[maxFields]*C.char)(unsafe.Pointer(rawFields))[:maxFields:maxFields]

	for _, field := range cFields {
		if field == nil {
			break
		}
		fields = append(fields, C.GoString(field))
	}
	return
}

func vipsImageGetExifData(in *C.VipsImage) map[string]string {
	fields := vipsImageGetFields(in)

	exifData := map[string]string{}
	for _, field := range fields {
		if strings.HasPrefix(field, "exif") {
			exifData[field] = vipsImageGetString(in, field)
		}
	}

	return exifData
}

func vipsRemoveMetadata(in *C.VipsImage, keep ...string) {
	fields := vipsImageGetFields(in)

	retain := append(keep, technicalMetadata...)

	for _, field := range fields {
		if contains(retain, field) {
			continue
		}

		cField := C.CString(field)

		C.remove_field(in, cField)

		C.free(unsafe.Pointer(cField))
	}
}

var technicalMetadata = []string{
	C.VIPS_META_ICC_NAME,
	C.VIPS_META_ORIENTATION,
	C.VIPS_META_N_PAGES,
	C.VIPS_META_PAGE_HEIGHT,
	"delay",
	"loop",
}

func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func vipsGetMetaOrientation(in *C.VipsImage) int {
	return int(C.get_meta_orientation(in))
}

func vipsRemoveMetaOrientation(in *C.VipsImage) {
	C.remove_meta_orientation(in)
}

func vipsSetMetaOrientation(in *C.VipsImage, orientation int) {
	C.set_meta_orientation(in, C.int(orientation))
}

func vipsGetImageNPages(in *C.VipsImage) int {
	return int(C.get_image_n_pages(in))
}

func vipsSetImageNPages(in *C.VipsImage, pages int) {
	C.set_image_n_pages(in, C.int(pages))
}

func vipsGetPageHeight(in *C.VipsImage) int {
	return int(C.get_page_height(in))
}

func vipsSetPageHeight(in *C.VipsImage, height int) {
	C.set_page_height(in, C.int(height))
}

func vipsImageGetMetaLoader(in *C.VipsImage) (string, bool) {
	var out *C.char
	defer freeCString(out)
	code := int(C.get_meta_loader(in, &out))
	return C.GoString(out), code == 0
}

func vipsImageGetDelay(in *C.VipsImage, n int) ([]int, error) {
	incOpCounter("imageGetDelay")
	var out *C.int
	defer gFreePointer(unsafe.Pointer(out))

	if err := C.get_image_delay(in, &out); err != 0 {
		return nil, handleVipsError()
	}
	return fromCArrayInt(out, n), nil
}

func vipsImageSetDelay(in *C.VipsImage, data []C.int) error {
	incOpCounter("imageSetDelay")
	if n := len(data); n > 0 {
		C.set_image_delay(in, &data[0], C.int(n))
	}
	return nil
}

func vipsImageGetLoop(in *C.VipsImage) int {
	return int(C.get_image_loop(in))
}

func vipsImageSetLoop(in *C.VipsImage, loop int) {
	C.set_image_loop(in, C.int(loop))
}

func vipsImageGetBackground(in *C.VipsImage) ([]float64, error) {
	incOpCounter("imageGetBackground")
	var out *C.double
	var n C.int
	defer gFreePointer(unsafe.Pointer(out))

	if err := C.get_background(in, &out, &n); err != 0 {
		return nil, handleVipsError()
	}
	return fromCArrayDouble(out, int(n)), nil
}

// vipsDetermineImageTypeFromMetaLoader determine the image type from vips-loader metadata
func vipsDetermineImageTypeFromMetaLoader(in *C.VipsImage) ImageType {
	vipsLoader, ok := vipsImageGetMetaLoader(in)
	if vipsLoader == "" || !ok {
		return ImageTypeUnknown
	}
	if strings.HasPrefix(vipsLoader, "jpeg") {
		return ImageTypeJPEG
	}
	if strings.HasPrefix(vipsLoader, "png") {
		return ImageTypePNG
	}
	if strings.HasPrefix(vipsLoader, "gif") {
		return ImageTypeGIF
	}
	if strings.HasPrefix(vipsLoader, "svg") {
		return ImageTypeSVG
	}
	if strings.HasPrefix(vipsLoader, "webp") {
		return ImageTypeWEBP
	}
	if strings.HasPrefix(vipsLoader, "jp2k") {
		return ImageTypeJP2K
	}
	if strings.HasPrefix(vipsLoader, "jxl") {
		return ImageTypeJXL
	}
	if strings.HasPrefix(vipsLoader, "magick") {
		return ImageTypeMagick
	}
	if strings.HasPrefix(vipsLoader, "tiff") {
		return ImageTypeTIFF
	}
	if strings.HasPrefix(vipsLoader, "heif") {
		return ImageTypeHEIF
	}
	if strings.HasPrefix(vipsLoader, "pdf") {
		return ImageTypePDF
	}
	return ImageTypeUnknown
}

func vipsImageSetBlob(in *C.VipsImage, name string, data []byte) {
	cData := unsafe.Pointer(&data)
	cDataLength := C.size_t(len(data))

	cField := C.CString(name)
	defer freeCString(cField)
	C.image_set_blob(in, cField, cData, cDataLength)
}

func vipsImageGetBlob(in *C.VipsImage, name string) []byte {
	var bufPtr unsafe.Pointer
	var dataLength C.size_t

	cField := C.CString(name)
	defer freeCString(cField)
	if int(C.image_get_blob(in, cField, &bufPtr, &dataLength)) != 0 {
		return nil
	}

	buf := C.GoBytes(bufPtr, C.int(dataLength))
	return buf
}

func vipsImageSetDouble(in *C.VipsImage, name string, f float64) {
	cField := C.CString(name)
	defer freeCString(cField)

	cDouble := C.double(f)
	C.image_set_double(in, cField, cDouble)
}

func vipsImageGetDouble(in *C.VipsImage, name string) float64 {
	cField := C.CString(name)
	defer freeCString(cField)

	var cDouble C.double
	if int(C.image_get_double(in, cField, &cDouble)) == 0 {
		return float64(cDouble)
	}

	return 0
}

func vipsImageSetInt(in *C.VipsImage, name string, i int) {
	cField := C.CString(name)
	defer freeCString(cField)

	cInt := C.int(i)
	C.image_set_int(in, cField, cInt)
}

func vipsImageGetInt(in *C.VipsImage, name string) int {
	cField := C.CString(name)
	defer freeCString(cField)

	var cInt C.int
	if int(C.image_get_int(in, cField, &cInt)) == 0 {
		return int(cInt)
	}

	return 0
}

func vipsImageSetString(in *C.VipsImage, name string, str string) {
	cField := C.CString(name)
	defer freeCString(cField)

	cStr := C.CString(str)
	defer freeCString(cStr)

	C.image_set_string(in, cField, cStr)
}

func vipsImageGetString(in *C.VipsImage, name string) string {
	cField := C.CString(name)
	defer freeCString(cField)
	var cFieldValue *C.char
	defer freeCString(cFieldValue)
	if int(C.image_get_string(in, cField, &cFieldValue)) == 0 {
		return C.GoString(cFieldValue)
	}

	return ""
}

func vipsImageGetAsString(in *C.VipsImage, name string) string {
	cField := C.CString(name)
	defer freeCString(cField)
	var cFieldValue *C.char
	defer freeCString(cFieldValue)
	if int(C.image_get_as_string(in, cField, &cFieldValue)) == 0 {
		return C.GoString(cFieldValue)
	}

	return ""
}

// Label

// Align represents VIPS_ALIGN
type Align int

// Direction enum
const (
	AlignLow    Align = C.VIPS_ALIGN_LOW
	AlignCenter Align = C.VIPS_ALIGN_CENTRE
	AlignHigh   Align = C.VIPS_ALIGN_HIGH
)

// DefaultFont is the default font to be used for label texts created by govips
const DefaultFont = "sans 10"

// LabelParams represents a text-based label
type LabelParams struct {
	Text      string
	Font      string
	Width     Scalar
	Height    Scalar
	OffsetX   Scalar
	OffsetY   Scalar
	Opacity   float32
	Color     Color
	Alignment Align
}

type vipsLabelOptions struct {
	Text      *C.char
	Font      *C.char
	Width     C.int
	Height    C.int
	OffsetX   C.int
	OffsetY   C.int
	Alignment C.VipsAlign
	DPI       C.int
	Margin    C.int
	Opacity   C.float
	Color     [3]C.double
}

func labelImage(in *C.VipsImage, params *LabelParams) (*C.VipsImage, error) {
	incOpCounter("label")
	var out *C.VipsImage

	text := C.CString(params.Text)
	defer freeCString(text)

	font := C.CString(params.Font)
	defer freeCString(font)

	// todo: release color?
	color := [3]C.double{C.double(params.Color.R), C.double(params.Color.G), C.double(params.Color.B)}

	w := params.Width.GetRounded(int(in.Xsize))
	h := params.Height.GetRounded(int(in.Ysize))
	offsetX := params.OffsetX.GetRounded(int(in.Xsize))
	offsetY := params.OffsetY.GetRounded(int(in.Ysize))

	opts := vipsLabelOptions{
		Text:      text,
		Font:      font,
		Width:     C.int(w),
		Height:    C.int(h),
		OffsetX:   C.int(offsetX),
		OffsetY:   C.int(offsetY),
		Alignment: C.VipsAlign(params.Alignment),
		Opacity:   C.float(params.Opacity),
		Color:     color,
	}

	// todo: release inline pointer?
	err := C.label(in, &out, (*C.LabelOptions)(unsafe.Pointer(&opts)))
	if err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// Resample

// Kernel represents VipsKernel type
type Kernel int

// Kernel enum
const (
	KernelAuto     Kernel = -1
	KernelNearest  Kernel = C.VIPS_KERNEL_NEAREST
	KernelLinear   Kernel = C.VIPS_KERNEL_LINEAR
	KernelCubic    Kernel = C.VIPS_KERNEL_CUBIC
	KernelLanczos2 Kernel = C.VIPS_KERNEL_LANCZOS2
	KernelLanczos3 Kernel = C.VIPS_KERNEL_LANCZOS3
	KernelMitchell Kernel = C.VIPS_KERNEL_MITCHELL
)

// Size represents VipsSize type
type Size int

const (
	SizeBoth  Size = C.VIPS_SIZE_BOTH
	SizeUp    Size = C.VIPS_SIZE_UP
	SizeDown  Size = C.VIPS_SIZE_DOWN
	SizeForce Size = C.VIPS_SIZE_FORCE
	SizeLast  Size = C.VIPS_SIZE_LAST
)

// https://libvips.github.io/libvips/API/current/libvips-resample.html#vips-resize
func vipsResizeWithVScale(in *C.VipsImage, hscale, vscale float64, kernel Kernel) (*C.VipsImage, error) {
	incOpCounter("resize")
	var out *C.VipsImage

	// libvips recommends Lanczos3 as the default kernel
	if kernel == KernelAuto {
		kernel = KernelLanczos3
	}

	if err := C.resize_image(in, &out, C.double(hscale), C.double(vscale), C.int(kernel)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

func vipsThumbnail(in *C.VipsImage, width, height int, crop Interesting, size Size) (*C.VipsImage, error) {
	incOpCounter("thumbnail")
	var out *C.VipsImage

	if err := C.thumbnail_image(in, &out, C.int(width), C.int(height), C.int(crop), C.int(size)); err != 0 {
		return nil, handleImageError(out)
	}

	return out, nil
}

// https://www.libvips.org/API/current/libvips-resample.html#vips-thumbnail
func vipsThumbnailFromFile(filename string, width, height int, crop Interesting, size Size, params *ImportParams) (*C.VipsImage, ImageType, error) {
	var out *C.VipsImage

	filenameOption := filename
	if params != nil {
		filenameOption += "[" + params.OptionString() + "]"
	}

	cFileName := C.CString(filenameOption)
	defer freeCString(cFileName)

	if err := C.thumbnail(cFileName, &out, C.int(width), C.int(height), C.int(crop), C.int(size)); err != 0 {
		err := handleImageError(out)
		if src, err2 := os.ReadFile(filename); err2 == nil {
			return vipsThumbnailFromBuffer(src, width, height, crop, size, params)
		}
		return nil, ImageTypeUnknown, err
	}

	imageType := vipsDetermineImageTypeFromMetaLoader(out)
	return out, imageType, nil
}

// https://www.libvips.org/API/current/libvips-resample.html#vips-thumbnail-buffer
func vipsThumbnailFromBuffer(buf []byte, width, height int, crop Interesting, size Size, params *ImportParams) (*C.VipsImage, ImageType, error) {
	src := buf
	// Reference src here so it's not garbage collected during image initialization.
	defer runtime.KeepAlive(src)

	var out *C.VipsImage

	var err C.int

	if params == nil {
		err = C.thumbnail_buffer(unsafe.Pointer(&src[0]), C.size_t(len(src)), &out, C.int(width), C.int(height), C.int(crop), C.int(size))
	} else {
		cOptionString := C.CString(params.OptionString())
		defer freeCString(cOptionString)

		err = C.thumbnail_buffer_with_option(unsafe.Pointer(&src[0]), C.size_t(len(src)), &out, C.int(width), C.int(height), C.int(crop), C.int(size), cOptionString)
	}
	if err != 0 {
		err := handleImageError(out)
		return nil, ImageTypeUnknown, err
	}

	imageType := vipsDetermineImageTypeFromMetaLoader(out)
	return out, imageType, nil
}
