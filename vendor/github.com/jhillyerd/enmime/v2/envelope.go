package enmime

import (
	"fmt"
	"io"
	"mime"
	"net/mail"
	"net/textproto"
	"strings"
	"time"

	"github.com/inbucket/html2text"
	"github.com/jhillyerd/enmime/v2/internal/coding"
	inttp "github.com/jhillyerd/enmime/v2/internal/textproto"
	"github.com/pkg/errors"
)

// Envelope is a simplified wrapper for MIME email messages.
type Envelope struct {
	Text        string  // The plain text portion of the message
	HTML        string  // The HTML portion of the message
	Root        *Part   // The top-level Part
	Attachments []*Part // All parts having a Content-Disposition of attachment
	Inlines     []*Part // All parts having a Content-Disposition of inline
	// All non-text parts that were not placed in Attachments or Inlines, such as multipart/related
	// content.
	OtherParts []*Part
	Errors     []*Error              // Errors encountered while parsing
	header     *textproto.MIMEHeader // Header from original message
}

// GetHeaderKeys returns a list of header keys seen in this message. Get
// individual headers with `GetHeader(name)`
func (e *Envelope) GetHeaderKeys() (headers []string) {
	if e.header == nil {
		return
	}
	for key := range *e.header {
		headers = append(headers, key)
	}
	return headers
}

// GetHeader processes the specified header for RFC 2047 encoded words and returns the result as a
// UTF-8 string
func (e *Envelope) GetHeader(name string) string {
	if e.header == nil {
		return ""
	}
	return coding.DecodeExtHeader(e.header.Get(name))
}

// GetHeaderValues processes the specified header for RFC 2047 encoded words and returns all existing
// values as a list of UTF-8 strings
func (e *Envelope) GetHeaderValues(name string) []string {
	if e.header == nil {
		return []string{}
	}

	rawValues := (*e.header)[inttp.CanonicalEmailMIMEHeaderKey(name)]
	values := make([]string, 0, len(rawValues))
	for _, v := range rawValues {
		values = append(values, coding.DecodeExtHeader(v))
	}
	return values
}

// SetHeader sets given header name to the given value.
// If the header exists already, all existing values are replaced.
func (e *Envelope) SetHeader(name string, value []string) error {
	if name == "" {
		return errors.New("provide non-empty header name")
	}

	for i, v := range value {
		if i == 0 {
			e.header.Set(name, mime.BEncoding.Encode("utf-8", v))
			continue
		}
		e.header.Add(name, mime.BEncoding.Encode("utf-8", v))
	}
	return nil
}

// AddHeader appends given header value to header name without changing existing values.
// If the header does not exist already, it will be created.
func (e *Envelope) AddHeader(name string, value string) error {
	if name == "" {
		return errors.New("provide non-empty header name")
	}

	e.header.Add(name, mime.BEncoding.Encode("utf-8", value))
	return nil
}

// DeleteHeader deletes given header.
func (e *Envelope) DeleteHeader(name string) error {
	if name == "" {
		return errors.New("provide non-empty header name")
	}

	e.header.Del(name)
	return nil
}

// AddressList returns a mail.Address slice with RFC 2047 encoded names converted to UTF-8
func (e *Envelope) AddressList(key string) ([]*mail.Address, error) {
	if e.header == nil {
		return nil, errors.New("no headers available")
	}
	if !AddressHeaders[strings.ToLower(key)] {
		return nil, fmt.Errorf("%s is not an address header", key)
	}

	return ParseAddressList(e.header.Get(key))
}

// Date parses the Date header field.
func (e *Envelope) Date() (time.Time, error) {
	hdr := e.GetHeader("Date")
	if hdr == "" {
		return time.Time{}, mail.ErrHeaderNotPresent
	}
	return mail.ParseDate(hdr)
}

// Clone returns a clone of the current Envelope
func (e *Envelope) Clone() *Envelope {
	if e == nil {
		return nil
	}

	newEnvelope := &Envelope{
		e.Text,
		e.HTML,
		e.Root.Clone(nil),
		e.Attachments,
		e.Inlines,
		e.OtherParts,
		e.Errors,
		e.header,
	}
	return newEnvelope
}

// ReadEnvelope is a wrapper around ReadParts and EnvelopeFromPart.  It parses the content of the
// provided reader into an Envelope, downconverting HTML to plain text if needed, and sorting the
// attachments, inlines and other parts into their respective slices. Errors are collected from all
// Parts and placed into the Envelope.Errors slice.
// Uses default parser.
func ReadEnvelope(r io.Reader) (*Envelope, error) {
	return defaultParser.ReadEnvelope(r)
}

// ReadEnvelope is the same as ReadEnvelope, but respects parser configurations.
func (p Parser) ReadEnvelope(r io.Reader) (*Envelope, error) {
	// Read MIME parts from reader
	root, err := p.ReadParts(r)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to ReadParts")
	}
	return p.EnvelopeFromPart(root)
}

// EnvelopeFromPart uses the provided Part tree to build an Envelope, downconverting HTML to plain
// text if needed, and sorting the attachments, inlines and other parts into their respective
// slices.  Errors are collected from all Parts and placed into the Envelopes Errors slice.
func EnvelopeFromPart(root *Part) (*Envelope, error) {
	return defaultParser.EnvelopeFromPart(root)
}

