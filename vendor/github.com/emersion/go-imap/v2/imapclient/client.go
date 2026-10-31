// Package imapclient implements an IMAP client.
//
// # Charset decoding
//
// By default, only basic charset decoding is performed. For non-UTF-8 decoding
// of message subjects and e-mail address names, users can set
// Options.WordDecoder. For instance, to use go-message's collection of
// charsets:
//
//	import (
//		"mime"
//
//		"github.com/emersion/go-message/charset"
//	)
//
//	options := &imapclient.Options{
//		WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
//	}
//	client, err := imapclient.DialTLS("imap.example.org:993", options)
package imapclient

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

const (
	idleReadTimeout    = time.Duration(0)
	respReadTimeout    = 30 * time.Second
	literalReadTimeout = 5 * time.Minute

	cmdWriteTimeout     = 30 * time.Second
	literalWriteTimeout = 5 * time.Minute
)

var dialer = &net.Dialer{
	Timeout: 30 * time.Second,
}

// SelectedMailbox contains metadata for the currently selected mailbox.
type SelectedMailbox struct {
	Name           string
	NumMessages    uint32
	Flags          []imap.Flag
	PermanentFlags []imap.Flag
}

func (mbox *SelectedMailbox) copy() *SelectedMailbox {
	copy := *mbox
	return &copy
}

// Options contains options for Client.
type Options struct {
	// TLS configuration for use by DialTLS and DialStartTLS. If nil, the
	// default configuration is used.
	TLSConfig *tls.Config
	// Raw ingress and egress data will be written to this writer, if any.
	// Note, this may include sensitive information such as credentials used
	// during authentication.
	DebugWriter io.Writer
	// Unilateral data handler.
	UnilateralDataHandler *UnilateralDataHandler
	// Decoder for RFC 2047 words.
	WordDecoder *mime.WordDecoder
}

func (options *Options) wrapReadWriter(rw io.ReadWriter) io.ReadWriter {
	if options.DebugWriter == nil {
		return rw
	}
	return struct {
		io.Reader
		io.Writer
	}{
		Reader: io.TeeReader(rw, options.DebugWriter),
		Writer: io.MultiWriter(rw, options.DebugWriter),
	}
}

func (options *Options) decodeText(s string) (string, error) {
	wordDecoder := options.WordDecoder
	if wordDecoder == nil {
		wordDecoder = &mime.WordDecoder{}
	}
	out, err := wordDecoder.DecodeHeader(s)
	if err != nil {
		return s, err
	}
	return out, nil
}

func (options *Options) unilateralDataHandler() *UnilateralDataHandler {
	if options.UnilateralDataHandler == nil {
		return &UnilateralDataHandler{}
	}
	return options.UnilateralDataHandler
}

func (options *Options) tlsConfig() *tls.Config {
	if options != nil && options.TLSConfig != nil {
		return options.TLSConfig.Clone()
	} else {
		return new(tls.Config)
	}
}

// Client is an IMAP client.
//
// IMAP commands are exposed as methods. These methods will block until the
// command has been sent to the server, but won't block until the server sends
// a response. They return a command struct which can be used to wait for the
// server response. This can be used to execute multiple commands concurrently,
// however care must be taken to avoid ambiguities. See RFC 9051 section 5.5.
//
// A client can be safely used from multiple goroutines, however this doesn't
// guarantee any command ordering and is subject to the same caveats as command
// pipelining (see above). Additionally, some commands (e.g. StartTLS,
// Authenticate, Idle) block the client during their execution.
type Client struct {
	conn     net.Conn
	options  Options
	br       *bufio.Reader
	bw       *bufio.Writer
	dec      *imapwire.Decoder
	encMutex sync.Mutex

	greetingCh   chan struct{}
	greetingRecv bool
	greetingErr  error

	decCh  chan struct{}
	decErr error

	mutex        sync.Mutex
	state        imap.ConnState
	caps         imap.CapSet
	enabled      imap.CapSet
	pendingCapCh chan struct{}
	mailbox      *SelectedMailbox
	cmdTag       uint64
	pendingCmds  []command
	contReqs     []continuationRequest
	closed       bool
}

