package imapwire

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/utf7"
)

// An Encoder writes IMAP data.
//
// Most methods don't return an error, instead they defer error handling until
// CRLF is called. These methods return the Encoder so that calls can be
// chained.
type Encoder struct {
	// QuotedUTF8 allows raw UTF-8 in quoted strings. This requires IMAP4rev2
	// to be available, or UTF8=ACCEPT to be enabled.
	QuotedUTF8 bool
	// LiteralMinus enables non-synchronizing literals for short payloads.
	// This requires IMAP4rev2 or LITERAL-. This is only meaningful for
	// clients.
	LiteralMinus bool
	// LiteralPlus enables non-synchronizing literals for all payloads. This
	// requires LITERAL+. This is only meaningful for clients.
	LiteralPlus bool
	// NewContinuationRequest creates a new continuation request. This is only
	// meaningful for clients.
	NewContinuationRequest func() *ContinuationRequest

	w       *bufio.Writer
	side    ConnSide
	err     error
	literal bool
}

// NewEncoder creates a new encoder.
func NewEncoder(w *bufio.Writer, side ConnSide) *Encoder {
	return &Encoder{w: w, side: side}
}

func (enc *Encoder) setErr(err error) {
	if enc.err == nil {
		enc.err = err
	}
}

func (enc *Encoder) writeString(s string) *Encoder {
	if enc.err != nil {
		return enc
	}
	if enc.literal {
		enc.err = fmt.Errorf("imapwire: cannot encode while a literal is open")
		return enc
	}
	if _, err := enc.w.WriteString(s); err != nil {
		enc.err = err
	}
	return enc
}

// CRLF writes a "\r\n" sequence and flushes the buffered writer.
func (enc *Encoder) CRLF() error {
	enc.writeString("\r\n")
	if enc.err != nil {
		return enc.err
	}
	return enc.w.Flush()
}

func (enc *Encoder) Atom(s string) *Encoder {
	return enc.writeString(s)
}

func (enc *Encoder) SP() *Encoder {
	return enc.writeString(" ")
}

func (enc *Encoder) Special(ch byte) *Encoder {
	return enc.writeString(string(ch))
}

func (enc *Encoder) Quoted(s string) *Encoder {
	var sb strings.Builder
	sb.Grow(2 + len(s))
	sb.WriteByte('"')
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '"' || ch == '\\' {
			sb.WriteByte('\\')
		}
		sb.WriteByte(ch)
	}
	sb.WriteByte('"')
	return enc.writeString(sb.String())
}

func (enc *Encoder) String(s string) *Encoder {
	if !enc.validQuoted(s) {
		enc.stringLiteral(s)
		return enc
	}
	return enc.Quoted(s)
}

