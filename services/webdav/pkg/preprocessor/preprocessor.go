package preprocessor

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/draw"
	"image/png"
	"io"
	"math"
	"mime"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/dhowden/tag"
)

// FileConverter is the interface for the file converter
type FileConverter interface {
	Convert(r io.Reader) (*Image, error)
}

// Image is what a converter hands on: the bytes and the type they are in.
// webdav does not decode them, the generator does the image work, so an
// embedded cover reaches it the way a plain image file does.
type Image struct {
	ContentType string
	Data        []byte
}

// GgsDecoder is a converter for the geogebra slides file
type GgsDecoder struct {
	thumbnailpath string

	// maxSize bounds the extracted image, because a deflated entry expands by
	// orders of magnitude and the container's own size says nothing about it.
	maxSize uint64
}

// Convert reads the ggs file and returns the thumbnail image.
//
// TODO: this parses user-provided bytes in-process (archive/zip streams the
// embedded thumbnail out of the .ggs container). The image itself is handed on
// undecoded, but reading the container in webdav is an attack vector of its
// own; Tika is the intended out-of-process extractor for embedded images.
func (g GgsDecoder) Convert(r io.Reader) (*Image, error) {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		return nil, err
	}
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return nil, err
	}
	for _, file := range zipReader.File {
		if file.Name == g.thumbnailpath {
			if file.UncompressedSize64 > g.maxSize {
				return nil, errors.Errorf("the embedded image declares %d bytes, more than the %d allowed", file.UncompressedSize64, g.maxSize)
			}

			thumbnail, err := file.Open()
			if err != nil {
				return nil, err
			}
			// archive/zip holds the entry to its declared size while reading,
			// so the bound here is what makes that declaration ours
			data, err := io.ReadAll(io.LimitReader(thumbnail, int64(g.maxSize)+1))
			if err != nil {
				return nil, errors.Wrap(err, `could not read the embedded image`)
			}
			if uint64(len(data)) > g.maxSize {
				return nil, errors.Errorf("the embedded image is larger than the %d bytes allowed", g.maxSize)
			}

			return &Image{ContentType: "image/png", Data: data}, nil
		}
	}
	return nil, errors.Errorf("%s not found", g.thumbnailpath)
}

// AudioDecoder is a converter for the audio file
type AudioDecoder struct{}

// Convert reads the audio file and extracts the thumbnail image from the id3 tag.
//
// TODO: this parses user-provided bytes in-process (dhowden/tag reads the audio
// metadata and pulls the embedded APIC picture). The picture itself is handed
// on undecoded, but reading the metadata in webdav is an attack vector of its
// own; Tika is the intended out-of-process extractor for embedded images.
func (i AudioDecoder) Convert(r io.Reader) (*Image, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	m, err := tag.ReadFrom(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	picture := m.Picture()
	if picture == nil {
		return nil, ErrNoImageFromAudioFile
	}

	if !strings.HasPrefix(picture.MIMEType, "image/") {
		return nil, ErrNoConverterForExtractedImageFromAudioFile
	}

	return &Image{ContentType: picture.MIMEType, Data: picture.Data}, nil
}

// TxtToImageConverter is a converter for the text file
type TxtToImageConverter struct {
	fontLoader *FontLoader
}

// Convert reads the text file and renders it into a thumbnail image
func (t TxtToImageConverter) Convert(r io.Reader) (*Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, 640, 480))

	imgBounds := img.Bounds()
	draw.Draw(img, imgBounds, image.White, image.Point{}, draw.Src)

	fontSizeAsInt := int(math.Ceil(t.fontLoader.GetFaceOptSize()))
	margin := 10
	minX := fixed.I(imgBounds.Min.X + margin)
	maxX := fixed.I(imgBounds.Max.X - margin)
	maxY := fixed.I(imgBounds.Max.Y - margin)
	initialPoint := fixed.P(imgBounds.Min.X+margin, imgBounds.Min.Y+margin+fontSizeAsInt)
	canvas := &font.Drawer{
		Dst: img,
		Src: image.Black,
		Dot: initialPoint,
	}

	scriptList := t.fontLoader.GetScriptList()
	textAnalyzer := NewTextAnalyzer(scriptList)
	taOpts := AnalysisOpts{
		UseMergeMap: true,
		MergeMap:    DefaultMergeMap,
	}

	scanner := bufio.NewScanner(r)