// New creates a new IMAP client.
//
// This function doesn't perform I/O.
//
// A nil options pointer is equivalent to a zero options value.
func New(conn net.Conn, options *Options) *Client {
	if options == nil {
		options = &Options{}
	}

	rw := options.wrapReadWriter(conn)
	br := bufio.NewReader(rw)
	bw := bufio.NewWriter(rw)

	client := &Client{
		conn:       conn,
		options:    *options,
		br:         br,
		bw:         bw,
		dec:        imapwire.NewDecoder(br, imapwire.ConnSideClient),
		greetingCh: make(chan struct{}),
		decCh:      make(chan struct{}),
		state:      imap.ConnStateNone,
		enabled:    make(imap.CapSet),
	}
	go client.read()
	return client
}

// NewStartTLS creates a new IMAP client with STARTTLS.
//
// A nil options pointer is equivalent to a zero options value.
func NewStartTLS(conn net.Conn, options *Options) (*Client, error) {
	if options == nil {
		options = &Options{}
	}

	client := New(conn, options)
	if err := client.startTLS(options.TLSConfig); err != nil {
		conn.Close()
		return nil, err
	}

	// Per section 7.1.4, refuse PREAUTH when using STARTTLS
	if client.State() != imap.ConnStateNotAuthenticated {
		client.Close()
		return nil, fmt.Errorf("imapclient: server sent PREAUTH on unencrypted connection")
	}

	return client, nil
}

// DialInsecure connects to an IMAP server without any encryption at all.
func DialInsecure(address string, options *Options) (*Client, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	return New(conn, options), nil
}

// DialTLS connects to an IMAP server with implicit TLS.
func DialTLS(address string, options *Options) (*Client, error) {
	tlsConfig := options.tlsConfig()
	if tlsConfig.NextProtos == nil {
		tlsConfig.NextProtos = []string{"imap"}
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	if err != nil {
		return nil, err
	}
	return New(conn, options), nil
}

// DialStartTLS connects to an IMAP server with STARTTLS.
func DialStartTLS(address string, options *Options) (*Client, error) {
	if options == nil {
		options = &Options{}
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	tlsConfig := options.tlsConfig()
	if tlsConfig.ServerName == "" {
		tlsConfig.ServerName = host
	}
	newOptions := *options
	newOptions.TLSConfig = tlsConfig
	return NewStartTLS(conn, &newOptions)
}

func (c *Client) setReadTimeout(dur time.Duration) {
	if dur > 0 {
		c.conn.SetReadDeadline(time.Now().Add(dur))
	} else {
		c.conn.SetReadDeadline(time.Time{})
	}
}

func (c *Client) setWriteTimeout(dur time.Duration) {
	if dur > 0 {
		c.conn.SetWriteDeadline(time.Now().Add(dur))
	} else {
		c.conn.SetWriteDeadline(time.Time{})
	}
}

// State returns the current connection state of the client.
func (c *Client) State() imap.ConnState {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.state
}

func (c *Client) setState(state imap.ConnState) {
	c.mutex.Lock()
	c.state = state
	if c.state != imap.ConnStateSelected {
		c.mailbox = nil
	}
	c.mutex.Unlock()
}

// Caps returns the capabilities advertised by the server.
//
// When the server hasn't sent the capability list, this method will request it
// and block until it's received. If the capabilities cannot be fetched, nil is
// returned.
func (c *Client) Caps() imap.CapSet {
	if err := c.WaitGreeting(); err != nil {
		return nil
	}

	c.mutex.Lock()
	caps := c.caps
	capCh := c.pendingCapCh
	c.mutex.Unlock()

	if caps != nil {
		return caps
	}

	if capCh == nil {
		capCmd := c.Capability()
		capCh := make(chan struct{})
		go func() {
			capCmd.Wait()
			close(capCh)
		}()
		c.mutex.Lock()
		c.pendingCapCh = capCh
		c.mutex.Unlock()
	}

	timer := time.NewTimer(respReadTimeout)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-capCh:
		// ok
	}

	// TODO: this is racy if caps are reset before we get the reply
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.caps
}

func (c *Client) setCaps(caps imap.CapSet) {
	// If the capabilities are being reset, request the updated capabilities
	// from the server
	var capCh chan struct{}
	if caps == nil {
		capCh = make(chan struct{})

		// We need to send the CAPABILITY command in a separate goroutine:
		// setCaps might be called with Client.encMutex locked
		go func() {
			c.Capability().Wait()
			close(capCh)
		}()
	}

	c.mutex.Lock()
	c.caps = caps
	c.pendingCapCh = capCh
	c.mutex.Unlock()
}

// Mailbox returns the state of the currently selected mailbox.
//
// If there is no currently selected mailbox, nil is returned.
//
// The returned struct must not be mutated.
func (c *Client) Mailbox() *SelectedMailbox {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.mailbox
}

// Close immediately closes the connection.
func (c *Client) Close() error {
	c.mutex.Lock()
	alreadyClosed := c.closed
	c.closed = true
	c.mutex.Unlock()

	// Ignore net.ErrClosed here, because we also call conn.Close in c.read
	if err := c.conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.ErrClosedPipe) {
		return err
	}

	<-c.decCh
	if err := c.decErr; err != nil {
		return err
	}

	if alreadyClosed {
		return net.ErrClosed
	}
	return nil
}

