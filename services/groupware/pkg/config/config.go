package config

import (
	"context"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	Log   *Log  `yaml:"log"`
	Debug Debug `yaml:"debug"`

	HTTP HTTP `yaml:"http"`

	Mail Mail `yaml:"mail"`

	TokenManager *TokenManager `yaml:"token_manager"`

	Context context.Context `yaml:"-"`
}

type MailMasterAuth struct {
	Username string `yaml:"username" env:"GROUPWARE_JMAP_MASTER_USERNAME" desc:"The username to use for master authentication for JMAP operations." introductionVersion:"4.0.0"`
	Password string `yaml:"password" env:"GROUPWARE_JMAP_MASTER_PASSWORD" desc:"The clear text password to use for master authentication for JMAP operations." introductionVersion:"4.0.0"`
}

type MailSessionCache struct {
	MaxCapacity int           `yaml:"max_capacity" env:"GROUPWARE_SESSION_CACHE_MAX_CAPACITY" desc:"The maximum capacity of the JMAP session cache." introductionVersion:"4.0.0"`
	Ttl         time.Duration `yaml:"ttl" env:"GROUPWARE_SESSION_CACHE_TTL" desc:"The time-to-live of cached successfully obtained JMAP sessions." introductionVersion:"4.0.0"`
	FailureTtl  time.Duration `yaml:"failure_ttl" env:"GROUPWARE_SESSION_FAILURE_CACHE_TTL" desc:"The time-to-live of cached JMAP session retrieval failures." introductionVersion:"4.0.0"`
}

type Mail struct {
	Master                MailMasterAuth   `yaml:"master"`
	BaseUrl               string           `yaml:"base_url" env:"GROUPWARE_JMAP_BASE_URL" desc:"The base fully-qualified URL to the JMAP server." introductionVersion:"4.0.0"`
	Timeout               time.Duration    `yaml:"timeout" env:"GROUPWARE_JMAP_TIMEOUT" desc:"The timeout for JMAP HTTP operations." introductionVersion:"4.0.0"`
	DefaultEmailLimit     uint             `yaml:"default_email_limit" env:"GROUPWARE_DEFAULT_EMAIL_LIMIT" desc:"The default email retrieval page size." introductionVersion:"4.0.0"`
	MaxBodyValueBytes     uint             `yaml:"max_body_value_bytes" env:"GROUPWARE_MAX_BODY_VALUE_BYTES" desc:"The maximum size when retrieving email bodies from the JMAP server." introductionVersion:"4.0.0"`
	DefaultContactLimit   uint             `yaml:"default_contact_limit" env:"GROUPWARE_DEFAULT_CONTACT_LIMIT" desc:"The default contacts retrieval page size." introductionVersion:"4.0.0"`
	ResponseHeaderTimeout time.Duration    `yaml:"response_header_timeout" env:"GROUPWARE_RESPONSE_HEADER_TIMEOUT" desc:"The timeout when waiting for JMAP response headers." introductionVersion:"4.0.0"`
	PushHandshakeTimeout  time.Duration    `yaml:"push_handshake_timeout" env:"GROUPWARE_PUSH_HANDSHAKE_TIMEOUT" desc:"The timeout when performing Websocket handshakes with the JMAP server." introductionVersion:"4.0.0"`
	SessionCache          MailSessionCache `yaml:"session_cache"`
}
