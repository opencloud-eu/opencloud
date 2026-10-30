package enmime

import (
	"bufio"
	"encoding/base64"
	"io"
	"mime"
	"mime/quotedprintable"
	"net/textproto"
	"sort"
	"strings"
	"time"

	"github.com/jhillyerd/enmime/v2/internal/coding"
	"github.com/jhillyerd/enmime/v2/internal/stringutil"
)

// b64Percent determines the percent of non-ASCII characters enmime will tolerate before switching
// from quoted-printable to base64 encoding.
const b64Percent = 20

type transferEncoding byte

const (
	te7Bit transferEncoding = iota
	te8Bit
	teQuoted
	teBase64
	teRaw
)

const (
	base64EncodedLineLen = 76
	base64DecodedLineLen = base64EncodedLineLen * 3 / 4 // this is ok since lineLen is divisible by 4
	linesPerChunk        = 128
	readChunkSize        = base64DecodedLineLen * linesPerChunk
)

var crnl = []byte{'\r', '\n'}

// Encode writes this Part and all its children to the specified writer in MIME format.
func (p *Part) Encode(writer io.Writer) error {
	if p.Header == nil {
		p.Header = make(textproto.MIMEHeader)
	}
	if p.ContentReader != nil {
		// read some data in order to check whether the content is empty
		p.Content = make([]byte, readChunkSize)
		n, err := p.ContentReader.Read(p.Content)
		if err != nil && err != io.EOF {
			return err
		}
		p.Content = p.Content[:n]
	}
	cte := teRaw
	if p.parser == nil || !p.parser.rawContent {
		cte = p.setupMIMEHeaders()
	}
	// Encode this part.
	b := bufio.NewWriter(writer)
	if err := p.encodeHeader(b); err != nil {
		return err
	}
	if len(p.Content) > 0 {
		if _, err := b.Write(crnl); err != nil {
			return err
		}
		if err := p.encodeContent(b, cte); err != nil {
			return err
		}
	}
	if p.FirstChild == nil {
		return b.Flush()
	}
	// Encode children.
	endMarker := []byte("\r\n--" + p.Boundary + "--")
	marker := endMarker[:len(endMarker)-2]
	c := p.FirstChild
	for c != nil {
		if _, err := b.Write(marker); err != nil {
			return err
		}
		if _, err := b.Write(crnl); err != nil {
			return err
		}
		if err := c.Encode(b); err != nil {
			return err
		}
		c = c.NextSibling
	}
	if _, err := b.Write(endMarker); err != nil {
		return err
	}
	if _, err := b.Write(crnl); err != nil {
		return err
	}
	return b.Flush()
}

// setupMIMEHeaders determines content transfer encoding, generates a boundary string if required,
// then sets the Content-Type (type, charset, filename, boundary) and Content-Disposition headers.
func (p *Part) setupMIMEHeaders() transferEncoding {
	// Determine content transfer encoding.

	// If we are encoding a part that previously had content-transfer-encoding set, unset it so
	// the correct encoding detection can be done below.
	p.Header.Del(hnContentEncoding)

	cte := te7Bit
	if len(p.Content) > 0 {
		if strings.Index(strings.ToLower(p.ContentType), "message/") == 0 {
			// RFC 1341: `message` types must have no encoding other than "7bit", "8bit", or
			// "binary". The message header fields are always US-ASCII in any case, and data within
			// the body can still be encoded, in which case the Content-Transfer-Encoding header
			// field in the encapsulated message will reflect this.
			cte = te8Bit
		} else {
			cte = teBase64
			if p.TextContent() && p.ContentReader == nil {
				cte = p.selectTransferEncoding(p.Content, false)
				if p.Charset == "" {
					p.Charset = utf8
				}
			}
		}

		// RFC 2045: 7bit is assumed if CTE header not present.
		switch cte {
		case te8Bit:
			p.Header.Set(hnContentEncoding, cte8Bit)
		case teBase64:
			p.Header.Set(hnContentEncoding, cteBase64)
		case teQuoted:
			p.Header.Set(hnContentEncoding, cteQuotedPrintable)
		}
	}

	// Setup headers.
	if p.FirstChild != nil && p.Boundary == "" {
		// Multipart, generate random boundary marker.
		p.Boundary = "enmime-" + stringutil.UUID(p.randSource)
	}
	if p.ContentID != "" {
		p.Header.Set(hnContentID, coding.ToIDHeader(p.ContentID))
	}
	fileName := p.FileName
	switch p.selectTransferEncoding([]byte(p.FileName), true) {
	case teBase64:
		fileName = mime.BEncoding.Encode(utf8, p.FileName)
	case teQuoted:
		fileName = mime.QEncoding.Encode(utf8, p.FileName)
	}

	if p.ContentType != "" {
		// Build content type header.
		param := make(map[string]string)
		for k, v := range p.ContentTypeParams {
			param[k] = v
		}
		setParamValue(param, hpCharset, p.Charset)
		setParamValue(param, hpName, fileName)
		setParamValue(param, hpBoundary, p.Boundary)
		if mt := mime.FormatMediaType(p.ContentType, param); mt != "" {
			p.ContentType = mt
		}
		p.Header.Set(hnContentType, p.ContentType)
	}

	if p.Disposition != "" {
		// Build disposition header.
		param := make(map[string]string)
		setParamValue(param, hpFilename, fileName)
		if !p.FileModDate.IsZero() {
			setParamValue(param, hpModDate, p.FileModDate.UTC().Format(time.RFC822))
		}
		if mt := mime.FormatMediaType(p.Disposition, param); mt != "" {
			p.Disposition = mt
		}
		p.Header.Set(hnContentDisposition, p.Disposition)
	}

	return cte
}