// beginCommand starts sending a command to the server.
//
// The command name and a space are written.
//
// The caller must call commandEncoder.end.
func (c *Client) beginCommand(name string, cmd command) *commandEncoder {
	c.encMutex.Lock() // unlocked by commandEncoder.end

	c.mutex.Lock()

	c.cmdTag++
	tag := fmt.Sprintf("T%v", c.cmdTag)

	baseCmd := cmd.base()
	*baseCmd = commandBase{
		tag:  tag,
		done: make(chan error, 1),
	}

	c.pendingCmds = append(c.pendingCmds, cmd)
	quotedUTF8 := c.caps.Has(imap.CapIMAP4rev2) || c.enabled.Has(imap.CapUTF8Accept)
	literalMinus := c.caps.Has(imap.CapLiteralMinus)
	literalPlus := c.caps.Has(imap.CapLiteralPlus)

	c.mutex.Unlock()

	c.setWriteTimeout(cmdWriteTimeout)

	wireEnc := imapwire.NewEncoder(c.bw, imapwire.ConnSideClient)
	wireEnc.QuotedUTF8 = quotedUTF8
	wireEnc.LiteralMinus = literalMinus
	wireEnc.LiteralPlus = literalPlus
	wireEnc.NewContinuationRequest = func() *imapwire.ContinuationRequest {
		return c.registerContReq(cmd)
	}

	enc := &commandEncoder{
		Encoder: wireEnc,
		client:  c,
		cmd:     baseCmd,
	}
	enc.Atom(tag).SP().Atom(name)
	return enc
}

func (c *Client) deletePendingCmdByTag(tag string) command {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for i, cmd := range c.pendingCmds {
		if cmd.base().tag == tag {
			c.pendingCmds = append(c.pendingCmds[:i], c.pendingCmds[i+1:]...)
			return cmd
		}
	}
	return nil
}

func (c *Client) findPendingCmdFunc(f func(cmd command) bool) command {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, cmd := range c.pendingCmds {
		if f(cmd) {
			return cmd
		}
	}
	return nil
}

func findPendingCmdByType[T command](c *Client) T {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, cmd := range c.pendingCmds {
		if cmd, ok := cmd.(T); ok {
			return cmd
		}
	}

	var cmd T
	return cmd
}