Scan: // Label for the scanner loop, so we can break it easily
	for scanner.Scan() {
		txt := scanner.Text()
		height := fixed.I(fontSizeAsInt) // reset to default height

		textResult := textAnalyzer.AnalyzeString(txt, taOpts)
		textResult.MergeCommon(DefaultMergeMap)

		for _, sRange := range textResult.ScriptRanges {
			targetFontFace, err := t.fontLoader.LoadFaceForScript(sRange.TargetScript)
			if err != nil {
				return nil, err
			}

			// if the target script is "_unknown" it's expected that the loaded face
			// uses the default font
			faceHeight := targetFontFace.Face.Metrics().Height
			if faceHeight > height {
				height = faceHeight
			}

			canvas.Face = targetFontFace.Face
			initialByte := sRange.Low
			for _, sRangeSpace := range sRange.Spaces {
				if canvas.Dot.Y > maxY {
					break Scan
				}

				drawWord(canvas, textResult.Text[initialByte:sRangeSpace], minX, maxX, height, maxY)
				initialByte = sRangeSpace
			}

			if initialByte <= sRange.High {
				// some bytes left to be written
				if canvas.Dot.Y > maxY {
					break Scan
				}

				drawWord(canvas, textResult.Text[initialByte:sRange.High+1], minX, maxX, height, maxY)
			}
		}

		canvas.Dot.X = minX
		canvas.Dot.Y += height.Mul(fixed.Int26_6(1<<6 + 1<<5)) // height * 1.5

		if canvas.Dot.Y > maxY {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// png, not jpeg: the render goes to the generator untouched and is
	// compressed once, in the size the client asked for
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, errors.Wrap(err, `could not encode the rendered text`)
	}

	return &Image{ContentType: "image/png", Data: buf.Bytes()}, nil
}

// GGPStruct is the layout of a ggp file (which is basically json)
type GGPStruct struct {
	Sections []struct {
		Cards []struct {
			Element struct {
				Image struct {
					Base64Image string
				}
			}
		}
	}
}

// GgpDecoder is a converter for the geogebra pinboard file
type GgpDecoder struct{}

// Convert reads the ggp file and returns the first thumbnail image.
//
// TODO: this parses user-provided bytes in-process (JSON/base64 extraction of
// the embedded image). The image itself is handed on undecoded, but reading the
// document in webdav is an attack vector of its own; Tika is the intended
// out-of-process extractor for embedded images.
func (j GgpDecoder) Convert(r io.Reader) (*Image, error) {
	ggp := &GGPStruct{}
	err := json.NewDecoder(r).Decode(ggp)
	if err != nil {
		return nil, err
	}

	elem, err := extractBase64ImageFromGGP(ggp)
	if err != nil {
		return nil, err
	}

	b, err := base64.StdEncoding.DecodeString(elem)
	if err != nil {
		return nil, err
	}

	contentType := http.DetectContentType(b)
	if !strings.HasPrefix(contentType, "image/") {
		return nil, ErrNoConverterForExtractedImageFromGgpFile
	}

	return &Image{ContentType: contentType, Data: b}, nil
}

func extractBase64ImageFromGGP(ggp *GGPStruct) (string, error) {
	if len(ggp.Sections) < 1 || len(ggp.Sections[0].Cards) < 1 {
		return "", errors.New("cant find thumbnail in ggp file")
	}

	raw := strings.Split(ggp.Sections[0].Cards[0].Element.Image.Base64Image, "base64,")
	if len(raw) < 2 {
		return "", errors.New("cant decode ggp thumbnail")
	}

	return raw[1], nil
}