// encodeHeader writes out a sorted list of headers.
func (p *Part) encodeHeader(b *bufio.Writer) error {
	keys := make([]string, 0, len(p.Header))
	for k := range p.Header {
		keys = append(keys, k)
	}
	rawContent := p.parser != nil && p.parser.rawContent

	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range p.Header[k] {
			encv := v
			if !rawContent {
				switch p.selectTransferEncoding([]byte(v), true) {
				case teBase64:
					encv = mime.BEncoding.Encode(utf8, v)
				case teQuoted:
					encv = mime.QEncoding.Encode(utf8, v)
				}
			}
			// _ used to prevent early wrapping
			wb := stringutil.Wrap(76, k, ":_", encv, "\r\n")
			wb[len(k)+1] = ' '
			if _, err := b.Write(wb); err != nil {
				return err
			}
		}
	}
	return nil
}

// encodeContent writes out the content in the selected encoding.
func (p *Part) encodeContent(b *bufio.Writer, cte transferEncoding) (err error) {
	if p.ContentReader != nil {
		return p.encodeContentFromReader(b)
	}

	if p.parser != nil && p.parser.rawContent {
		cte = teRaw
	}

	switch cte {
	case teBase64:
		enc := base64.StdEncoding
		text := make([]byte, enc.EncodedLen(len(p.Content)))
		enc.Encode(text, p.Content)
		// Wrap lines.
		lineLen := 76
		for len(text) > 0 {
			if lineLen > len(text) {
				lineLen = len(text)
			}
			if _, err = b.Write(text[:lineLen]); err != nil {
				return err
			}
			if _, err := b.Write(crnl); err != nil {
				return err
			}
			text = text[lineLen:]
		}
	case teQuoted:
		qp := quotedprintable.NewWriter(b)
		if _, err = qp.Write(p.Content); err != nil {
			return err
		}
		err = qp.Close()
	default:
		_, err = b.Write(p.Content)
	}
	return err
}

// encodeContentFromReader writes out the content read from the reader using base64 encoding.
func (p *Part) encodeContentFromReader(b *bufio.Writer) error {
	text := make([]byte, base64EncodedLineLen) // a single base64 encoded line
	enc := base64.StdEncoding

	chunk := make([]byte, readChunkSize) // contains a whole number of lines
	copy(chunk, p.Content)               // copy the data of the initial read that was issued by `Encode`
	n := len(p.Content)

	for {
		// call read until we get a full chunk / error
		for n < len(chunk) {
			c, err := p.ContentReader.Read(chunk[n:])
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}

			n += c
		}

		for i := 0; i < n; i += base64DecodedLineLen {
			size := n - i
			if size > base64DecodedLineLen {
				size = base64DecodedLineLen
			}

			enc.Encode(text, chunk[i:i+size])
			if _, err := b.Write(text[:enc.EncodedLen(size)]); err != nil {
				return err
			}
			if _, err := b.Write(crnl); err != nil {
				return err
			}
		}

		if n < len(chunk) {
			break
		}

		n = 0
	}

	return nil
}

// selectTransferEncoding scans content for non-ASCII characters and selects 'b' or 'q' encoding.
func (p *Part) selectTransferEncoding(content []byte, quoteLineBreaks bool) transferEncoding {
	if len(content) == 0 {
		return te7Bit
	}

	if p.encoder != nil && p.encoder.forceQuotedPrintableCteOption {
		return teQuoted
	}

	// Binary chars remaining before we choose b64 encoding.
	threshold := b64Percent * len(content) / 100
	bincount := 0
	for _, b := range content {
		if (b < ' ' || '~' < b) && b != '\t' {
			if !quoteLineBreaks && (b == '\r' || b == '\n') {
				continue
			}
			bincount++
			if bincount >= threshold {
				return teBase64
			}
		}
	}
	if bincount == 0 {
		return te7Bit
	}
	return teQuoted
}

// setParamValue will ignore empty values
func setParamValue(p map[string]string, k, v string) {
	if v != "" {
		p[k] = v
	}
}