func (c *Client) completeCommand(cmd command, err error) {
	done := cmd.base().done
	done <- err
	close(done)

	// Ensure the command is not blocked waiting on continuation requests
	c.mutex.Lock()
	var filtered []continuationRequest
	for _, contReq := range c.contReqs {
		if contReq.cmd != cmd.base() {
			filtered = append(filtered, contReq)
		} else {
			contReq.Cancel(err)
		}
	}
	c.contReqs = filtered
	c.mutex.Unlock()

	switch cmd := cmd.(type) {
	case *authenticateCommand, *loginCommand:
		if err == nil {
			c.setState(imap.ConnStateAuthenticated)
		}
	case *unauthenticateCommand:
		if err == nil {
			c.mutex.Lock()
			c.state = imap.ConnStateNotAuthenticated
			c.mailbox = nil
			c.enabled = make(imap.CapSet)
			c.mutex.Unlock()
		}
	case *SelectCommand:
		if err == nil {
			c.mutex.Lock()
			c.state = imap.ConnStateSelected
			c.mailbox = &SelectedMailbox{
				Name:           cmd.mailbox,
				NumMessages:    cmd.data.NumMessages,
				Flags:          cmd.data.Flags,
				PermanentFlags: cmd.data.PermanentFlags,
			}
			c.mutex.Unlock()
		}
	case *unselectCommand:
		if err == nil {
			c.setState(imap.ConnStateAuthenticated)
		}
	case *logoutCommand:
		if err == nil {
			c.setState(imap.ConnStateLogout)
		}
	case *ListCommand:
		if cmd.pendingData != nil {
			cmd.mailboxes <- cmd.pendingData
		}
		close(cmd.mailboxes)
	case *FetchCommand:
		close(cmd.msgs)
	case *ExpungeCommand:
		close(cmd.seqNums)
	}
}

func (c *Client) registerContReq(cmd command) *imapwire.ContinuationRequest {
	contReq := imapwire.NewContinuationRequest()

	c.mutex.Lock()
	c.contReqs = append(c.contReqs, continuationRequest{
		ContinuationRequest: contReq,
		cmd:                 cmd.base(),
	})
	c.mutex.Unlock()

	return contReq
}

func (c *Client) closeWithError(err error) {
	c.conn.Close()

	c.mutex.Lock()
	c.state = imap.ConnStateLogout
	pendingCmds := c.pendingCmds
	c.pendingCmds = nil
	c.mutex.Unlock()

	for _, cmd := range pendingCmds {
		c.completeCommand(cmd, err)
	}
}

// read continuously reads data coming from the server.
//
// All the data is decoded in the read goroutine, then dispatched via channels
// to pending commands.
func (c *Client) read() {
	defer close(c.decCh)
	defer func() {
		if v := recover(); v != nil {
			c.decErr = fmt.Errorf("imapclient: panic reading response: %v\n%s", v, debug.Stack())
		}

		cmdErr := c.decErr
		if cmdErr == nil {
			cmdErr = io.ErrUnexpectedEOF
		}
		c.closeWithError(cmdErr)
	}()

	c.setReadTimeout(respReadTimeout) // We're waiting for the greeting
	for {
		// Ignore net.ErrClosed here, because we also call conn.Close in c.Close
		if c.dec.EOF() || errors.Is(c.dec.Err(), net.ErrClosed) || errors.Is(c.dec.Err(), io.ErrClosedPipe) {
			break
		}
		if err := c.readResponse(); err != nil {
			c.decErr = err
			break
		}
		if c.greetingErr != nil {
			break
		}
	}
}

func (c *Client) readResponse() error {
	c.setReadTimeout(respReadTimeout)
	defer c.setReadTimeout(idleReadTimeout)

	if c.dec.Special('+') {
		if err := c.readContinueReq(); err != nil {
			return fmt.Errorf("in continue-req: %v", err)
		}
		return nil
	}

	var tag, typ string
	if !c.dec.Expect(c.dec.Special('*') || c.dec.Atom(&tag), "'*' or atom") {
		return fmt.Errorf("in response: cannot read tag: %v", c.dec.Err())
	}
	if !c.dec.ExpectSP() {
		return fmt.Errorf("in response: %v", c.dec.Err())
	}
	if !c.dec.ExpectAtom(&typ) {
		return fmt.Errorf("in response: cannot read type: %v", c.dec.Err())
	}

	// Change typ to uppercase, as it's case-insensitive
	typ = strings.ToUpper(typ)

	var (
		token    string
		err      error
		startTLS *startTLSCommand
	)
	if tag != "" {
		token = "response-tagged"
		startTLS, err = c.readResponseTagged(tag, typ)
	} else {
		token = "response-data"
		err = c.readResponseData(typ)
	}
	if err != nil {
		return fmt.Errorf("in %v: %v", token, err)
	}

	if !c.dec.ExpectCRLF() {
		return fmt.Errorf("in response: %v", c.dec.Err())
	}

	if startTLS != nil {
		c.upgradeStartTLS(startTLS)
	}

	return nil
}

