package enmime

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"io"
	"math/rand"
	"mime/quotedprintable"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/gogs/chardet"
	"github.com/jhillyerd/enmime/v2/internal/coding"
	inttp "github.com/jhillyerd/enmime/v2/internal/textproto"
	"github.com/jhillyerd/enmime/v2/mediatype"
	"github.com/pkg/errors"
)

const (
	minCharsetConfidence = 85
)

// Part represents a node in the MIME multipart tree.  The Content-Type, Disposition and File Name
// are parsed out of the header for easier access.
type Part struct {
	PartID      string               // PartID labels this part's position within the tree.
	Parent      *Part                // Parent of this part (can be nil.)
	FirstChild  *Part                // FirstChild is the top most child of this part.
	NextSibling *Part                // NextSibling of this part.
	Header      textproto.MIMEHeader // Header for this part.

	Boundary          string            // Boundary marker used within this part.
	ContentID         string            // ContentID header for cid URL scheme.
	ContentType       string            // ContentType header without parameters.
	ContentTypeParams map[string]string // Params, added to ContentType header.
	Disposition       string            // Content-Disposition header without parameters.
	FileName          string            // The file-name from disposition or type header.
	FileModDate       time.Time         // The modification date of the file.
	Charset           string            // The content charset encoding, may differ from charset in header.
	OrigCharset       string            // The original content charset when a different charset was detected.

	Errors        []*Error  // Errors encountered while parsing this part.
	Content       []byte    // Content after decoding, UTF-8 conversion if applicable.
	ContentReader io.Reader // Reader interface for pulling the content for encoding.
	Epilogue      []byte    // Epilogue contains data following the closing boundary marker.

	parser *Parser // Provides access to parsing options.

	randSource rand.Source // optional rand for uuid boundary generation

	encoder *Encoder // provides encoding options
}

// NewPart creates a new Part object.
func NewPart(contentType string) *Part {
	return &Part{
		Header:            make(textproto.MIMEHeader),
		ContentType:       contentType,
		ContentTypeParams: make(map[string]string),
		parser:            &defaultParser,
	}
}

// AddChild adds a child part to either FirstChild or the end of the children NextSibling chain.
// The child may have siblings and children attached.  This method will set the Parent field on
// child and all its siblings. Safe to call on nil.
func (p *Part) AddChild(child *Part) {
	if p == child {
		// Prevent paradox.
		return
	}
	if p != nil {
		if p.FirstChild == nil {
			// Make it the first child.
			p.FirstChild = child
		} else {
			// Append to sibling chain.
			current := p.FirstChild
			for current.NextSibling != nil {
				current = current.NextSibling
			}
			if current == child {
				// Prevent infinite loop.
				return
			}
			current.NextSibling = child
		}
	}
	// Update all new first-level children Parent pointers.
	for c := child; c != nil; c = c.NextSibling {
		if c == c.NextSibling {
			// Prevent infinite loop.
			return
		}
		c.Parent = p
	}
}

// TextContent indicates whether the content is text based on its content type.  This value
// determines what content transfer encoding scheme to use.
func (p *Part) TextContent() bool {
	if p.ContentType == "" {
		// RFC 2045: no CT is equivalent to "text/plain; charset=us-ascii"
		return true
	}
	return strings.HasPrefix(p.ContentType, "text/") ||
		strings.HasPrefix(p.ContentType, ctMultipartPrefix)
}

// setupHeaders reads the header, then populates the MIME header values for this Part.
func (p *Part) setupHeaders(r *bufio.Reader, defaultContentType string) error {
	header, err := readHeader(r, p)
	if err != nil {
		return err
	}
	p.Header = textproto.MIMEHeader(header)
	ctype := header.Get(hnContentType)
	if ctype == "" {
		if defaultContentType == "" {
			p.addWarning(ErrorMissingContentType, "MIME parts should have a Content-Type header")
			return nil
		}
		ctype = defaultContentType
	}
	// Parse Content-Type header.
	mtype, mparams, minvalidParams, err := p.parseMediaType(ctype)
	if err != nil {
		return err
	}
	for i := range minvalidParams {
		p.addWarningf(
			ErrorMalformedHeader,
			"Content-Type header has malformed parameter %q",
			minvalidParams[i])
	}
	p.ContentType = mtype
	// Set disposition, filename, charset if available.
	p.setupContentHeaders(mparams)
	p.Boundary = mparams[hpBoundary]
	p.ContentID = coding.FromIDHeader(header.Get(hnContentID))
	return nil
}

