package imapwire

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapnum"
	"github.com/emersion/go-imap/v2/internal/utf7"
)

// This limits the max list nesting depth to prevent stack overflow.
const maxListDepth = 1000

// IsAtomChar returns true if ch is an ATOM-CHAR.
func IsAtomChar(ch byte) bool {
	switch ch {
	case '(', ')', '{', ' ', '%', '*', '"', '\\', ']':
		return false
	default:
		return !unicode.IsControl(rune(ch))
	}
}

// Is non-empty char
func isAStringChar(ch byte) bool {
	return IsAtomChar(ch) || ch == ']'
}

// DecoderExpectError is an error due to the Decoder.Expect family of methods.
type DecoderExpectError struct {
	Message string
}

func (err *DecoderExpectError) Error() string {
	return fmt.Sprintf("imapwire: %v", err.Message)
}

// A Decoder reads IMAP data.
//
// There are multiple families of methods:
//
//   - Methods directly named after IMAP grammar elements attempt to decode
//     said element, and return false if it's another element.
//   - "Expect" methods do the same, but set the decoder error (see Err) on
//     failure.
type Decoder struct {
	// CheckBufferedLiteralFunc is called when a literal is about to be decoded
	// and needs to be fully buffered in memory.
	CheckBufferedLiteralFunc func(size int64, nonSync bool) error
	// MaxSize defines a maximum number of bytes to be read from the input.
	// Literals are ignored.
	MaxSize int64

	r         *bufio.Reader
	side      ConnSide
	err       error
	literal   bool
	crlf      bool
	listDepth int
	readBytes int64
}

// NewDecoder creates a new decoder.
func NewDecoder(r *bufio.Reader, side ConnSide) *Decoder {
	return &Decoder{r: r, side: side}
}

func (dec *Decoder) mustUnreadByte() {
	if err := dec.r.UnreadByte(); err != nil {
		panic(fmt.Errorf("imapwire: failed to unread byte: %v", err))
	}
	dec.readBytes--
}

// Err returns the decoder error, if any.
func (dec *Decoder) Err() error {
	return dec.err
}

func (dec *Decoder) returnErr(err error) bool {
	if err == nil {
		return true
	}
	if dec.err == nil {
		dec.err = err
	}
	return false
}

func (dec *Decoder) readByte() (byte, bool) {
	if dec.MaxSize > 0 && dec.readBytes > dec.MaxSize {
		return 0, dec.returnErr(fmt.Errorf("imapwire: max size exceeded"))
	}
	dec.crlf = false
	if dec.literal {
		return 0, dec.returnErr(fmt.Errorf("imapwire: cannot decode while a literal is open"))
	}
	b, err := dec.r.ReadByte()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return b, dec.returnErr(err)
	}
	dec.readBytes++
	return b, true
}

func (dec *Decoder) acceptByte(want byte) bool {
	got, ok := dec.readByte()
	if !ok {
		return false
	} else if got != want {
		dec.mustUnreadByte()
		return false
	}
	return true
}

// EOF returns true if end-of-file is reached.
func (dec *Decoder) EOF() bool {
	_, err := dec.r.ReadByte()
	if err == io.EOF {
		return true
	} else if err != nil {
		return dec.returnErr(err)
	}
	dec.mustUnreadByte()
	return false
}

// Expect sets the decoder error if ok is false.
func (dec *Decoder) Expect(ok bool, name string) bool {
	if !ok {
		msg := fmt.Sprintf("expected %v", name)
		if dec.r.Buffered() > 0 {
			b, _ := dec.r.Peek(1)
			msg += fmt.Sprintf(", got %q", b)
		}
		return dec.returnErr(&DecoderExpectError{Message: msg})
	}
	return true
}

func (dec *Decoder) SP() bool {
	if dec.acceptByte(' ') {
		// https://github.com/emersion/go-imap/issues/571
		b, ok := dec.readByte()
		if !ok {
			return false
		}
		dec.mustUnreadByte()
		return b != '\r' && b != '\n'
	}

	// Special case: SP is optional if the next field is a parenthesized list
	b, ok := dec.readByte()
	if !ok {
		return false
	}
	dec.mustUnreadByte()
	return b == '('
}