func (c *Client) readContinueReq() error {
	var text string
	if c.dec.SP() {
		c.dec.Text(&text)
	}
	if !c.dec.ExpectCRLF() {
		return c.dec.Err()
	}

	var contReq *imapwire.ContinuationRequest
	c.mutex.Lock()
	if len(c.contReqs) > 0 {
		contReq = c.contReqs[0].ContinuationRequest
		c.contReqs = append(c.contReqs[:0], c.contReqs[1:]...)
	}
	c.mutex.Unlock()

	if contReq == nil {
		return fmt.Errorf("received unmatched continuation request")
	}

	contReq.Done(text)
	return nil
}

func (c *Client) readResponseTagged(tag, typ string) (startTLS *startTLSCommand, err error) {
	cmd := c.deletePendingCmdByTag(tag)
	if cmd == nil {
		return nil, fmt.Errorf("received tagged response with unknown tag %q", tag)
	}

	// We've removed the command from the pending queue above. Make sure we
	// don't stall it on error.
	defer func() {
		if err != nil {
			c.completeCommand(cmd, err)
		}
	}()

	// Some servers don't provide a text even if the RFC requires it,
	// see #500 and #502
	hasSP := c.dec.SP()

	var code string
	if hasSP && c.dec.Special('[') { // resp-text-code
		if !c.dec.ExpectAtom(&code) {
			return nil, fmt.Errorf("in resp-text-code: %v", c.dec.Err())
		}
		// TODO: LONGENTRIES and MAXSIZE from METADATA
		switch code {
		case "CAPABILITY": // capability-data
			caps, err := readCapabilities(c.dec)
			if err != nil {
				return nil, fmt.Errorf("in capability-data: %v", err)
			}
			c.setCaps(caps)
		case "APPENDUID":
			var (
				uidValidity uint32
				uid         imap.UID
			)
			if !c.dec.ExpectSP() || !c.dec.ExpectNumber(&uidValidity) || !c.dec.ExpectSP() || !c.dec.ExpectUID(&uid) {
				return nil, fmt.Errorf("in resp-code-apnd: %v", c.dec.Err())
			}
			if cmd, ok := cmd.(*AppendCommand); ok {
				cmd.data.UID = uid
				cmd.data.UIDValidity = uidValidity
			}
		case "COPYUID":
			if !c.dec.ExpectSP() {
				return nil, c.dec.Err()
			}
			uidValidity, srcUIDs, dstUIDs, err := readRespCodeCopyUID(c.dec)
			if err != nil {
				return nil, fmt.Errorf("in resp-code-copy: %v", err)
			}
			switch cmd := cmd.(type) {
			case *CopyCommand:
				cmd.data.UIDValidity = uidValidity
				cmd.data.SourceUIDs = srcUIDs
				cmd.data.DestUIDs = dstUIDs
			case *MoveCommand:
				// This can happen when Client.Move falls back to COPY +
				// STORE + EXPUNGE
				cmd.data.UIDValidity = uidValidity
				cmd.data.SourceUIDs = srcUIDs
				cmd.data.DestUIDs = dstUIDs
			}
		default: // [SP 1*<any TEXT-CHAR except "]">]
			if c.dec.SP() {
				c.dec.DiscardUntilByte(']')
			}
		}
		if !c.dec.ExpectSpecial(']') {
			return nil, fmt.Errorf("in resp-text: %v", c.dec.Err())
		}
		hasSP = c.dec.SP()
	}
	var text string
	if hasSP && !c.dec.ExpectText(&text) {
		return nil, fmt.Errorf("in resp-text: %v", c.dec.Err())
	}

	var cmdErr error
	switch typ {
	case "OK":
		// nothing to do
	case "NO", "BAD":
		cmdErr = &imap.Error{
			Type: imap.StatusResponseType(typ),
			Code: imap.ResponseCode(code),
			Text: text,
		}
	default:
		return nil, fmt.Errorf("in resp-cond-state: expected OK, NO or BAD status condition, but got %v", typ)
	}

	c.completeCommand(cmd, cmdErr)

	if cmd, ok := cmd.(*startTLSCommand); ok && cmdErr == nil {
		startTLS = cmd
	}

	if cmdErr == nil && code != "CAPABILITY" {
		switch cmd.(type) {
		case *startTLSCommand, *loginCommand, *authenticateCommand, *unauthenticateCommand:
			// These commands invalidate the capabilities
			c.setCaps(nil)
		}
	}

	return startTLS, nil
}

