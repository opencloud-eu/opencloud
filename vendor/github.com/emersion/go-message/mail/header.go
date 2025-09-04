package mail

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emersion/go-message"
)

const dateLayout = "Mon, 02 Jan 2006 15:04:05 -0700"

type headerParser struct {
	s string
}

func (p *headerParser) len() int {
	return len(p.s)
}

func (p *headerParser) empty() bool {
	return p.len() == 0
}

func (p *headerParser) peek() byte {
	return p.s[0]
}

func (p *headerParser) consume(c byte) bool {
	if p.empty() || p.peek() != c {
		return false
	}
	p.s = p.s[1:]
	return true
}

// skipSpace skips the leading space and tab characters.
func (p *headerParser) skipSpace() {
	p.s = strings.TrimLeft(p.s, " \t")
}

// skipCFWS skips CFWS as defined in RFC5322. It returns false if the CFWS is
// malformed.
func (p *headerParser) skipCFWS() bool {
	p.skipSpace()

	for {
		if !p.consume('(') {
			break
		}

		if _, ok := p.consumeComment(); !ok {
			return false
		}

		p.skipSpace()
	}

	return true
}

func (p *headerParser) consumeComment() (string, bool) {
	// '(' already consumed.
	depth := 1

	var comment string
	for {
		if p.empty() || depth == 0 {
			break
		}

		if p.peek() == '\\' && p.len() > 1 {
			p.s = p.s[1:]
		} else if p.peek() == '(' {
			depth++
		} else if p.peek() == ')' {
			depth--
		}

		if depth > 0 {
			comment += p.s[:1]
		}

		p.s = p.s[1:]
	}

	return comment, depth == 0
}

func (p *headerParser) parseAtomText(dot bool) (string, error) {
	i := 0
	for {
		r, size := utf8.DecodeRuneInString(p.s[i:])
		if size == 1 && r == utf8.RuneError {
			return "", fmt.Errorf("mail: invalid UTF-8 in atom-text: %q", p.s)
		} else if size == 0 || !isAtext(r, dot) {
			break
		}
		i += size
	}
	if i == 0 {
		return "", errors.New("mail: invalid string")
	}

	var atom string
	atom, p.s = p.s[:i], p.s[i:]
	return atom, nil
}

func isAtext(r rune, dot bool) bool {
	switch r {
	case '.':
		return dot
	// RFC 5322 3.2.3 specials
	case '(', ')', '[', ']', ';', '@', '\\', ',':
		return false
	case '<', '>', '"', ':':
		return false
	}
	return isVchar(r)
}

// isVchar reports whether r is an RFC 5322 VCHAR character.
func isVchar(r rune) bool {
	// Visible (printing) characters
	return '!' <= r && r <= '~' || isMultibyte(r)
}

// isMultibyte reports whether r is a multi-byte UTF-8 character
// as supported by RFC 6532
func isMultibyte(r rune) bool {
	return r >= utf8.RuneSelf
}

func (p *headerParser) parseNoFoldLiteral() (string, error) {
	if !p.consume('[') {
		return "", errors.New("mail: missing '[' in no-fold-literal")
	}

	i := 0
	for {
		r, size := utf8.DecodeRuneInString(p.s[i:])
		if size == 1 && r == utf8.RuneError {
			return "", fmt.Errorf("mail: invalid UTF-8 in no-fold-literal: %q", p.s)
		} else if size == 0 || !isDtext(r) {
			break
		}
		i += size
	}
	var lit string
	lit, p.s = p.s[:i], p.s[i:]

	if !p.consume(']') {
		return "", errors.New("mail: missing ']' in no-fold-literal")
	}
	return "[" + lit + "]", nil
}

func isDtext(r rune) bool {
	switch r {
	case '[', ']', '\\':
		return false
	}
	return isVchar(r)
}

func (p *headerParser) parseMsgID() (string, error) {
	if !p.skipCFWS() {
		return "", errors.New("mail: malformed parenthetical comment")
	}

	if !p.consume('<') {
		return "", errors.New("mail: missing '<' in msg-id")
	}

	left, err := p.parseAtomText(true)
	if err != nil {
		return "", err
	}

	if !p.consume('@') {
		return "", errors.New("mail: missing '@' in msg-id")
	}

	var right string
	if !p.empty() && p.peek() == '[' {
		// no-fold-literal
		right, err = p.parseNoFoldLiteral()
	} else {
		right, err = p.parseAtomText(true)
	}
	if err != nil {
		return "", err
	}

	if !p.consume('>') {
		return "", errors.New("mail: missing '>' in msg-id")
	}

	if !p.skipCFWS() {
		return "", errors.New("mail: malformed parenthetical comment")
	}

	return left + "@" + right, nil
}

// A Header is a mail header.
type Header struct {
	message.Header
}

