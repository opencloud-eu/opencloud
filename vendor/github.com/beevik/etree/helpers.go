// Copyright 2015-2019 Brett Vickers.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package etree

import (
	"io"
	"strings"
	"unicode/utf8"
)

type stack[E any] struct {
	data []E
}

func (s *stack[E]) empty() bool {
	return len(s.data) == 0
}

func (s *stack[E]) push(value E) {
	s.data = append(s.data, value)
}

func (s *stack[E]) pop() E {
	value := s.data[len(s.data)-1]
	var empty E
	s.data[len(s.data)-1] = empty
	s.data = s.data[:len(s.data)-1]
	return value
}

func (s *stack[E]) peek() E {
	return s.data[len(s.data)-1]
}

type queue[E any] struct {
	data       []E
	head, tail int
}

func (f *queue[E]) add(value E) {
	if f.len()+1 >= len(f.data) {
		f.grow()
	}
	f.data[f.tail] = value
	if f.tail++; f.tail == len(f.data) {
		f.tail = 0
	}
}

func (f *queue[E]) remove() E {
	value := f.data[f.head]
	var empty E
	f.data[f.head] = empty
	if f.head++; f.head == len(f.data) {
		f.head = 0
	}
	return value
}

func (f *queue[E]) len() int {
	if f.tail >= f.head {
		return f.tail - f.head
	}
	return len(f.data) - f.head + f.tail
}

func (f *queue[E]) grow() {
	c := len(f.data) * 2
	if c == 0 {
		c = 4
	}
	buf, count := make([]E, c), f.len()
	if f.tail >= f.head {
		copy(buf[:count], f.data[f.head:f.tail])
	} else {
		hindex := len(f.data) - f.head
		copy(buf[:hindex], f.data[f.head:])
		copy(buf[hindex:count], f.data[:f.tail])
	}
	f.data, f.head, f.tail = buf, 0, count
}

// xmlReader provides the interface by which an XML byte stream is
// processed and decoded.
type xmlReader interface {
	Bytes() int64
	Read(p []byte) (n int, err error)
}

// xmlSimpleReader implements a proxy reader that counts the number of
// bytes read from its encapsulated reader.
type xmlSimpleReader struct {
	r     io.Reader
	bytes int64
}

func newXmlSimpleReader(r io.Reader) xmlReader {
	return &xmlSimpleReader{r, 0}
}

func (xr *xmlSimpleReader) Bytes() int64 {
	return xr.bytes
}

func (xr *xmlSimpleReader) Read(p []byte) (n int, err error) {
	n, err = xr.r.Read(p)
	xr.bytes += int64(n)
	return n, err
}

// xmlPeekReader implements a proxy reader that counts the number of
// bytes read from its encapsulated reader. It also allows the caller to
// "peek" at the previous portions of the buffer after they have been
// parsed.
type xmlPeekReader struct {
	r          io.Reader
	bytes      int64  // total bytes read by the Read function
	buf        []byte // internal read buffer
	bufSize    int    // total bytes used in the read buffer
	bufOffset  int64  // total bytes read when buf was last filled
	window     []byte // current read buffer window
	peekBuf    []byte // buffer used to store data to be peeked at later
	peekOffset int64  // total read offset of the start of the peek buffer
}

func newXmlPeekReader(r io.Reader) *xmlPeekReader {
	buf := make([]byte, 4096)
	return &xmlPeekReader{
		r:          r,
		bytes:      0,
		buf:        buf,
		bufSize:    0,
		bufOffset:  0,
		window:     buf[0:0],
		peekBuf:    make([]byte, 0),
		peekOffset: -1,
	}
}

func (xr *xmlPeekReader) Bytes() int64 {
	return xr.bytes
}

func (xr *xmlPeekReader) Read(p []byte) (n int, err error) {
	if len(xr.window) == 0 {
		err = xr.fill()
		if err != nil {
			return 0, err
		}
		if len(xr.window) == 0 {
			return 0, nil
		}
	}

	if len(xr.window) < len(p) {
		n = len(xr.window)
	} else {
		n = len(p)
	}

	copy(p, xr.window)
	xr.window = xr.window[n:]
	xr.bytes += int64(n)

	return n, err
}

func (xr *xmlPeekReader) PeekPrepare(offset int64, maxLen int) {
	if maxLen > cap(xr.peekBuf) {
		xr.peekBuf = make([]byte, 0, maxLen)
	}
	xr.peekBuf = xr.peekBuf[0:0]
	xr.peekOffset = offset
	xr.updatePeekBuf()
}