func (c *Client) readResponseData(typ string) error {
	// number SP ("EXISTS" / "RECENT" / "FETCH" / "EXPUNGE")
	var num uint32
	if typ[0] >= '0' && typ[0] <= '9' {
		v, err := strconv.ParseUint(typ, 10, 32)
		if err != nil {
			return err
		}

		num = uint32(v)
		if !c.dec.ExpectSP() || !c.dec.ExpectAtom(&typ) {
			return c.dec.Err()
		}
	}

	// All response type are case insensitive
	switch strings.ToUpper(typ) {
	case "OK", "PREAUTH", "NO", "BAD", "BYE": // resp-cond-state / resp-cond-bye / resp-cond-auth
		// Some servers don't provide a text even if the RFC requires it,
		// see #500 and #502
		hasSP := c.dec.SP()

		var code string
		if hasSP && c.dec.Special('[') { // resp-text-code
			if !c.dec.ExpectAtom(&code) {
				return fmt.Errorf("in resp-text-code: %v", c.dec.Err())
			}
			switch code {
			case "CAPABILITY": // capability-data
				caps, err := readCapabilities(c.dec)
				if err != nil {
					return fmt.Errorf("in capability-data: %v", err)
				}
				c.setCaps(caps)
			case "PERMANENTFLAGS":
				if !c.dec.ExpectSP() {
					return c.dec.Err()
				}
				flags, err := internal.ExpectFlagList(c.dec)
				if err != nil {
					return err
				}

				c.mutex.Lock()
				if c.state == imap.ConnStateSelected {
					c.mailbox = c.mailbox.copy()
					c.mailbox.PermanentFlags = flags
				}
				c.mutex.Unlock()

				if cmd := findPendingCmdByType[*SelectCommand](c); cmd != nil {
					cmd.data.PermanentFlags = flags
				} else if handler := c.options.unilateralDataHandler().Mailbox; handler != nil {
					handler(&UnilateralDataMailbox{PermanentFlags: flags})
				}
			case "UIDNEXT":
				var uidNext imap.UID
				if !c.dec.ExpectSP() || !c.dec.ExpectUID(&uidNext) {
					return c.dec.Err()
				}
				if cmd := findPendingCmdByType[*SelectCommand](c); cmd != nil {
					cmd.data.UIDNext = uidNext
				}
			case "UIDVALIDITY":
				var uidValidity uint32
				if !c.dec.ExpectSP() || !c.dec.ExpectNumber(&uidValidity) {
					return c.dec.Err()
				}
				if cmd := findPendingCmdByType[*SelectCommand](c); cmd != nil {
					cmd.data.UIDValidity = uidValidity
				}
			case "COPYUID":
				if !c.dec.ExpectSP() {
					return c.dec.Err()
				}
				uidValidity, srcUIDs, dstUIDs, err := readRespCodeCopyUID(c.dec)
				if err != nil {
					return fmt.Errorf("in resp-code-copy: %v", err)
				}
				if cmd := findPendingCmdByType[*MoveCommand](c); cmd != nil {
					cmd.data.UIDValidity = uidValidity
					cmd.data.SourceUIDs = srcUIDs
					cmd.data.DestUIDs = dstUIDs
				}
			case "HIGHESTMODSEQ":
				var modSeq uint64
				if !c.dec.ExpectSP() || !c.dec.ExpectModSeq(&modSeq) {
					return c.dec.Err()
				}
				if cmd := findPendingCmdByType[*SelectCommand](c); cmd != nil {
					cmd.data.HighestModSeq = modSeq
				}
			case "NOMODSEQ":
				// ignore
			default: // [SP 1*<any TEXT-CHAR except "]">]
				if c.dec.SP() {
					c.dec.DiscardUntilByte(']')
				}
			}
			if !c.dec.ExpectSpecial(']') {
				return fmt.Errorf("in resp-text: %v", c.dec.Err())
			}
			hasSP = c.dec.SP()
		}

		var text string
		if hasSP && !c.dec.ExpectText(&text) {
			return fmt.Errorf("in resp-text: %v", c.dec.Err())
		}

		if code == "CLOSED" {
			c.setState(imap.ConnStateAuthenticated)
		}

		if !c.greetingRecv {
			switch typ {
			case "OK":
				c.setState(imap.ConnStateNotAuthenticated)
			case "PREAUTH":
				c.setState(imap.ConnStateAuthenticated)
			default:
				c.setState(imap.ConnStateLogout)
				c.greetingErr = &imap.Error{
					Type: imap.StatusResponseType(typ),
					Code: imap.ResponseCode(code),
					Text: text,
				}
			}
			c.greetingRecv = true
			if c.greetingErr == nil && code != "CAPABILITY" {
				c.setCaps(nil) // request initial capabilities
			}
			close(c.greetingCh)
		}
	case "ID":
		return c.handleID()
	case "CAPABILITY":
		return c.handleCapability()
	case "ENABLED":
		return c.handleEnabled()
	case "NAMESPACE":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleNamespace()
	case "FLAGS":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleFlags()
	case "EXISTS":
		return c.handleExists(num)
	case "RECENT":
		// ignore
	case "LIST":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleList()
	case "STATUS":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleStatus()
	case "FETCH":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleFetch(num)
	case "EXPUNGE":
		return c.handleExpunge(num)
	case "SEARCH":
		return c.handleSearch()
	case "ESEARCH":
		return c.handleESearch()
	case "SORT":
		return c.handleSort()
	case "THREAD":
		return c.handleThread()
	case "METADATA":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleMetadata()
	case "QUOTA":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleQuota()
	case "QUOTAROOT":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleQuotaRoot()
	case "MYRIGHTS":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleMyRights()
	case "ACL":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleGetACL()
	default:
		return fmt.Errorf("unsupported response type %q", typ)
	}

	return nil
}

