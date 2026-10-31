package enmime

// ReadPartErrorPolicy allows to recover the buffer (or not) on an error when reading a Part content.
//
// See AllowCorruptTextPartErrorPolicy for usage.
type ReadPartErrorPolicy func(*Part, error) bool

// AllowCorruptTextPartErrorPolicy recovers partial content from base64.CorruptInputError when content type is text/plain or text/html.
func AllowCorruptTextPartErrorPolicy(p *Part, err error) bool {
	if IsBase64CorruptInputError(err) && (p.ContentType == ctTextHTML || p.ContentType == ctTextPlain) {
		return true
	}
	return false
}

// CustomParseMediaType parses media type. See ParseMediaType for more details
type CustomParseMediaType func(ctype string) (mtype string, params map[string]string, invalidParams []string, err error)

// Parser parses MIME.  Create with NewParser to inherit recommended defaults.
type Parser struct {
	maxStoredPartErrors             int
	multipartWOBoundaryAsSinglePart bool
	readPartErrorPolicy             ReadPartErrorPolicy
	skipMalformedParts              bool
	rawContent                      bool
	customParseMediaType            CustomParseMediaType
	stripMediaTypeInvalidCharacters bool
	disableTextConversion           bool
	disableCharacterDetection       bool
	minCharsetDetectRunes           int
}

// defaultParser is a Parser with default configuration.
var defaultParser = *NewParser()

// NewParser creates new parser with given options.
func NewParser(ops ...Option) *Parser {
	// Construct parser with default options.
	p := Parser{
		minCharsetDetectRunes: 100,
	}

	for _, o := range ops {
		o.apply(&p)
	}

	return &p
}