// EnvelopeFromPart is the same as EnvelopeFromPart, but respects parser configurations.
func (p Parser) EnvelopeFromPart(root *Part) (*Envelope, error) {
	e := &Envelope{
		Root:   root,
		header: &root.Header,
	}

	if detectMultipartMessage(root, p.multipartWOBoundaryAsSinglePart) {
		// Multi-part message (message with attachments, etc)
		if err := parseMultiPartBody(root, e); err != nil {
			return nil, err
		}
	} else {
		if detectBinaryBody(root) {
			// Attachment only, no text
			if root.Disposition == cdInline {
				e.Inlines = append(e.Inlines, root)
			} else {
				e.Attachments = append(e.Attachments, root)
			}
		} else {
			// Only text, no attachments
			parseTextOnlyBody(root, e)
		}
	}

	// Down-convert HTML to text if necessary
	if e.Text == "" && e.HTML != "" {
		// We always warn when this happens
		e.Root.addWarning(ErrorPlainTextFromHTML, "Message did not contain a text/plain part")

		if !p.disableTextConversion {
			var err error
			if e.Text, err = html2text.FromString(e.HTML); err != nil {
				e.Text = "" // Down-conversion shouldn't fail
				p := e.Root.BreadthMatchFirst(matchHTMLBodyPart)
				p.addErrorf(ErrorPlainTextFromHTML, "Failed to downconvert HTML: %v", err)
			}
		}
	}

	// Copy part errors into Envelope.
	if e.Root != nil {
		_ = e.Root.DepthMatchAll(func(part *Part) bool {
			// Using DepthMatchAll to traverse all parts, don't care about result.
			e.Errors = append(e.Errors, part.Errors...)
			return false
		})
	}

	return e, nil
}

// parseTextOnlyBody parses a plain text message in root that has MIME-like headers, but
// only contains a single part - no boundaries, etc.  The result is placed in e.
func parseTextOnlyBody(root *Part, e *Envelope) {
	// Determine character set
	var charset string
	var isHTML bool
	if ctype := root.Header.Get(hnContentType); ctype != "" {
		if mediatype, mparams, _, err := root.parseMediaType(ctype); err == nil {
			isHTML = (mediatype == ctTextHTML)
			if mparams[hpCharset] != "" {
				charset = mparams[hpCharset]
			}
		}
	}

	// Read transcoded text
	if isHTML {
		rawHTML := string(root.Content)
		// Note: Empty e.Text will trigger html2text conversion
		e.HTML = rawHTML
		if charset == "" {
			// Search for charset in HTML metadata
			if charset = coding.FindCharsetInHTML(rawHTML); charset != "" {
				// Found charset in HTML
				if convHTML, err := coding.ConvertToUTF8String(charset, root.Content); err == nil {
					// Successful conversion
					e.HTML = convHTML
				} else {
					// Conversion failed
					root.addWarning(ErrorCharsetConversion, err.Error())
				}
			}
		}
	} else {
		e.Text = string(root.Content)
	}
}

// parseMultiPartBody parses a multipart message in root.  The result is placed in e.
func parseMultiPartBody(root *Part, e *Envelope) error {
	// Parse top-level multipart
	ctype := root.Header.Get(hnContentType)
	mediatype, params, _, err := root.parseMediaType(ctype)
	if err != nil {
		return fmt.Errorf("unable to parse media type: %v", err)
	}
	if !strings.HasPrefix(mediatype, ctMultipartPrefix) {
		return fmt.Errorf("unknown mediatype: %v", mediatype)
	}
	boundary := params[hpBoundary]
	if boundary == "" {
		return errors.New("unable to locate boundary param in Content-Type header")
	}

	// Locate text body
	if mediatype == ctMultipartAltern {
		p := root.BreadthMatchFirst(func(p *Part) bool {
			return p.ContentType == ctTextPlain && p.Disposition != cdAttachment
		})
		if p != nil {
			e.Text = string(p.Content)
		}
	} else {
		// multipart is of a mixed type
		parts := root.DepthMatchAll(func(p *Part) bool {
			return p.ContentType == ctTextPlain && p.Disposition != cdAttachment
		})
		for i, p := range parts {
			if i > 0 {
				e.Text += "\n--\n"
			}
			e.Text += string(p.Content)
		}
	}

	// Locate HTML body
	p := root.DepthMatchFirst(matchHTMLBodyPart)
	if p != nil {
		e.HTML += string(p.Content)
	}

	// Locate attachments
	e.Attachments = root.BreadthMatchAll(func(p *Part) bool {
		return p.Disposition == cdAttachment || p.ContentType == ctAppOctetStream
	})

	// Locate inlines
	e.Inlines = root.BreadthMatchAll(func(p *Part) bool {
		return p.Disposition == cdInline && !strings.HasPrefix(p.ContentType, ctMultipartPrefix)
	})

	// Locate others parts not considered in attachments or inlines
	e.OtherParts = root.BreadthMatchAll(func(p *Part) bool {
		if strings.HasPrefix(p.ContentType, ctMultipartPrefix) {
			return false
		}
		if p.Disposition != "" {
			return false
		}
		if p.ContentType == ctAppOctetStream {
			return false
		}
		return p.ContentType != ctTextPlain && p.ContentType != ctTextHTML
	})

	return nil
}

// Used by Part matchers to locate the HTML body.  Not inlined because it's used in multiple places.
func matchHTMLBodyPart(p *Part) bool {
	return p.ContentType == ctTextHTML && p.Disposition != cdAttachment
}
