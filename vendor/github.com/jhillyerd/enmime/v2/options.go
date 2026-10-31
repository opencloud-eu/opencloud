package enmime

// Option to configure parsing.
type Option interface {
	apply(p *Parser)
}

// SkipMalformedParts sets parsing to skip parts that's can't be parsed.
func SkipMalformedParts(s bool) Option {
	return skipMalformedPartsOption(s)
}

type skipMalformedPartsOption bool

func (o skipMalformedPartsOption) apply(p *Parser) {
	p.skipMalformedParts = bool(o)
}

// MultipartWOBoundaryAsSinglePart if set to true will treat a multi-part messages without boundary parameter as single-part.
// Otherwise, will return error that boundary is not found.
func MultipartWOBoundaryAsSinglePart(a bool) Option {
	return multipartWOBoundaryAsSinglePartOption(a)
}

type multipartWOBoundaryAsSinglePartOption bool

func (o multipartWOBoundaryAsSinglePartOption) apply(p *Parser) {
	p.multipartWOBoundaryAsSinglePart = bool(o)
}

// SetReadPartErrorPolicy sets the given callback function to readPartErrorPolicy.
func SetReadPartErrorPolicy(f ReadPartErrorPolicy) Option {
	return readPartErrorPolicyOption(f)
}

type readPartErrorPolicyOption ReadPartErrorPolicy

func (o readPartErrorPolicyOption) apply(p *Parser) {
	p.readPartErrorPolicy = ReadPartErrorPolicy(o)
}

// MaxStoredPartErrors limits number of part parsing errors, errors beyond the limit are discarded.
// Zero, the default, means all errors will be kept.
func MaxStoredPartErrors(n int) Option {
	return maxStoredPartErrorsOption(n)
}

type maxStoredPartErrorsOption int

func (o maxStoredPartErrorsOption) apply(p *Parser) {
	p.maxStoredPartErrors = int(o)
}

// RawContent if set to true will not try to decode the CTE and return the raw part content.
// Otherwise, will try to automatically decode the CTE.
func RawContent(a bool) Option {
	return rawContentOption(a)
}

type rawContentOption bool

func (o rawContentOption) apply(p *Parser) {
	p.rawContent = bool(o)
}

// SetCustomParseMediaType if provided, will be used to parse media type instead of the default ParseMediaType
// function.  This may be used to parse media type parameters that would otherwise be considered malformed.
// By default parsing happens using ParseMediaType
func SetCustomParseMediaType(customParseMediaType CustomParseMediaType) Option {
	return parseMediaTypeOption(customParseMediaType)
}

type parseMediaTypeOption CustomParseMediaType

func (o parseMediaTypeOption) apply(p *Parser) {
	p.customParseMediaType = CustomParseMediaType(o)
}

type stripMediaTypeInvalidCharactersOption bool

func (o stripMediaTypeInvalidCharactersOption) apply(p *Parser) {
	p.stripMediaTypeInvalidCharacters = bool(o)
}

// StripMediaTypeInvalidCharacters sets stripMediaTypeInvalidCharacters option. If true, invalid characters
// will be removed from media type during parsing.
func StripMediaTypeInvalidCharacters(stripMediaTypeInvalidCharacters bool) Option {
	return stripMediaTypeInvalidCharactersOption(stripMediaTypeInvalidCharacters)
}

type disableTextConversionOption bool

func (o disableTextConversionOption) apply(p *Parser) {
	p.disableTextConversion = bool(o)
}

// DisableTextConversion sets the disableTextConversion option. When true, there will be no
// automated down conversion of HTML to text when a plain/text body is missing.
func DisableTextConversion(disableTextConversion bool) Option {
	return disableTextConversionOption(disableTextConversion)
}

type disableCharacterDetectionOption bool

func (o disableCharacterDetectionOption) apply(p *Parser) {
	p.disableCharacterDetection = bool(o)
}

// DisableCharacterDetection sets the disableCharacterDetection option. When true, the parser will use the
// defined character set if it is defined in the message part.
func DisableCharacterDetection(disableCharacterDetection bool) Option {
	return disableCharacterDetectionOption(disableCharacterDetection)
}

type minCharsetDetectRunesOption int

func (o minCharsetDetectRunesOption) apply(p *Parser) {
	p.minCharsetDetectRunes = int(o)
}

// MinCharsetDetectRunes sets the minimum length of a MIME part before enmime will attempt to
// detect its character set.  The shorter the text, the more likely an incorrect character set
// will be chosen.  The default is 100.
func MinCharsetDetectRunes(minCharsetDetectRunes int) Option {
	return minCharsetDetectRunesOption(minCharsetDetectRunes)
}
