package centrifuge

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/centrifugal/protocol"
	"github.com/gorilla/websocket"
)

func extractDisconnectWebsocket(err error) *disconnect {
	if err != nil {
		if closeErr, ok := err.(*websocket.CloseError); ok {
			var d disconnect
			err := json.Unmarshal([]byte(closeErr.Text), &d)
			if err == nil {
				return &d
			} else {
				code := uint32(closeErr.Code)
				reason := closeErr.Text
				reconnect := code < 3500 || code >= 5000 || (code >= 4000 && code < 4500)
				if code < 3000 {
					switch code {
					case websocket.CloseMessageTooBig:
						code = disconnectMessageSizeLimit
					default:
						// We expose codes defined by Centrifuge protocol, hiding
						// details about transport-specific error codes. We may have extra
						// optional transportCode field in the future.
						code = connectingTransportClosed
					}
				}
				return &disconnect{
					Code:      code,
					Reason:    reason,
					Reconnect: reconnect,
				}
			}
		}
	}
	return nil
}

type websocketTransport struct {
	mu             sync.Mutex
	conn           *websocket.Conn
	protocolType   protocol.Type
	commandEncoder protocol.CommandEncoder
	replyCh        chan *protocol.Reply
	config         websocketConfig
	disconnect     *disconnect
	closed         bool
	closeCh        chan struct{}
}

// websocketConfig configures Websocket transport.
type websocketConfig struct {
	// Proxy specifies a function to return a proxy for a given Request.
	// If the function returns a non-nil error, the request is aborted with the
	// provided error. If function returns a nil *URL, no proxy is used.
	// If Proxy is nil then http.ProxyFromEnvironment will be used.
	Proxy func(*http.Request) (*url.URL, error)

	// NetDialContext specifies the dial function for creating TCP connections. If
	// NetDialContext is nil, net.DialContext is used.
	NetDialContext func(ctx context.Context, network, addr string) (net.Conn, error)

	// TLSConfig specifies the TLS configuration to use with tls.Client.
	// If nil, the default configuration is used.
	TLSConfig *tls.Config

	// HandshakeTimeout specifies the duration for the handshake to complete.
	HandshakeTimeout time.Duration

	// EnableCompression specifies if the client should attempt to negotiate
	// per message compression (RFC 7692). Setting this value to true does not
	// guarantee that compression will be supported. Currently only "no context
	// takeover" modes are supported.
	EnableCompression bool

	// CookieJar specifies the cookie jar.
	// If CookieJar is nil, cookies are not sent in requests and ignored
	// in responses.
	CookieJar http.CookieJar

	// Header specifies custom HTTP Header to send.
	Header http.Header
}

func newWebsocketTransport(url string, protocolType protocol.Type, config websocketConfig) (transport, error) {
	wsHeaders := config.Header

	dialer := &websocket.Dialer{}
	if config.Proxy != nil {
		dialer.Proxy = config.Proxy
	} else {
		dialer.Proxy = http.ProxyFromEnvironment
	}
	dialer.NetDialContext = config.NetDialContext

	dialer.HandshakeTimeout = config.HandshakeTimeout
	dialer.EnableCompression = config.EnableCompression
	dialer.TLSClientConfig = config.TLSConfig
	dialer.Jar = config.CookieJar

	if protocolType == protocol.TypeProtobuf {
		dialer.Subprotocols = []string{"centrifuge-protobuf"}
	}

	conn, resp, err := dialer.Dial(url, wsHeaders)
	if err != nil {
		return nil, fmt.Errorf("error dial: %v", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return nil, fmt.Errorf("wrong status code while connecting to server: %d", resp.StatusCode)
	}

	t := &websocketTransport{
		conn:           conn,
		replyCh:        make(chan *protocol.Reply),
		config:         config,
		closeCh:        make(chan struct{}),
		commandEncoder: newCommandEncoder(protocolType),
		protocolType:   protocolType,
	}
	go t.reader()
	return t, nil
}

func (t *websocketTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	t.closed = true
	close(t.closeCh)
	_ = t.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	return t.conn.Close()
}

func (t *websocketTransport) reader() {
	defer func() { _ = t.Close() }()
	defer close(t.replyCh)

	for {
		_, data, err := t.conn.ReadMessage()
		if err != nil {
			disconnect := extractDisconnectWebsocket(err)
			t.disconnect = disconnect
			return
		}
		//println("<----", strings.Trim(string(data), "\n"))
	loop:
		for {
			decoder := newReplyDecoder(t.protocolType, data)
			for {
				reply, err := decoder.Decode()
				if err != nil {
					if err == io.EOF {
						break loop
					}
					t.disconnect = &disconnect{Code: disconnectBadProtocol, Reason: "decode error", Reconnect: false}
					return
				}
				select {
				case <-t.closeCh:
					return
				case t.replyCh <- reply:
					// Send is blocking here, but slow client will be disconnected
					// eventually with `no ping` reason – so we will exit from this
					// goroutine.
				}
			}
		}
	}
}

func (t *websocketTransport) Write(cmd *protocol.Command, timeout time.Duration) error {
	data, err := t.commandEncoder.Encode(cmd)
	if err != nil {
		return err
	}
	return t.writeData(data, timeout)
}

func (t *websocketTransport) writeData(data []byte, timeout time.Duration) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if timeout > 0 {
		_ = t.conn.SetWriteDeadline(time.Now().Add(timeout))
	}
	//println("---->", strings.Trim(string(data), "\n"))
	var err error
	if t.protocolType == protocol.TypeJSON {
		err = t.conn.WriteMessage(websocket.TextMessage, data)
	} else {
		err = t.conn.WriteMessage(websocket.BinaryMessage, data)
	}
	if timeout > 0 {
		_ = t.conn.SetWriteDeadline(time.Time{})
	}
	return err
}

func (t *websocketTransport) Read() (*protocol.Reply, *disconnect, error) {
	reply, ok := <-t.replyCh
	if !ok {
		return nil, t.disconnect, io.EOF
	}
	return reply, nil, nil
}