// setupContentHeaders uses Content-Type media params and Content-Disposition headers to populate
// the disposition, filename, and charset fields.
func (p *Part) setupContentHeaders(mediaParams map[string]string) {
	header := inttp.MIMEHeader(p.Header)
	// Determine content disposition, filename, character set.
	disposition, dparams, _, err := p.parseMediaType(header.Get(hnContentDisposition))
	if err == nil {
		// Disposition is optional
		p.Disposition = disposition
		p.FileName = coding.DecodeExtHeader(dparams[hpFilename])
	}
	if p.FileName == "" && mediaParams[hpName] != "" {
		p.FileName = coding.DecodeExtHeader(mediaParams[hpName])
	}
	if p.FileName == "" && mediaParams[hpFile] != "" {
		p.FileName = coding.DecodeExtHeader(mediaParams[hpFile])
	}
	if p.Charset == "" {
		p.Charset = mediaParams[hpCharset]
	}
	if p.FileModDate.IsZero() {
		p.FileModDate, _ = time.Parse(time.RFC822, mediaParams[hpModDate])
	}
}

func (p *Part) readPartContent(r io.Reader, readPartErrorPolicy ReadPartErrorPolicy) ([]byte, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		if readPartErrorPolicy != nil && readPartErrorPolicy(p, err) {
			p.addWarningf(ErrorMalformedChildPart, "partial content: %s", err.Error())
			return buf, nil
		}
		return nil, err
	}
	return buf, nil
}

// convertFromDetectedCharset attempts to detect the character set for the given part, and returns
// an io.Reader that will convert from that charset to UTF-8. If the charset cannot be detected,
// this method adds a warning to the part and automatically falls back to using
// `convertFromStatedCharset` and returns the reader from that method.
func (p *Part) convertFromDetectedCharset(r io.Reader, readPartErrorPolicy ReadPartErrorPolicy) (io.Reader, error) {
	// Attempt to detect character set from part content.
	var cd *chardet.Detector
	switch p.ContentType {
	case "text/html":
		cd = chardet.NewHtmlDetector()
	default:
		cd = chardet.NewTextDetector()
	}

	buf, err := p.readPartContent(r, readPartErrorPolicy)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Restore r.
	r = bytes.NewReader(buf)

	if p.parser.disableCharacterDetection && p.Charset != "" {
		// Charset detection is disabled, use declared charset.
		return p.convertFromStatedCharset(r), nil
	}

	cs, err := cd.DetectBest(buf)
	switch err {
	case nil:
		// Carry on
	default:
		p.addWarningf(ErrorCharsetDeclaration, "charset could not be detected: %v", err)
	}

	if cs == nil || cs.Confidence < minCharsetConfidence || len(bytes.Runes(buf)) < p.parser.minCharsetDetectRunes {
		// Low confidence or not enough characters, use declared character set.
		return p.convertFromStatedCharset(r), nil
	}

	// Confidence exceeded our threshold, use detected character set.
	if p.Charset != "" && !strings.EqualFold(cs.Charset, p.Charset) {
		p.addWarningf(ErrorCharsetDeclaration,
			"declared charset %q, detected %q, confidence %d",
			p.Charset, cs.Charset, cs.Confidence)
	}

	if reader, err := coding.NewCharsetReader(cs.Charset, r); err == nil {
		r = reader
		p.OrigCharset = p.Charset
		p.Charset = cs.Charset
	}

	return r, nil
}