// WaitGreeting waits for the server's initial greeting.
func (c *Client) WaitGreeting() error {
	select {
	case <-c.greetingCh:
		return c.greetingErr
	case <-c.decCh:
		if c.decErr != nil {
			return fmt.Errorf("got error before greeting: %v", c.decErr)
		}
		return fmt.Errorf("connection closed before greeting")
	}
}

// Noop sends a NOOP command.
func (c *Client) Noop() *Command {
	cmd := &Command{}
	c.beginCommand("NOOP", cmd).end()
	return cmd
}

// Logout sends a LOGOUT command.
//
// This command informs the server that the client is done with the connection.
func (c *Client) Logout() *Command {
	cmd := &logoutCommand{}
	c.beginCommand("LOGOUT", cmd).end()
	return &cmd.Command
}

// Login sends a LOGIN command.
func (c *Client) Login(username, password string) *Command {
	cmd := &loginCommand{}
	enc := c.beginCommand("LOGIN", cmd)
	enc.SP().String(username).SP().String(password)
	enc.end()
	return &cmd.Command
}

// Delete sends a DELETE command.
func (c *Client) Delete(mailbox string) *Command {
	cmd := &Command{}
	enc := c.beginCommand("DELETE", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// Rename sends a RENAME command.
func (c *Client) Rename(mailbox, newName string) *Command {
	cmd := &Command{}
	enc := c.beginCommand("RENAME", cmd)
	enc.SP().Mailbox(mailbox).SP().Mailbox(newName)
	enc.end()
	return cmd
}

// Subscribe sends a SUBSCRIBE command.
func (c *Client) Subscribe(mailbox string) *Command {
	cmd := &Command{}
	enc := c.beginCommand("SUBSCRIBE", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// Subscribe sends an UNSUBSCRIBE command.
func (c *Client) Unsubscribe(mailbox string) *Command {
	cmd := &Command{}
	enc := c.beginCommand("UNSUBSCRIBE", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

func uidCmdName(name string, kind imapwire.NumKind) string {
	switch kind {
	case imapwire.NumKindSeq:
		return name
	case imapwire.NumKindUID:
		return "UID " + name
	default:
		panic("imapclient: invalid imapwire.NumKind")
	}
}

type commandEncoder struct {
	*imapwire.Encoder
	client *Client
	cmd    *commandBase
}

// end ends an outgoing command.
//
// A CRLF is written, the encoder is flushed and its lock is released.
func (ce *commandEncoder) end() {
	if ce.Encoder != nil {
		ce.flush()
	}
	ce.client.setWriteTimeout(0)
	ce.client.encMutex.Unlock()
}

// flush sends an outgoing command, but keeps the encoder lock.
//
// A CRLF is written and the encoder is flushed. Callers must call
// commandEncoder.end to release the lock.
func (ce *commandEncoder) flush() {
	if err := ce.Encoder.CRLF(); err != nil {
		// TODO: consider stashing the error in Client to return it in future
		// calls
		ce.client.closeWithError(err)
	}
	ce.Encoder = nil
}

// Literal encodes a literal.
func (ce *commandEncoder) Literal(size int64) io.WriteCloser {
	var contReq *imapwire.ContinuationRequest
	ce.client.mutex.Lock()
	hasCapLiteralMinus := ce.client.caps.Has(imap.CapLiteralMinus)
	ce.client.mutex.Unlock()
	if size > 4096 || !hasCapLiteralMinus {
		contReq = ce.client.registerContReq(ce.cmd)
	}
	ce.client.setWriteTimeout(literalWriteTimeout)
	return literalWriter{
		WriteCloser: ce.Encoder.Literal(size, contReq),
		client:      ce.client,
	}
}

type literalWriter struct {
	io.WriteCloser
	client *Client
}

func (lw literalWriter) Close() error {
	lw.client.setWriteTimeout(cmdWriteTimeout)
	return lw.WriteCloser.Close()
}

// continuationRequest is a pending continuation request.
type continuationRequest struct {
	*imapwire.ContinuationRequest
	cmd *commandBase
}

// UnilateralDataMailbox describes a mailbox status update.
//
// If a field is nil, it hasn't changed.
type UnilateralDataMailbox struct {
	NumMessages    *uint32
	Flags          []imap.Flag
	PermanentFlags []imap.Flag
}

// UnilateralDataHandler handles unilateral data.
//
// The handler will block the client while running. If the caller intends to
// perform slow operations, a buffered channel and a separate goroutine should
// be used.
//
// The handler will be invoked in an arbitrary goroutine.
//
// See Options.UnilateralDataHandler.
type UnilateralDataHandler struct {
	Expunge func(seqNum uint32)
	Mailbox func(data *UnilateralDataMailbox)
	Fetch   func(msg *FetchMessageData)

	// requires ENABLE METADATA or ENABLE SERVER-METADATA
	Metadata func(mailbox string, entries []string)
}

// command is an interface for IMAP commands.
//
// Commands are represented by the Command type, but can be extended by other
// types (e.g. CapabilityCommand).
type command interface {
	base() *commandBase
}

type commandBase struct {
	tag  string
	done chan error
	err  error
}

func (cmd *commandBase) base() *commandBase {
	return cmd
}

func (cmd *commandBase) wait() error {
	if cmd.err == nil {
		cmd.err = <-cmd.done
	}
	return cmd.err
}

// Command is a basic IMAP command.
type Command struct {
	commandBase
}

// Wait blocks until the command has completed.
func (cmd *Command) Wait() error {
	return cmd.wait()
}

type loginCommand struct {
	Command
}

// logoutCommand is a LOGOUT command.
type logoutCommand struct {
	Command
}