func (enc *Encoder) validQuoted(s string) bool {
	if len(s) > 4096 {
		return false
	}

	for i := 0; i < len(s); i++ {
		ch := s[i]

		// NUL, CR and LF are never valid
		switch ch {
		case 0, '\r', '\n':
			return false
		}

		if !enc.QuotedUTF8 && ch > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func (enc *Encoder) stringLiteral(s string) {
	var sync *ContinuationRequest
	if enc.side == ConnSideClient && (!enc.LiteralMinus || len(s) > 4096) && !enc.LiteralPlus {
		if enc.NewContinuationRequest != nil {
			sync = enc.NewContinuationRequest()
		}
		if sync == nil {
			enc.setErr(fmt.Errorf("imapwire: cannot send synchronizing literal"))
			return
		}
	}
	wc := enc.Literal(int64(len(s)), sync)
	_, writeErr := io.WriteString(wc, s)
	closeErr := wc.Close()
	if writeErr != nil {
		enc.setErr(writeErr)
	} else if closeErr != nil {
		enc.setErr(closeErr)
	}
}

func (enc *Encoder) Mailbox(name string) *Encoder {
	if strings.EqualFold(name, "INBOX") {
		return enc.Atom("INBOX")
	} else {
		if enc.QuotedUTF8 {
			name = utf7.Escape(name)
		} else {
			name = utf7.Encode(name)
		}
		return enc.String(name)
	}
}

func (enc *Encoder) NumSet(numSet imap.NumSet) *Encoder {
	s := numSet.String()
	if s == "" {
		enc.setErr(fmt.Errorf("imapwire: cannot encode empty sequence set"))
		return enc
	}
	return enc.writeString(s)
}

func (enc *Encoder) Flag(flag imap.Flag) *Encoder {
	if flag != "\\*" && !isValidFlag(string(flag)) {
		enc.setErr(fmt.Errorf("imapwire: invalid flag %q", flag))
		return enc
	}
	return enc.writeString(string(flag))
}

func (enc *Encoder) MailboxAttr(attr imap.MailboxAttr) *Encoder {
	if !strings.HasPrefix(string(attr), "\\") || !isValidFlag(string(attr)) {
		enc.setErr(fmt.Errorf("imapwire: invalid mailbox attribute %q", attr))
		return enc
	}
	return enc.writeString(string(attr))
}

// isValidFlag checks whether the provided string satisfies
// flag-keyword / flag-extension.
func isValidFlag(s string) bool {
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '\\' {
			if i != 0 {
				return false
			}
		} else {
			if !IsAtomChar(ch) {
				return false
			}
		}
	}
	return len(s) > 0
}

func (enc *Encoder) Number(v uint32) *Encoder {
	return enc.writeString(strconv.FormatUint(uint64(v), 10))
}

func (enc *Encoder) Number64(v int64) *Encoder {
	// TODO: disallow negative values
	return enc.writeString(strconv.FormatInt(v, 10))
}

func (enc *Encoder) ModSeq(v uint64) *Encoder {
	// TODO: disallow zero values
	return enc.writeString(strconv.FormatUint(v, 10))
}

// List writes a parenthesized list.
func (enc *Encoder) List(n int, f func(i int)) *Encoder {
	enc.Special('(')
	for i := 0; i < n; i++ {
		if i > 0 {
			enc.SP()
		}
		f(i)
	}
	enc.Special(')')
	return enc
}

func (enc *Encoder) BeginList() *ListEncoder {
	enc.Special('(')
	return &ListEncoder{enc: enc}
}

func (enc *Encoder) NIL() *Encoder {
	return enc.Atom("NIL")
}

func (enc *Encoder) Text(s string) *Encoder {
	return enc.writeString(s)
}

func (enc *Encoder) UID(uid imap.UID) *Encoder {
	return enc.Number(uint32(uid))
}

// Literal writes a literal.
//
// The caller must write exactly size bytes to the returned writer.
//
// If sync is non-nil, the literal is synchronizing: the encoder will wait for
// nil to be sent to the channel before writing the literal data. If an error
// is sent to the channel, the literal will be cancelled.
func (enc *Encoder) Literal(size int64, sync *ContinuationRequest) io.WriteCloser {
	if sync != nil && enc.side == ConnSideServer {
		panic("imapwire: sync must be nil on a server-side Encoder.Literal")
	}

	// TODO: literal8
	enc.writeString("{")
	enc.Number64(size)
	if sync == nil && enc.side == ConnSideClient {
		enc.writeString("+")
	}
	enc.writeString("}")

	if sync == nil {
		enc.writeString("\r\n")
	} else {
		if err := enc.CRLF(); err != nil {
			return errorWriter{err}
		}
		if _, err := sync.Wait(); err != nil {
			enc.setErr(err)
			return errorWriter{err}
		}
	}

	enc.literal = true
	return &literalWriter{
		enc: enc,
		n:   size,
	}
}

type errorWriter struct {
	err error
}

func (ew errorWriter) Write(b []byte) (int, error) {
	return 0, ew.err
}

func (ew errorWriter) Close() error {
	return ew.err
}

type literalWriter struct {
	enc *Encoder
	n   int64
}

func (lw *literalWriter) Write(b []byte) (int, error) {
	if lw.n-int64(len(b)) < 0 {
		return 0, fmt.Errorf("wrote too many bytes in literal")
	}
	n, err := lw.enc.w.Write(b)
	lw.n -= int64(n)
	return n, err
}

func (lw *literalWriter) Close() error {
	lw.enc.literal = false
	if lw.n != 0 {
		return fmt.Errorf("wrote too few bytes in literal (%v remaining)", lw.n)
	}
	return nil
}

type ListEncoder struct {
	enc *Encoder
	n   int
}

func (le *ListEncoder) Item() *Encoder {
	if le.n > 0 {
		le.enc.SP()
	}
	le.n++
	return le.enc
}

func (le *ListEncoder) End() {
	le.enc.Special(')')
	le.enc = nil
}