// convertFromStatedCharset returns a reader that will convert from the charset specified for the
// current `*Part` to UTF-8. In case of error, or an unhandled character set, a warning will be
// added to the `*Part` and the original io.Reader will be returned.
func (p *Part) convertFromStatedCharset(r io.Reader) io.Reader {
	if p.Charset == "" {
		// US-ASCII. Just read.
		return r
	}

	reader, err := coding.NewCharsetReader(p.Charset, r)
	if err != nil {
		// Failed to get a conversion reader.
		p.addWarningf(ErrorCharsetConversion, "failed to get reader for charset %q: %v", p.Charset, err)
	} else {
		return reader
	}

	// Try to parse charset again here to see if we can salvage some badly formed
	// ones like charset="charset=utf-8".
	charsetp := strings.Split(p.Charset, "=")
	if strings.EqualFold(charsetp[0], "charset") && len(charsetp) > 1 ||
		strings.EqualFold(charsetp[0], "iso") && len(charsetp) > 1 {
		p.Charset = charsetp[1]
		reader, err = coding.NewCharsetReader(p.Charset, r)
		if err != nil {
			// Failed to get a conversion reader.
			p.addWarningf(ErrorCharsetConversion, "failed to get reader for charset %q: %v", p.Charset, err)
		} else {
			return reader
		}
	}

	return r
}

// decodeContent performs transport decoding (base64, quoted-printable) and charset decoding,
// placing the result into Part.Content.  IO errors will be returned immediately; other errors
// and warnings will be added to Part.Errors.
func (p *Part) decodeContent(r io.Reader, readPartErrorPolicy ReadPartErrorPolicy) error {
	header := inttp.MIMEHeader(p.Header)
	// contentReader will point to the end of the content decoding pipeline.
	contentReader := r
	// b64cleaner aggregates errors, must maintain a reference to it to get them later.
	var b64cleaner *coding.Base64Cleaner
	// Build content decoding reader.
	encoding := ""
	if p.parser != nil && !p.parser.rawContent {
		encoding = header.Get(hnContentEncoding)
	}
	validEncoding := true
	switch strings.ToLower(encoding) {
	case cteQuotedPrintable:
		contentReader = coding.NewQPCleaner(contentReader)
		contentReader = quotedprintable.NewReader(contentReader)
	case cteBase64:
		b64cleaner = coding.NewBase64Cleaner(contentReader)
		contentReader = base64.NewDecoder(base64.RawStdEncoding, b64cleaner)
	case cte8Bit, cte7Bit, cteBinary, "":
		// No decoding required.
	default:
		// Unknown encoding.
		validEncoding = false
		p.addWarningf(
			ErrorContentEncoding,
			"Unrecognized Content-Transfer-Encoding type %q",
			encoding)
	}
	// Build charset decoding reader.
	if validEncoding && strings.HasPrefix(p.ContentType, "text/") && !p.parser.rawContent {
		var err error
		contentReader, err = p.convertFromDetectedCharset(contentReader, readPartErrorPolicy)
		if err != nil {
			return p.base64CorruptInputCheck(err)
		}
	}
	// Decode and store content.
	content, err := p.readPartContent(contentReader, readPartErrorPolicy)
	if err != nil {
		return p.base64CorruptInputCheck(errors.WithStack(err))
	}
	p.Content = content
	// Collect base64 errors.
	if b64cleaner != nil {
		for _, err := range b64cleaner.Errors {
			p.addWarning(ErrorMalformedBase64, err.Error())
		}
	}
	// Set empty content-type error.
	if p.ContentType == "" {
		p.addWarningf(
			ErrorMissingContentType, "content-type is empty for part id: %s", p.PartID)
	}
	return nil
}

// parses media type using custom or default media type parser
func (p *Part) parseMediaType(ctype string) (mtype string, params map[string]string, invalidParams []string, err error) {
	if p.parser == nil || p.parser.customParseMediaType == nil {
		return mediatype.ParseWithOptions(ctype, mediatype.ParseOptions{StripMediaTypeInvalidCharacters: p.parser.stripMediaTypeInvalidCharacters})
	}

	return p.parser.customParseMediaType(ctype)
}

// IsBase64CorruptInputError returns true when err is of type base64.CorruptInputError.
//
// It can be used to create ReadPartErrorPolicy functions.
func IsBase64CorruptInputError(err error) bool {
	switch errors.Cause(err).(type) {
	case base64.CorruptInputError:
		return true
	default:
		return false
	}
}

// base64CorruptInputCheck will avoid fatal failure on corrupt base64 input
//
// This is a switch on errors.Cause(err).(type) for base64.CorruptInputError
func (p *Part) base64CorruptInputCheck(err error) error {
	if IsBase64CorruptInputError(err) {
		p.Content = nil
		p.addError(ErrorMalformedBase64, err.Error())
		return nil
	}
	return err
}

