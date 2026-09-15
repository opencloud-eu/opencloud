package preprocessor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/sync"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// FontMap maps a script with the target font to be used for that script
// It also uses a DefaultFont in case there isn't a matching script in the map
//
// For cases like Japanese where multiple scripts are used, we rely on the text
// analyzer to use the script which is unique to japanese (Hiragana or Katakana)
// even if it has to overwrite the "official" detected script (Han). This means
// that "Han" should be used just for chinese while "Hiragana" and "Katakana"
// should be used for japanese
type FontMap struct {
	FontMap     map[string]string `json:"fontMap"`
	DefaultFont string            `json:"defaultFont"`
}

// FontMapData contains the location of the loaded file (in FLoc) and the FontMap loaded
// from the file
type FontMapData struct {
	FMap *FontMap
	FLoc string
}

// LoadedFace contains the location of the font used, and the loaded face (font.Face)
// ready to be used
type LoadedFace struct {
	FontFile string
	Face     font.Face
}

// FontLoader represents a FontLoader. Use the "NewFontLoader" to get a instance
type FontLoader struct {
	faceCache   sync.Cache
	fontMapData *FontMapData
	faceOpts    *opentype.FaceOptions
}

// NewFontLoader creates a new FontLoader based on the fontMapFile. The FaceOptions will
// be the same for all the font loaded by this instance.
// Note that only the fonts described in the fontMapFile will be used.
//
// The fontMapFile has the following structure
//
//	{
//		"fontMap": {
//			"Han": "packaged/myFont-CJK.otf",
//			"Arabic": "packaged/myFont-Arab.otf",
//			"Latin": "/fonts/regular/myFont.otf"
//		}
//		"defaultFont": "/fonts/regular/myFont.otf"
//	}
//
// The fontMapFile contains paths to where the fonts are located in the FS.
// Absolute paths can be used as shown above. If a relative path is used,
// it will be relative to the fontMapFile location. This should make the
// packaging easier since all the fonts can be placed in the same directory
// where the fontMapFile is, or in inner directories.
func NewFontLoader(fontMapFile string, faceOpts *opentype.FaceOptions) (*FontLoader, error) {
	fontMap := &FontMap{}

	if fontMapFile != "" {
		file, err := os.Open(fontMapFile)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		parser := json.NewDecoder(file)
		if err = parser.Decode(fontMap); err != nil {
			return nil, err
		}
	}

	return &FontLoader{
		faceCache: sync.NewCache(5),
		fontMapData: &FontMapData{
			FMap: fontMap,
			FLoc: fontMapFile,
		},
		faceOpts: faceOpts,
	}, nil
}

// LoadFaceForScript loads and returns the font face to be used for that script according to the
// FontMap set when the FontLoader was created. If the script doesn't have
// an associated font, a default font will be used. Note that the default font
// might not be able to handle properly the script
func (fl *FontLoader) LoadFaceForScript(script string) (*LoadedFace, error) {
	var parsedFont *opentype.Font
	var parsingError error

	fontFile := fl.fontMapData.FMap.DefaultFont
	if val, ok := fl.fontMapData.FMap.FontMap[script]; ok {
		fontFile = val
	}

	if fontFile != "" && !filepath.IsAbs(fontFile) {
		fontFile = filepath.Join(filepath.Dir(fl.fontMapData.FLoc), fontFile)
	}

	// if the face for the script isn't cached, load the font file and create a new face
	cachedFace := fl.faceCache.Load(fontFile)
	if cachedFace != nil {
		return cachedFace.V.(*LoadedFace), nil
	}

	if fontFile == "" {
		parsedFont, parsingError = opentype.Parse(goregular.TTF)
		if parsingError != nil {
			return nil, parsingError
		}
	} else {
		// opentype.ParseReaderAt seems to require to keep the file opened
		// so read the font file into memory
		data, err := os.ReadFile(fontFile)
		if err != nil {
			return nil, err
		}
		parsedFont, parsingError = opentype.Parse(data)
		if parsingError != nil {
			return nil, parsingError
		}
	}

	face, err := opentype.NewFace(parsedFont, fl.faceOpts)
	if err != nil {
		return nil, err
	}

	loadedFace := &LoadedFace{
		FontFile: fontFile,
		Face:     face,
	}
	fl.faceCache.Store(fontFile, loadedFace, time.Now().Add(10*time.Minute))
	return loadedFace, nil
}

// GetFaceOptSize returns face opt size
func (fl *FontLoader) GetFaceOptSize() float64 {
	return fl.faceOpts.Size
}

// GetFaceOptDPI returns face opt DPI
func (fl *FontLoader) GetFaceOptDPI() float64 {
	return fl.faceOpts.DPI
}

// GetScriptList returns script list
func (fl *FontLoader) GetScriptList() []string {
	fontMap := fl.fontMapData.FMap.FontMap

	arePresent := map[string]bool{
		"Common":    false,
		"Inherited": false,
	}
	listSize := len(fontMap)

	for key := range arePresent {
		if _, inFontMap := fontMap[key]; inFontMap {
			arePresent[key] = true
		} else {
			listSize++
		}
	}

	keys := make([]string, listSize)

	i := 0
	for k := range fontMap {
		keys[i] = k
		i++
	}

	for script, isPresent := range arePresent {
		if !isPresent {
			keys[i] = script
			i++
		}
	}
	return keys
}