// Draw the word in the canvas. The mixX and maxX defines the drawable range
// (X axis) where the word can be drawn (in case the word is too big and doesn't
// fit in the canvas), and the incY defines the increment in the Y axis if we
// need to draw the word in a new line
//
// Note that the word will likely start with a white space char
func drawWord(canvas *font.Drawer, word string, minX, maxX, incY, maxY fixed.Int26_6) {
	// calculate the actual measurement of the string at a given X position
	measure := func(s string, dotX fixed.Int26_6) (min, max fixed.Int26_6) {
		bbox, _ := canvas.BoundString(s)
		return dotX + bbox.Min.X, dotX + bbox.Max.X
	}

	// first try to draw the whole word
	absMin, absMax := measure(word, canvas.Dot.X)
	if absMin >= minX && absMax <= maxX {
		canvas.DrawString(word)
		return
	}

	// try to draw the trimmed word in a new line
	trimmed := strings.TrimSpace(word)
	oldDot := canvas.Dot
	canvas.Dot.X = minX
	canvas.Dot.Y += incY

	if canvas.Dot.Y <= maxY {
		tMin, tMax := measure(trimmed, canvas.Dot.X)
		if tMin >= minX && tMax <= maxX {
			canvas.DrawString(trimmed)
			return
		}
	}

	// if the trimmed word is still too big, draw it char by char
	canvas.Dot = oldDot
	for _, char := range trimmed {
		s := string(char)
		_, cMax := measure(s, canvas.Dot.X)

		if cMax > maxX {
			canvas.Dot.X = minX
			canvas.Dot.Y += incY
		}

		// stop drawing if we exceed maxY
		if canvas.Dot.Y > maxY {
			return
		}

		// ensure that we don't start drawing before minX
		cMin, _ := measure(s, canvas.Dot.X)
		if cMin < minX {
			canvas.Dot.X += minX - cMin
		}

		canvas.DrawString(s)
	}
}

// defaultMaxExtractedSize bounds an image extracted from a container when the
// caller passes no limit of its own.
const defaultMaxExtractedSize = 50 << 20

// maxExtractedSize is the caller's configured maximum input size, which bounds
// what an extractor may produce as well. An unlimited input (0) still bounds
// the extracted image, because the point of the limit here is the expansion
// factor, not the size of the file the user uploaded.
func maxExtractedSize(opts Options) uint64 {
	if opts.MaxImageSize > 0 {
		return opts.MaxImageSize
	}

	return defaultMaxExtractedSize
}

// Options carries what a converter needs from the caller. The zero value asks
// for the defaults.
type Options struct {
	FontMapFile  string
	FontFace     *opentype.FaceOptions
	MaxImageSize uint64
}

// ForType returns the converter for the specified mimeType
func ForType(mimeType string, opts Options) FileConverter {
	// We can ignore the error here because we parse it in IsMimeTypeSupported before and if it fails
	// return the service call. So we should only get here when the mimeType parses fine.
	mimeType, _, _ = mime.ParseMediaType(mimeType)
	switch mimeType {
	case "text/plain":
		fontFace := opts.FontFace
		if fontFace == nil {
			fontFace = &opentype.FaceOptions{
				Size:    12,
				DPI:     72,
				Hinting: font.HintingNone,
			}
		}

		fontLoader, err := NewFontLoader(opts.FontMapFile, fontFace)
		if err != nil {
			// if it couldn't create the FontLoader with the specified fontFileMap,
			// try to use the default font
			fontLoader, _ = NewFontLoader("", fontFace)
		}
		return TxtToImageConverter{
			fontLoader: fontLoader,
		}
	case "application/vnd.geogebra.slides":
		return GgsDecoder{thumbnailpath: "_slide0/geogebra_thumbnail.png", maxSize: maxExtractedSize(opts)}
	case "application/vnd.geogebra.pinboard":
		return GgpDecoder{}
	case "audio/flac":
		fallthrough
	case "audio/mpeg":
		fallthrough
	case "audio/ogg":
		return AudioDecoder{}
	default:
		// webdav decodes no image: an image/* source is streamed to the
		// generator untouched, so a type without a converter here is one
		// nobody can make a preview of.
		return nil
	}
}