// Clone returns a clone of the current Part.
func (p *Part) Clone(parent *Part) *Part {
	if p == nil {
		return nil
	}

	newPart := &Part{
		PartID:      p.PartID,
		Header:      p.Header,
		Parent:      parent,
		Boundary:    p.Boundary,
		ContentID:   p.ContentID,
		ContentType: p.ContentType,
		Disposition: p.Disposition,
		FileName:    p.FileName,
		Charset:     p.Charset,
		Errors:      p.Errors,
		Content:     p.Content,
		Epilogue:    p.Epilogue,
	}
	newPart.FirstChild = p.FirstChild.Clone(newPart)
	newPart.NextSibling = p.NextSibling.Clone(parent)

	return newPart
}

// ReadParts reads a MIME document from the provided reader and parses it into tree of Part objects.
func ReadParts(r io.Reader) (*Part, error) {
	return defaultParser.ReadParts(r)
}

// ReadParts reads a MIME document from the provided reader and parses it into tree of Part objects.
func (p Parser) ReadParts(r io.Reader) (*Part, error) {
	br := bufio.NewReader(r)
	root := &Part{PartID: "0", parser: &p}

	// Read header; top-level default CT is text/plain us-ascii according to RFC 822.
	if err := root.setupHeaders(br, `text/plain; charset="us-ascii"`); err != nil {
		return nil, err
	}

	if detectMultipartMessage(root, p.multipartWOBoundaryAsSinglePart) {
		// Content is multipart, parse it.
		if err := parseParts(root, br); err != nil {
			return nil, err
		}
	} else {
		// Content is text or data, decode it.
		if err := root.decodeContent(br, p.readPartErrorPolicy); err != nil {
			return nil, err
		}
	}
	return root, nil
}

// parseParts recursively parses a MIME multipart document and sets each Parts PartID.
func parseParts(parent *Part, reader *bufio.Reader) error {
	firstRecursion := parent.Parent == nil
	// Loop over MIME boundaries.
	br := newBoundaryReader(reader, parent.Boundary)
	for indexPartID := 1; true; indexPartID++ {
		next, err := br.Next()
		if err != nil && errors.Cause(err) != io.EOF {
			return err
		}
		if br.unbounded {
			parent.addWarningf(ErrorMissingBoundary, "Boundary %q was not closed correctly",
				parent.Boundary)
		}
		if !next {
			break
		}

		// Set this Part's PartID, indicating its position within the MIME Part tree.
		p := &Part{parser: parent.parser}
		if firstRecursion {
			p.PartID = strconv.Itoa(indexPartID)
		} else {
			p.PartID = parent.PartID + "." + strconv.Itoa(indexPartID)
		}

		// Look for part header.
		bbr := bufio.NewReader(br)
		if err = p.setupHeaders(bbr, ""); err != nil {
			if p.parser.skipMalformedParts {
				parent.addErrorf(ErrorMalformedChildPart, "read header: %s", err.Error())
				continue
			}

			return err
		}

		// Insert this Part into the MIME tree.
		if p.Boundary == "" {
			// Content is text or data, decode it.
			if err = p.decodeContent(bbr, p.parser.readPartErrorPolicy); err != nil {
				if p.parser.skipMalformedParts {
					parent.addErrorf(ErrorMalformedChildPart, "decode content: %s", err.Error())
					continue
				}
				return err
			}
			parent.AddChild(p)
			continue
		}

		parent.AddChild(p)
		// Content is another multipart.
		if err = parseParts(p, bbr); err != nil {
			if p.parser.skipMalformedParts {
				parent.addErrorf(ErrorMalformedChildPart, "parse parts: %s", err.Error())
				continue
			}
			return err
		}
	}

	// Store any content following the closing boundary marker into the epilogue.
	epilogue, err := io.ReadAll(reader)
	if err != nil {
		return errors.WithStack(err)
	}
	parent.Epilogue = epilogue

	// If a Part is "multipart/" Content-Type, it will have .0 appended to its PartID
	// i.e. it is the root of its MIME Part subtree.
	if !firstRecursion {
		parent.PartID += ".0"
	}
	return nil
}

func (p *Part) WithEncoder(e *Encoder) *Part {
	if e != nil {
		p.encoder = e
	}
	return p
}
