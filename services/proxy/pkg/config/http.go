package config

import "time"

// HTTP defines the available http configuration.
type HTTP struct {
	Addr      string `yaml:"addr" env:"PROXY_HTTP_ADDR" desc:"The bind address of the HTTP service." introductionVersion:"1.0.0"`
	Root      string `yaml:"root" env:"PROXY_HTTP_ROOT" desc:"Subdirectory that serves as the root for this HTTP service." introductionVersion:"1.0.0"`
	Namespace string `yaml:"-"`
	TLSCert   string `yaml:"tls_cert" env:"PROXY_TRANSPORT_TLS_CERT" desc:"Path/File name of the TLS server certificate (in PEM format) for the external http services. If not defined, the root directory derives from $OC_BASE_DATA_PATH/proxy." introductionVersion:"1.0.0"`
	TLSKey    string `yaml:"tls_key" env:"PROXY_TRANSPORT_TLS_KEY" desc:"Path/File name for the TLS certificate key (in PEM format) for the server certificate to use for the external http services. If not defined, the root directory derives from $OC_BASE_DATA_PATH/proxy." introductionVersion:"1.0.0"`
	TLS       bool   `yaml:"tls" env:"PROXY_TLS" desc:"Enable/Disable HTTPS for external HTTP services. Must be set to 'true' if the built-in IDP service and no reverse proxy is used. See the text description for details." introductionVersion:"1.0.0"`
	Client    Client `yaml:"client"`
}

type Client struct {
	DialTimeout           time.Duration `yaml:"dial_timeout" env:"PROXY_TRANSPORT_CLIENT_DIAL_TIMEOUT" desc:"Dial timeout for the HTTP client." introductionVersion:"7.5.0"`
	DialKeepAlive         time.Duration `yaml:"dial_keep_alive" env:"PROXY_TRANSPORT_CLIENT_DIAL_KEEP_ALIVE" desc:"Dial keep alive for the HTTP client." introductionVersion:"7.5.0"`
	ForceAttemptHTTP2     bool          `yaml:"force_attempt_http2" env:"PROXY_TRANSPORT_CLIENT_FORCE_ATTEMPT_HTTP2" desc:"Force attempt HTTP/2 for the HTTP client. Set to true if backend communication is encrypted and HTTP/2 is desired." introductionVersion:"7.5.0"`
	MaxIdleConns          int           `yaml:"max_idle_conns" env:"PROXY_TRANSPORT_CLIENT_MAX_IDLE_CONNS" desc:"Maximum idle connections for the HTTP client." introductionVersion:"7.5.0"`
	MaxIdleConnsPerHost   int           `yaml:"max_idle_conns_per_host" env:"PROXY_TRANSPORT_CLIENT_MAX_IDLE_CONNS_PER_HOST" desc:"Maximum idle connections per host for the HTTP client." introductionVersion:"7.5.0"`
	IdleConnTimeout       time.Duration `yaml:"idle_conn_timeout" env:"PROXY_TRANSPORT_CLIENT_IDLE_CONN_TIMEOUT" desc:"Idle connection timeout for the HTTP client." introductionVersion:"7.5.0"`
	TLSHandshakeTimeout   time.Duration `yaml:"tls_handshake_timeout" env:"PROXY_TRANSPORT_CLIENT_TLS_HANDSHAKE_TIMEOUT" desc:"TLS handshake timeout for the HTTP client." introductionVersion:"7.5.0"`
	ExpectContinueTimeout time.Duration `yaml:"expect_continue_timeout" env:"PROXY_TRANSPORT_CLIENT_EXPECT_CONTINUE_TIMEOUT" desc:"Expect continue timeout for the HTTP client." introductionVersion:"7.5.0"`
}