func (dec *Decoder) ExpectSP() bool {
	return dec.Expect(dec.SP(), "SP")
}

func (dec *Decoder) CRLF() bool {
	dec.acceptByte(' ')  // https://github.com/emersion/go-imap/issues/540
	dec.acceptByte('\r') // be liberal in what we receive and accept lone LF
	if !dec.acceptByte('\n') {
		return false
	}
	dec.crlf = true
	return true
}

func (dec *Decoder) ExpectCRLF() bool {
	return dec.Expect(dec.CRLF(), "CRLF")
}

func (dec *Decoder) Func(ptr *string, valid func(ch byte) bool) bool {
	var sb strings.Builder
	for {
		b, ok := dec.readByte()
		if !ok {
			return false
		}

		if !valid(b) {
			dec.mustUnreadByte()
			break
		}

		sb.WriteByte(b)
	}
	if sb.Len() == 0 {
		return false
	}
	*ptr = sb.String()
	return true
}

func (dec *Decoder) Atom(ptr *string) bool {
	return dec.Func(ptr, IsAtomChar)
}

func (dec *Decoder) ExpectAtom(ptr *string) bool {
	return dec.Expect(dec.Atom(ptr), "atom")
}

func (dec *Decoder) ExpectNIL() bool {
	var s string
	return dec.ExpectAtom(&s) && dec.Expect(s == "NIL", "NIL")
}

func (dec *Decoder) Special(b byte) bool {
	return dec.acceptByte(b)
}

func (dec *Decoder) ExpectSpecial(b byte) bool {
	return dec.Expect(dec.Special(b), fmt.Sprintf("'%v'", string(b)))
}

func (dec *Decoder) Text(ptr *string) bool {
	var sb strings.Builder
	for {
		b, ok := dec.readByte()
		if !ok {
			return false
		} else if b == '\r' || b == '\n' {
			dec.mustUnreadByte()
			break
		}
		sb.WriteByte(b)
	}
	if sb.Len() == 0 {
		return false
	}
	*ptr = sb.String()
	return true
}

func (dec *Decoder) ExpectText(ptr *string) bool {
	return dec.Expect(dec.Text(ptr), "text")
}

func (dec *Decoder) DiscardUntilByte(untilCh byte) {
	for {
		ch, ok := dec.readByte()
		if !ok {
			return
		} else if ch == untilCh {
			dec.mustUnreadByte()
			return
		}
	}
}

func (dec *Decoder) DiscardLine() {
	if dec.crlf {
		return
	}
	var text string
	dec.Text(&text)
	dec.CRLF()
}

func (dec *Decoder) DiscardValue() bool {
	var s string
	if dec.String(&s) {
		return true
	}

	isList, err := dec.List(func() error {
		if !dec.DiscardValue() {
			return dec.Err()
		}
		return nil
	})
	if err != nil {
		return false
	} else if isList {
		return true
	}

	if dec.Atom(&s) {
		return true
	}

	dec.Expect(false, "value")
	return false
}

func (dec *Decoder) numberStr() (s string, ok bool) {
	var sb strings.Builder
	for {
		ch, ok := dec.readByte()
		if !ok {
			return "", false
		} else if ch < '0' || ch > '9' {
			dec.mustUnreadByte()
			break
		}
		sb.WriteByte(ch)
	}
	if sb.Len() == 0 {
		return "", false
	}
	return sb.String(), true
}

func (dec *Decoder) Number(ptr *uint32) bool {
	s, ok := dec.numberStr()
	if !ok {
		return false
	}
	v64, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return false // can happen on overflow
	}
	*ptr = uint32(v64)
	return true
}

func (dec *Decoder) ExpectNumber(ptr *uint32) bool {
	return dec.Expect(dec.Number(ptr), "number")
}

