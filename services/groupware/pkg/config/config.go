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

type Mail struct {
	Master                 MailMasterAuth `yaml:"master"`
	BaseUrl                string         `yaml:"base_url" env:"GROUPWARE_JMAP_BASE_URL"`
	Timeout                time.Duration  `yaml:"timeout" env:"GROUPWARE_JMAP_TIMEOUT"`
	DefaultEmailLimit      int            `yaml:"default_email_limit" env:"GROUPWARE_DEFAULT_EMAIL_LIMIT"`
	MaxBodyValueBytes      int            `yaml:"max_body_value_bytes" env:"GROUPWARE_MAX_BODY_VALUE_BYTES"`
	ResponseHeaderTimeout  time.Duration  `yaml:"response_header_timeout" env:"GROUPWARE_RESPONSE_HEADER_TIMEOUT"`
	SessionCacheTtl        time.Duration  `yaml:"session_cache_ttl" env:"GROUPWARE_SESSION_CACHE_TTL"`
	SessionFailureCacheTtl time.Duration  `yaml:"session_failure_cache_ttl" env:"GROUPWARE_SESSION_FAILURE_CACHE_TTL"`
}