func (xr *xmlPeekReader) PeekFinalize() []byte {
	xr.updatePeekBuf()
	return xr.peekBuf
}

func (xr *xmlPeekReader) fill() error {
	xr.bufOffset = xr.bytes
	xr.bufSize = 0
	n, err := xr.r.Read(xr.buf)
	if err != nil {
		xr.window, xr.bufSize = xr.buf[0:0], 0
		return err
	}
	xr.window, xr.bufSize = xr.buf[:n], n
	xr.updatePeekBuf()
	return nil
}

func (xr *xmlPeekReader) updatePeekBuf() {
	peekRemain := cap(xr.peekBuf) - len(xr.peekBuf)
	if xr.peekOffset >= 0 && peekRemain > 0 {
		rangeMin := xr.peekOffset
		rangeMax := xr.peekOffset + int64(cap(xr.peekBuf))
		bufMin := xr.bufOffset
		bufMax := xr.bufOffset + int64(xr.bufSize)
		if rangeMin < bufMin {
			rangeMin = bufMin
		}
		if rangeMax > bufMax {
			rangeMax = bufMax
		}
		if rangeMax > rangeMin {
			rangeMin -= xr.bufOffset
			rangeMax -= xr.bufOffset
			if int(rangeMax-rangeMin) > peekRemain {
				rangeMax = rangeMin + int64(peekRemain)
			}
			xr.peekBuf = append(xr.peekBuf, xr.buf[rangeMin:rangeMax]...)
		}
	}
}

// xmlWriter implements a proxy writer that counts the number of
// bytes written by its encapsulated writer.
type xmlWriter struct {
	w     io.Writer
	bytes int64
}

func newXmlWriter(w io.Writer) *xmlWriter {
	return &xmlWriter{w: w}
}

func (xw *xmlWriter) Write(p []byte) (n int, err error) {
	n, err = xw.w.Write(p)
	xw.bytes += int64(n)
	return n, err
}

// isWhitespace returns true if the byte slice contains only
// whitespace characters.
func isWhitespace(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i]; c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return false
		}
	}
	return true
}

// spaceMatch returns true if namespace a is the empty string
// or if namespace a equals namespace b.
func spaceMatch(a, b string) bool {
	switch {
	case a == "":
		return true
	default:
		return a == b
	}
}

// spaceDecompose breaks a namespace:tag identifier at the ':'
// and returns the two parts.
func spaceDecompose(str string) (space, key string) {
	colon := strings.IndexByte(str, ':')
	if colon == -1 {
		return "", str
	}
	return str[:colon], str[colon+1:]
}

// Strings used by indentCRLF and indentLF
const (
	indentSpaces = "\r\n                                                                "
	indentTabs   = "\r\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t"
)

// indentCRLF returns a CRLF newline followed by n copies of the first
// non-CRLF character in the source string.
func indentCRLF(n int, source string) string {
	switch {
	case n < 0:
		return source[:2]
	case n < len(source)-1:
		return source[:n+2]
	default:
		return source + strings.Repeat(source[2:3], n-len(source)+2)
	}
}

// indentLF returns a LF newline followed by n copies of the first non-LF
// character in the source string.
func indentLF(n int, source string) string {
	switch {
	case n < 0:
		return source[1:2]
	case n < len(source)-1:
		return source[1 : n+2]
	default:
		return source[1:] + strings.Repeat(source[2:3], n-len(source)+2)
	}
}

// nextIndex returns the index of the next occurrence of byte ch in s,
// starting from offset.  It returns -1 if the byte is not found.
func nextIndex(s string, ch byte, offset int) int {
	switch i := strings.IndexByte(s[offset:], ch); i {
	case -1:
		return -1
	default:
		return offset + i
	}
}

// isInteger returns true if the string s contains an integer.
func isInteger(s string) bool {
	for i := 0; i < len(s); i++ {
		if (s[i] < '0' || s[i] > '9') && !(i == 0 && s[i] == '-') {
			return false
		}
	}
	return true
}

type escapeMode byte

const (
	escapeNormal escapeMode = iota
	escapeCanonicalText
	escapeCanonicalAttr
)

