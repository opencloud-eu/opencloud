package centrifuge

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"
)

// Config contains various client options.
type Config struct {
	// Token for a connection authentication.
	Token string
	// GetToken called by SDK to get or refresh connection token.
	GetToken func(ConnectionTokenEvent) (string, error)
	// Data is an arbitrary data which can be sent to a server in a Connect command.
	// Make sure it's a valid JSON when using JSON protocol client.
	Data []byte
	// CookieJar specifies the cookie jar to send in WebSocket Upgrade request.
	CookieJar http.CookieJar
	// Header specifies custom HTTP Header to send in WebSocket Upgrade request.
	Header http.Header
	// Name allows setting client name. You should only use a limited
	// amount of client names throughout your applications – i.e. don't
	// make it unique per user for example, this name semantically represents
	// an environment from which client connects.
	// Zero value means "go".
	Name string
	// Version allows setting client version. This is an application
	// specific information. By default, no version set.
	Version string
	// Proxy specifies a function to return a proxy for a given Request.
	// If the function returns a non-nil error, the request is aborted with the
	// provided error. If function returns a nil *URL, no proxy is used.
	// If Proxy is nil then http.ProxyFromEnvironment will be used.
	Proxy func(*http.Request) (*url.URL, error)
	// NetDialContext specifies the dial function for creating TCP connections. If
	// NetDialContext is nil, net.DialContext is used.
	NetDialContext func(ctx context.Context, network, addr string) (net.Conn, error)
	// ReadTimeout is how long to wait read operations to complete.
	// Zero value means 5 * time.Second.
	ReadTimeout time.Duration
	// WriteTimeout is Websocket write timeout.
	// Zero value means 1 * time.Second.
	WriteTimeout time.Duration
	// HandshakeTimeout specifies the duration for the handshake to complete.
	// Zero value means 1 * time.Second.
	HandshakeTimeout time.Duration
	// MaxServerPingDelay used to set maximum delay of ping from server.
	// Zero value means 10 * time.Second.
	MaxServerPingDelay time.Duration
	// MinReconnectDelay is the minimum delay between reconnection attempts.
	// This delay is jittered.
	// Zero value means 200 * time.Millisecond.
	MinReconnectDelay time.Duration
	// MaxReconnectDelay is the maximum delay between reconnection attempts.
	// Zero value means 20 * time.Second.
	MaxReconnectDelay time.Duration
	// TLSConfig specifies the TLS configuration to use with tls.Client.
	// If nil, the default configuration is used.
	TLSConfig *tls.Config
	// EnableCompression specifies if the client should attempt to negotiate
	// per message compression (RFC 7692). Setting this value to true does not
	// guarantee that compression will be supported. Currently, only "no context
	// takeover" modes are supported.
	EnableCompression bool
	// LogLevel to use, by default no logs will be exposed by centrifuge-go. Most of the
	// time available protocol callbacks cover all necessary information about client-server
	// communication.
	LogLevel LogLevel
	// LogHandler is a function that will be called for each log entry. Log entries
	// are sent asynchronously and from a separate goroutine by centrifuge-go with an
	// intermediary channel buffer (fixed capacity 256). If your LogHandler is not
	// processing log entries fast enough, centrifuge-go will drop log entries.
	LogHandler func(LogEntry)
}
