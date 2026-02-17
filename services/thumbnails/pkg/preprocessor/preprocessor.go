package preprocessor

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/draw"
	"image/gif"
	"io"
	"math"
	"mime"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/dhowden/tag"

	thumbnailerErrors "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/errors"
)

// FileConverter is the interface for the file converter
type FileConverter interface {
	Convert(r io.Reader) (interface{}, error)
}

// GifDecoder is a converter for the gif file
type GifDecoder struct{}

// Convert reads the gif file and returns the thumbnail image
func (i GifDecoder) Convert(r io.Reader) (interface{}, error) {
	img, err := gif.DecodeAll(r)
	if err != nil {
		return nil, errors.Wrap(err, `could not decode the image`)
	}
	return img, nil
}

// GgsDecoder is a converter for the geogebra slides file
type GgsDecoder struct{ thumbnailpath string }

// Convert reads the ggs file and returns the thumbnail image
func (g GgsDecoder) Convert(r io.Reader) (interface{}, error) {
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
			thumbnail, err := file.Open()
			if err != nil {
				return nil, err
			}
			converter := ForType("image/png", nil)
			if converter == nil {
				return nil, thumbnailerErrors.ErrNoConverterForExtractedImageFromGgsFile
			}
			img, err := converter.Convert(thumbnail)
			if err != nil {
				return nil, errors.Wrap(err, `could not decode the image`)
			}
			return img, nil
		}
	}
	return nil, errors.Errorf("%s not found", g.thumbnailpath)
}

// AudioDecoder is a converter for the audio file
type AudioDecoder struct{}

// Convert reads the audio file and extracts the thumbnail image from the id3 tag
func (i AudioDecoder) Convert(r io.Reader) (interface{}, error) {
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
		return nil, thumbnailerErrors.ErrNoImageFromAudioFile
	}

	converter := ForType(picture.MIMEType, nil)
	if converter == nil {
		return nil, thumbnailerErrors.ErrNoConverterForExtractedImageFromAudioFile
	}

	return converter.Convert(bytes.NewReader(picture.Data))
}

// TxtToImageConverter is a converter for the text file
type TxtToImageConverter struct {
	fontLoader *FontLoader
}

// Convert reads the text file and renders it into a thumbnail image
func (t TxtToImageConverter) Convert(r io.Reader) (interface{}, error) {
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
	return img, scanner.Err()
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

// Convert reads the ggp file and returns the first thumbnail image
func (j GgpDecoder) Convert(r io.Reader) (interface{}, error) {
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

	img, _, err := image.Decode(bytes.NewReader(b))
	return img, err
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

// ForType returns the converter for the specified mimeType
func ForType(mimeType string, opts map[string]interface{}) FileConverter {
	// We can ignore the error here because we parse it in IsMimeTypeSupported before and if it fails
	// return the service call. So we should only get here when the mimeType parses fine.
	mimeType, _, _ = mime.ParseMediaType(mimeType)
	switch mimeType {
	case "text/plain":
		fontFileMap := ""
		fontFaceOpts := &opentype.FaceOptions{
			Size:    12,
			DPI:     72,
			Hinting: font.HintingNone,
		}

		if optedFontFileMap, ok := opts["fontFileMap"]; ok {
			if stringFontFileMap, ok := optedFontFileMap.(string); ok {
				fontFileMap = stringFontFileMap
			}
		}

		if optedFontFaceOpts, ok := opts["fontFaceOpts"]; ok {
			if typedFontFaceOpts, ok := optedFontFaceOpts.(*opentype.FaceOptions); ok {
				fontFaceOpts = typedFontFaceOpts
			}
		}

		fontLoader, err := NewFontLoader(fontFileMap, fontFaceOpts)
		if err != nil {
			// if it couldn't create the FontLoader with the specified fontFileMap,
			// try to use the default font
			fontLoader, _ = NewFontLoader("", fontFaceOpts)
		}
		return TxtToImageConverter{
			fontLoader: fontLoader,
		}
	case "application/vnd.geogebra.slides":
		return GgsDecoder{"_slide0/geogebra_thumbnail.png"}
	case "application/vnd.geogebra.pinboard":
		return GgpDecoder{}
	case "image/gif":
		return GifDecoder{}
	case "audio/flac":
		fallthrough
	case "audio/mpeg":
		fallthrough
	case "audio/ogg":
		return AudioDecoder{}
	default:
		return ImageDecoder{}
	}
}