// escapeString writes an escaped version of a string to the writer.
func escapeString(w Writer, s string, m escapeMode) {
	var esc []byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRuneInString(s[i:])
		i += width
		switch r {
		case '&':
			esc = []byte("&amp;")
		case '<':
			esc = []byte("&lt;")
		case '>':
			if m == escapeCanonicalAttr {
				continue
			}
			esc = []byte("&gt;")
		case '\'':
			if m != escapeNormal {
				continue
			}
			esc = []byte("&apos;")
		case '"':
			if m == escapeCanonicalText {
				continue
			}
			esc = []byte("&quot;")
		case '\t':
			if m != escapeCanonicalAttr {
				continue
			}
			esc = []byte("&#x9;")
		case '\n':
			if m != escapeCanonicalAttr {
				continue
			}
			esc = []byte("&#xA;")
		case '\r':
			if m == escapeNormal {
				continue
			}
			esc = []byte("&#xD;")
		default:
			if !isInCharacterRange(r) || (r == 0xFFFD && width == 1) {
				esc = []byte("\uFFFD")
				break
			}
			continue
		}
		w.WriteString(s[last : i-width])
		w.Write(esc)
		last = i
	}
	w.WriteString(s[last:])
}

// sanitizeCData writes the sanitized contents of a CDATA section to the
// writer. XML provides no way to escape the "]]>" sequence within a CDATA
// section, so any occurrence of it is split across two CDATA sections.
func sanitizeCData(w Writer, s string) {
	for {
		i := strings.Index(s, "]]>")
		if i < 0 {
			break
		}
		w.WriteString(s[:i+2])
		w.WriteString("]]><![CDATA[")
		s = s[i+2:]
	}
	w.WriteString(s)
}

// sanitizeComment writes the sanitized contents of a comment to the writer.
// An XML comment may not contain the string "--", and it may not end with a
// '-'. Because XML provides no way to escape these sequences, spaces are
// inserted where necessary.
func sanitizeComment(w Writer, s string) {
	last, hyphen := 0, false
	for i := 0; i < len(s); i++ {
		if s[i] != '-' {
			hyphen = false
			continue
		}
		if hyphen {
			w.WriteString(s[last:i])
			w.WriteByte(' ')
			last = i
		}
		hyphen = true
	}
	w.WriteString(s[last:])
	if hyphen {
		w.WriteByte(' ')
	}
}

// sanitizeProcInst writes the contents of a sanitized processing instruction
// to the writer. XML provides no way to escape the "?>" sequence within a
// processing instruction, so a space is inserted between the two characters.
func sanitizeProcInst(w Writer, s string) {
	for {
		i := strings.Index(s, "?>")
		if i < 0 {
			break
		}
		w.WriteString(s[:i+1])
		w.WriteByte(' ')
		s = s[i+1:]
	}
	w.WriteString(s)
}

// sanitizeDirective writes the sanitized contents of an XML directive to the
// writer.
func sanitizeDirective(w Writer, s string) {
	// The XML decoder reserves the character following "<!" for comments
	// ('-') and CDATA sections ('['), and it treats "<!>" as an unterminated
	// directive. Insert a space to avoid conflicts with reserved sequences.
	scan := s
	if s == "" || s[0] == '-' || s[0] == '[' {
		w.WriteByte(' ')
	} else {
		scan = s[1:]
	}

	// A directive's contents may legitimately contain '<' and '>' characters,
	// so write them without modification when they are balanced.
	if isDirectiveBalanced(scan) {
		w.WriteString(s)
		return
	}

	// The contents are unbalanced, so escape every character in the string.
	escapeString(w, s, escapeNormal)
}

// isDirectiveBalanced returns true if the interpreted portion of an XML
// directive's contents may be enclosed by "<!" and ">" without changing the
// extents of the resulting directive.
func isDirectiveBalanced(s string) bool {
	var quote byte
	var depth int
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '>':
			if depth == 0 {
				return false
			}
			depth--
		case c == '<':
			if !strings.HasPrefix(s[i+1:], "!--") {
				depth++
				break
			}
			j := strings.Index(s[i+4:], "-->")
			if j < 0 {
				return false
			}
			i += 4 + j + 2
		}
	}
	return quote == 0 && depth == 0
}

func isInCharacterRange(r rune) bool {
	return r == 0x09 ||
		r == 0x0A ||
		r == 0x0D ||
		r >= 0x20 && r <= 0xD7FF ||
		r >= 0xE000 && r <= 0xFFFD ||
		r >= 0x10000 && r <= 0x10FFFF
}
