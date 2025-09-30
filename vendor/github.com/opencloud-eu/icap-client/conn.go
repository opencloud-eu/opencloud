package icapclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

// ICAPConnConfig is the configuration for the icap connection.
type ICAPConnConfig struct {
	// Timeout is the maximum amount of time a connection will be kept open
	Timeout time.Duration
}

// ICAPConn manages the transport layer for ICAP protocol.
type ICAPConn struct {
	tcp     net.Conn
	mu      sync.Mutex
	timeout time.Duration
}

// NewICAPConn creates a new connection configuration.
func NewICAPConn(conf ICAPConnConfig) (*ICAPConn, error) {
	return &ICAPConn{
		timeout: conf.Timeout,
	}, nil
}

// Connect connects to the ICAP server.
func (c *ICAPConn) Connect(ctx context.Context, address string) error {
	dialer := net.Dialer{Timeout: c.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	c.tcp = conn

	if c.timeout > 0 {
		deadline := time.Now().Add(c.timeout)
		if err := c.tcp.SetDeadline(deadline); err != nil {
			return err
		}
	}

	return nil
}

// Send sends a request to the ICAP server and reads the response.
func (c *ICAPConn) Send(in []byte) ([]byte, error) {
	if !c.ok() {
		return nil, ErrInvalidConnection
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.tcp.Write(in)
	if err != nil {
		return nil, err
	}

	var data []byte
	buf := make([]byte, 4096)
	for {
		n, err := c.tcp.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}

		if errors.Is(err, io.EOF) || n == 0 {
			break
		}

		data = append(data, buf[:n]...)

		// Protocol checks for message termination
		{
			if bytes.Equal(data, []byte(icap100ContinueMsg)) {
				break
			}

			if bytes.HasSuffix(data, []byte(doubleCRLF)) {
				break
			}

			if bytes.Contains(data, []byte(icap204NoModsMsg)) {
				break
			}
		}
	}

	return data, nil
}

// Close closes the TCP connection.
func (c *ICAPConn) Close() error {
	if !c.ok() {
		return ErrInvalidConnection
	}
	return c.tcp.Close()
}

func (c *ICAPConn) ok() bool {
	return c != nil && c.tcp != nil
}