func (dec *Decoder) ExpectBodyFldOctets(ptr *uint32) bool {
	// Workaround: some servers incorrectly return "-1" for the body structure
	// size. See:
	// https://github.com/emersion/go-imap/issues/534
	if dec.acceptByte('-') {
		*ptr = 0
		return dec.Expect(dec.acceptByte('1'), "-1 (body-fld-octets workaround)")
	}
	return dec.ExpectNumber(ptr)
}

func (dec *Decoder) Number64(ptr *int64) bool {
	s, ok := dec.numberStr()
	if !ok {
		return false
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return false // can happen on overflow
	}
	*ptr = v
	return true
}

func (dec *Decoder) ExpectNumber64(ptr *int64) bool {
	return dec.Expect(dec.Number64(ptr), "number64")
}

func (dec *Decoder) ModSeq(ptr *uint64) bool {
	s, ok := dec.numberStr()
	if !ok {
		return false
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return false // can happen on overflow
	}
	*ptr = v
	return true
}

func (dec *Decoder) ExpectModSeq(ptr *uint64) bool {
	return dec.Expect(dec.ModSeq(ptr), "mod-sequence-value")
}

func (dec *Decoder) Quoted(ptr *string) bool {
	if !dec.Special('"') {
		return false
	}
	var sb strings.Builder
	for {
		ch, ok := dec.readByte()
		if !ok {
			return false
		}

		if ch == '"' {
			break
		}

		if ch == '\\' {
			ch, ok = dec.readByte()
			if !ok {
				return false
			}
		}

		sb.WriteByte(ch)
	}
	*ptr = sb.String()
	return true
}

func (dec *Decoder) ExpectAString(ptr *string) bool {
	if dec.Quoted(ptr) {
		return true
	}
	if dec.Literal(ptr) {
		return true
	}
	// We cannot do dec.Atom(ptr) here because sometimes mailbox names are unquoted,
	// and they can contain special characters like `]`.
	return dec.Expect(dec.Func(ptr, isAStringChar), "ASTRING-CHAR")
}

func (dec *Decoder) String(ptr *string) bool {
	return dec.Quoted(ptr) || dec.Literal(ptr)
}

func (dec *Decoder) ExpectString(ptr *string) bool {
	return dec.Expect(dec.String(ptr), "string")
}

func (dec *Decoder) ExpectNString(ptr *string) bool {
	var s string
	if dec.Atom(&s) {
		if !dec.Expect(s == "NIL", "nstring") {
			return false
		}
		*ptr = ""
		return true
	}
	return dec.ExpectString(ptr)
}

func (dec *Decoder) ExpectNStringReader() (lit *LiteralReader, nonSync, ok bool) {
	var s string
	if dec.Atom(&s) {
		if !dec.Expect(s == "NIL", "nstring") {
			return nil, false, false
		}
		return nil, true, true
	}
	// TODO: read quoted string as a string instead of buffering
	if dec.Quoted(&s) {
		return newLiteralReaderFromString(s), true, true
	}
	if lit, nonSync, ok = dec.LiteralReader(); ok {
		return lit, nonSync, true
	} else {
		return nil, false, dec.Expect(false, "nstring")
	}
}

func (dec *Decoder) List(f func() error) (isList bool, err error) {
	if !dec.Special('(') {
		return false, nil
	}
	if dec.Special(')') {
		return true, nil
	}

	dec.listDepth++
	defer func() {
		dec.listDepth--
	}()

	if dec.listDepth >= maxListDepth {
		return false, fmt.Errorf("imapwire: exceeded max depth")
	}

	for {
		if err := f(); err != nil {
			return true, err
		}

		if dec.Special(')') {
			return true, nil
		} else if !dec.ExpectSP() {
			return true, dec.Err()
		}
	}
}

func (dec *Decoder) ExpectList(f func() error) error {
	isList, err := dec.List(f)
	if err != nil {
		return err
	} else if !dec.Expect(isList, "(") {
		return dec.Err()
	}
	return nil
}