// HeaderFromMap creates a header from a map of header fields.
//
// This function is provided for interoperability with the standard library.
// If possible, ReadHeader should be used instead to avoid loosing information.
// The map representation looses the ordering of the fields, the capitalization
// of the header keys, and the whitespace of the original header.
func HeaderFromMap(m map[string][]string) Header {
	return Header{message.HeaderFromMap(m)}
}

// AddressList parses the named header field as a list of addresses. If the
// header field is missing, it returns nil.
//
// This can be used on From, Sender, Reply-To, To, Cc and Bcc header fields.
func (h *Header) AddressList(key string) ([]*Address, error) {
	v := h.Get(key)
	if v == "" {
		return nil, nil
	}
	return ParseAddressList(v)
}

// SetAddressList formats the named header field to the provided list of
// addresses.
//
// This can be used on From, Sender, Reply-To, To, Cc and Bcc header fields.
func (h *Header) SetAddressList(key string, addrs []*Address) {
	if len(addrs) > 0 {
		h.Set(key, formatAddressList(addrs))
	} else {
		h.Del(key)
	}
}

// Date parses the Date header field. If the header field is missing, it
// returns the zero time.
func (h *Header) Date() (time.Time, error) {
	v := h.Get("Date")
	if v == "" {
		return time.Time{}, nil
	}
	return mail.ParseDate(v)
}

// SetDate formats the Date header field.
func (h *Header) SetDate(t time.Time) {
	if !t.IsZero() {
		h.Set("Date", t.Format(dateLayout))
	} else {
		h.Del("Date")
	}
}

// Subject parses the Subject header field. If there is an error, the raw field
// value is returned alongside the error.
func (h *Header) Subject() (string, error) {
	return h.Text("Subject")
}

// SetSubject formats the Subject header field.
func (h *Header) SetSubject(s string) {
	h.SetText("Subject", s)
}

// MessageID parses the Message-ID field. It returns the message identifier,
// without the angle brackets. If the message doesn't have a Message-ID header
// field, it returns an empty string.
func (h *Header) MessageID() (string, error) {
	v := h.Get("Message-Id")
	if v == "" {
		return "", nil
	}

	p := headerParser{v}
	return p.parseMsgID()
}

// MsgIDList parses a list of message identifiers. It returns message
// identifiers without angle brackets. If the header field is missing, it
// returns nil.
//
// This can be used on In-Reply-To and References header fields.
func (h *Header) MsgIDList(key string) ([]string, error) {
	v := h.Get(key)
	if v == "" {
		return nil, nil
	}

	p := headerParser{v}
	var l []string
	for !p.empty() {
		msgID, err := p.parseMsgID()
		if err != nil {
			return l, err
		}
		l = append(l, msgID)
	}

	return l, nil
}

// GenerateMessageID wraps GenerateMessageIDWithHostname and therefore uses the
// hostname of the local machine. This is done to not break existing software.
// Wherever possible better use GenerateMessageIDWithHostname, because the local
// hostname of a machine tends to not be unique nor a FQDN which especially
// brings problems with spam filters.
func (h *Header) GenerateMessageID() error {
	var err error
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	return h.GenerateMessageIDWithHostname(hostname)
}

// GenerateMessageIDWithHostname generates an RFC 2822-compliant Message-Id
// based on the informational draft "Recommendations for generating Message
// IDs", it takes an hostname as argument, so that software using this library
// could use a hostname they know to be unique
func (h *Header) GenerateMessageIDWithHostname(hostname string) error {
	now := uint64(time.Now().UnixNano())

	nonceByte := make([]byte, 8)
	if _, err := rand.Read(nonceByte); err != nil {
		return err
	}
	nonce := binary.BigEndian.Uint64(nonceByte)

	msgID := fmt.Sprintf("%s.%s@%s", base36(now), base36(nonce), hostname)
	h.SetMessageID(msgID)
	return nil
}

func base36(input uint64) string {
	return strings.ToUpper(strconv.FormatUint(input, 36))
}

// SetMessageID sets the Message-ID field. id is the message identifier,
// without the angle brackets.
func (h *Header) SetMessageID(id string) {
	if id != "" {
		h.Set("Message-Id", "<"+id+">")
	} else {
		h.Del("Message-Id")
	}
}

// SetMsgIDList formats a list of message identifiers. Message identifiers
// don't include angle brackets.
//
// This can be used on In-Reply-To and References header fields.
func (h *Header) SetMsgIDList(key string, l []string) {
	if len(l) > 0 {
		h.Set(key, "<"+strings.Join(l, "> <")+">")
	} else {
		h.Del(key)
	}
}

// Copy creates a stand-alone copy of the header.
func (h *Header) Copy() Header {
	return Header{h.Header.Copy()}
}
