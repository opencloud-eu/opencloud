// Package imapwire implements the IMAP wire protocol.
//
// The IMAP wire protocol is defined in RFC 9051 section 4.
package imapwire

import (
	"fmt"
)

// ConnSide describes the side of a connection: client or server.
type ConnSide int

const (
	ConnSideClient ConnSide = 1 + iota
	ConnSideServer
)

// ContinuationRequest is a continuation request.
//
// The sender must call either Done or Cancel. The receiver must call Wait.
type ContinuationRequest struct {
	done chan struct{}
	err  error
	text string
}

func NewContinuationRequest() *ContinuationRequest {
	return &ContinuationRequest{done: make(chan struct{})}
}

func (cont *ContinuationRequest) Cancel(err error) {
	if err == nil {
		err = fmt.Errorf("imapwire: continuation request cancelled")
	}
	cont.err = err
	close(cont.done)
}

func (cont *ContinuationRequest) Done(text string) {
	cont.text = text
	close(cont.done)
}

func (cont *ContinuationRequest) Wait() (string, error) {
	<-cont.done
	return cont.text, cont.err
}
