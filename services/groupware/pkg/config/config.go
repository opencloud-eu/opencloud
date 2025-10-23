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

	Tracing *Tracing `yaml:"tracing"`
	Log     *Log     `yaml:"log"`
	Debug   Debug    `yaml:"debug"`

	HTTP HTTP `yaml:"http"`

	Mail Mail `yaml:"mail"`

	TokenManager *TokenManager `yaml:"token_manager"`

	Context context.Context `yaml:"-"`
}

type MailMasterAuth struct {
	Username string `yaml:"username" env:"GROUPWARE_JMAP_MASTER_USERNAME"`
	Password string `yaml:"password" env:"GROUPWARE_JMAP_MASTER_PASSWORD"`
}

type MailSessionCache struct {
	MaxCapacity int           `yaml:"max_capacity" env:"GROUPWARE_SESSION_CACHE_MAX_CAPACITY"`
	Ttl         time.Duration `yaml:"ttl" env:"GROUPWARE_SESSION_CACHE_TTL"`
	FailureTtl  time.Duration `yaml:"failure_ttl" env:"GROUPWARE_SESSION_FAILURE_CACHE_TTL"`
}

type Mail struct {
	Master                MailMasterAuth   `yaml:"master"`
	BaseUrl               string           `yaml:"base_url" env:"GROUPWARE_JMAP_BASE_URL"`
	Timeout               time.Duration    `yaml:"timeout" env:"GROUPWARE_JMAP_TIMEOUT"`
	DefaultEmailLimit     uint             `yaml:"default_email_limit" env:"GROUPWARE_DEFAULT_EMAIL_LIMIT"`
	MaxBodyValueBytes     uint             `yaml:"max_body_value_bytes" env:"GROUPWARE_MAX_BODY_VALUE_BYTES"`
	DefaultContactLimit   uint             `yaml:"default_contact_limit" env:"GROUPWARE_DEFAULT_CONTACT_LIMIT"`
	ResponseHeaderTimeout time.Duration    `yaml:"response_header_timeout" env:"GROUPWARE_RESPONSE_HEADER_TIMEOUT"`
	PushHandshakeTimeout  time.Duration    `yaml:"push_handshake_timeout" env:"GROUPWARE_PUSH_HANDSHAKE_TIMEOUT"`
	SessionCache          MailSessionCache `yaml:"session_cache"`
}
