package config

import (
	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// HTTP defines the available http configuration.
type HTTP struct {
	Addr           string                `yaml:"addr" env:"COLLABORATION_HTTP_ADDR" desc:"The bind address of the HTTP service." introductionVersion:"1.0.0"`
	Namespace      string                `yaml:"-"`
	TLS            shared.HTTPServiceTLS `yaml:"tls"`
	TrustedProxies []string              `yaml:"trusted_proxies" env:"OC_TRUSTED_PROXIES;COLLABORATION_HTTP_TRUSTED_PROXIES" desc:"IP addresses and CIDR ranges of the reverse proxies in front of the collaboration service. The client address in the access log is only taken from the header set in COLLABORATION_HTTP_CLIENT_IP_HEADER if the request comes from one of these. Defaults to the loopback and private network ranges. See the Environment Variable Types description for more details." introductionVersion:"%%NEXT%%"`
	ClientIPHeader string                `yaml:"client_ip_header" env:"OC_CLIENT_IP_HEADER;COLLABORATION_HTTP_CLIENT_IP_HEADER" desc:"The header a trusted reverse proxy sends the client address in, e.g. X-Real-IP or CF-Connecting-IP. X-Forwarded-For is read from the right, the first address that is not a trusted proxy is the client." introductionVersion:"%%NEXT%%"`
}