func (dec *Decoder) ExpectNList(f func() error) error {
	var s string
	if dec.Atom(&s) {
		if !dec.Expect(s == "NIL", "NIL") {
			return dec.Err()
		}
		return nil
	}
	return dec.ExpectList(f)
}

func (dec *Decoder) ExpectMailbox(ptr *string) bool {
	var name string
	if !dec.ExpectAString(&name) {
		return false
	}
	if strings.EqualFold(name, "INBOX") {
		*ptr = "INBOX"
		return true
	}
	name, err := utf7.Decode(name)
	if err == nil {
		*ptr = name
	}
	return dec.returnErr(err)
}

func (dec *Decoder) ExpectUID(ptr *imap.UID) bool {
	var num uint32
	if !dec.ExpectNumber(&num) {
		return false
	}
	*ptr = imap.UID(num)
	return true
}

func (dec *Decoder) ExpectNumSet(kind NumKind, ptr *imap.NumSet) bool {
	if dec.Special('$') {
		*ptr = imap.SearchRes()
		return true
	}

	var s string
	if !dec.Expect(dec.Func(&s, isNumSetChar), "sequence-set") {
		return false
	}
	numSet, err := imapnum.ParseSet(s)
	if err != nil {
		return dec.returnErr(err)
	}

	switch kind {
	case NumKindSeq:
		*ptr = seqSetFromNumSet(numSet)
	case NumKindUID:
		*ptr = uidSetFromNumSet(numSet)
	}
	return true
}

func (dec *Decoder) ExpectUIDSet(ptr *imap.UIDSet) bool {
	var numSet imap.NumSet
	ok := dec.ExpectNumSet(NumKindUID, &numSet)
	if ok {
		*ptr = numSet.(imap.UIDSet)
	}
	return ok
}

func isNumSetChar(ch byte) bool {
	return ch == '*' || IsAtomChar(ch)
}

func (dec *Decoder) Literal(ptr *string) bool {
	lit, nonSync, ok := dec.LiteralReader()
	if !ok {
		return false
	}
	if dec.CheckBufferedLiteralFunc != nil {
		if err := dec.CheckBufferedLiteralFunc(lit.Size(), nonSync); err != nil {
			lit.cancel()
			return false
		}
	}
	var sb strings.Builder
	_, err := io.Copy(&sb, lit)
	if err == nil {
		*ptr = sb.String()
	}
	return dec.returnErr(err)
}

func (dec *Decoder) LiteralReader() (lit *LiteralReader, nonSync, ok bool) {
	if !dec.Special('{') {
		return nil, false, false
	}
	var size int64
	if !dec.ExpectNumber64(&size) {
		return nil, false, false
	}
	if dec.side == ConnSideServer {
		nonSync = dec.acceptByte('+')
	}
	if !dec.ExpectSpecial('}') || !dec.ExpectCRLF() {
		return nil, false, false
	}
	dec.literal = true
	lit = &LiteralReader{
		dec:  dec,
		size: size,
		r:    io.LimitReader(dec.r, size),
	}
	return lit, nonSync, true
}

func (dec *Decoder) ExpectLiteralReader() (lit *LiteralReader, nonSync bool, err error) {
	lit, nonSync, ok := dec.LiteralReader()
	if !dec.Expect(ok, "literal") {
		return nil, false, dec.Err()
	}
	return lit, nonSync, nil
}

type LiteralReader struct {
	dec  *Decoder
	size int64
	r    io.Reader
}

func newLiteralReaderFromString(s string) *LiteralReader {
	return &LiteralReader{
		size: int64(len(s)),
		r:    strings.NewReader(s),
	}
}

func (lit *LiteralReader) Size() int64 {
	return lit.size
}

func (lit *LiteralReader) Read(b []byte) (int, error) {
	n, err := lit.r.Read(b)
	if err == io.EOF {
		lit.cancel()
	}
	return n, err
}

func (lit *LiteralReader) cancel() {
	if lit.dec == nil {
		return
	}
	lit.dec.literal = false
	lit.dec = nil
}
