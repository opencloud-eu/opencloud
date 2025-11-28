package config

import (
	"context"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// ScannerType gives info which scanner is used
type ScannerType string

const (
	// ScannerTypeClamAV defines that clamav is used
	ScannerTypeClamAV ScannerType = "clamav"
	// ScannerTypeICap defines that icap is used
	ScannerTypeICap ScannerType = "icap"
)

// MaxScanSizeMode defines the mode of handling files that exceed the maximum scan size
type MaxScanSizeMode string

const (
	// MaxScanSizeModeSkip defines that files that are bigger than the max scan size will be skipped
	MaxScanSizeModeSkip MaxScanSizeMode = "skip"
	// MaxScanSizeModePartial defines that only the file up to the max size will be used
	MaxScanSizeModePartial MaxScanSizeMode = "partial"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service
	File    string
	Log     *Log

	Debug Debug `yaml:"debug" mask:"struct"`

	Service Service `yaml:"-"`

	InfectedFileHandling string `yaml:"infected-file-handling" env:"ANTIVIRUS_INFECTED_FILE_HANDLING" desc:"Defines the behaviour when a virus has been found. Supported options are: 'delete', 'continue' and 'abort '. Delete will delete the file. Continue will mark the file as infected but continues further processing. Abort will keep the file in the uploads folder for further admin inspection and will not move it to its final destination." introductionVersion:"1.0.0"`
	Events               Events
	Workers              int `yaml:"workers" env:"ANTIVIRUS_WORKERS" desc:"The number of concurrent go routines that fetch events from the event queue." introductionVersion:"1.0.0"`

	Scanner         Scanner
	MaxScanSize     string          `yaml:"max-scan-size" env:"ANTIVIRUS_MAX_SCAN_SIZE" desc:"The maximum scan size the virus scanner can handle.0 means unlimited. Usable common abbreviations: [KB, KiB, MB, MiB, GB, GiB, TB, TiB, PB, PiB, EB, EiB], example: 2GB." introductionVersion:"1.0.0"`
	MaxScanSizeMode MaxScanSizeMode `yaml:"max-scan-size-mode" env:"ANTIVIRUS_MAX_SCAN_SIZE_MODE" desc:"Defines the mode of handling files that exceed the maximum scan size. Supported options are: 'skip', which skips files that are bigger than the max scan size, and 'truncate' (default), which only uses the file up to the max size." introductionVersion:"2.1.0"`

	Context context.Context `json:"-" yaml:"-"`

	DebugScanOutcome string `yaml:"-" env:"ANTIVIRUS_DEBUG_SCAN_OUTCOME" desc:"A predefined outcome for virus scanning, FOR DEBUG PURPOSES ONLY! (example values: 'found,infected')" introductionVersion:"1.0.0"`
}

// Service defines the available service configuration.
type Service struct {
	Name string `yaml:"-"`
}

// Log defines the available log configuration.
type Log struct {
	Level  string `mapstructure:"level" env:"OC_LOG_LEVEL;ANTIVIRUS_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Pretty bool   `mapstructure:"pretty" env:"OC_LOG_PRETTY;ANTIVIRUS_LOG_PRETTY" desc:"Activates pretty log output." introductionVersion:"1.0.0"`
	Color  bool   `mapstructure:"color" env:"OC_LOG_COLOR;ANTIVIRUS_LOG_COLOR" desc:"Activates colorized log output." introductionVersion:"1.0.0"`
	File   string `mapstructure:"file" env:"OC_LOG_FILE;ANTIVIRUS_LOG_FILE" desc:"The path to the log file. Activates logging to this file if set." introductionVersion:"1.0.0"`
}

// Debug defines the available debug configuration.
type Debug struct {
	Addr   string `yaml:"addr" env:"ANTIVIRUS_DEBUG_ADDR" desc:"Bind address of the debug server, where metrics, health, config and debug endpoints will be exposed." introductionVersion:"1.0.0"`
	Token  string `yaml:"token" env:"ANTIVIRUS_DEBUG_TOKEN" desc:"Token to secure the metrics endpoint." introductionVersion:"1.0.0"`
	Pprof  bool   `yaml:"pprof" env:"ANTIVIRUS_DEBUG_PPROF" desc:"Enables pprof, which can be used for profiling." introductionVersion:"1.0.0"`
	Zpages bool   `yaml:"zpages" env:"ANTIVIRUS_DEBUG_ZPAGES" desc:"Enables zpages, which can be used for collecting and viewing in-memory traces." introductionVersion:"1.0.0"`
}

// Events combines the configuration options for the event bus.
type Events struct {
	Endpoint             string `yaml:"endpoint" env:"OC_EVENTS_ENDPOINT;ANTIVIRUS_EVENTS_ENDPOINT" desc:"The address of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture." introductionVersion:"1.0.0"`
	Cluster              string `yaml:"cluster" env:"OC_EVENTS_CLUSTER;ANTIVIRUS_EVENTS_CLUSTER" desc:"The clusterID of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture. Mandatory when using NATS as event system." introductionVersion:"1.0.0"`
	TLSInsecure          bool   `yaml:"tls_insecure" env:"OC_INSECURE;OC_EVENTS_TLS_INSECURE;ANTIVIRUS_EVENTS_TLS_INSECURE" desc:"Whether to verify the server TLS certificates." introductionVersion:"1.0.0"`
	TLSRootCACertificate string `yaml:"tls_root_ca_certificate" env:"OC_EVENTS_TLS_ROOT_CA_CERTIFICATE;ANTIVIRUS_EVENTS_TLS_ROOT_CA_CERTIFICATE" desc:"The root CA certificate used to validate the server's TLS certificate. If provided ANTIVIRUS_EVENTS_TLS_INSECURE will be seen as false." introductionVersion:"1.0.0"`
	EnableTLS            bool   `yaml:"enable_tls" env:"OC_EVENTS_ENABLE_TLS;ANTIVIRUS_EVENTS_ENABLE_TLS" desc:"Enable TLS for the connection to the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthUsername         string `yaml:"username" env:"OC_EVENTS_AUTH_USERNAME;ANTIVIRUS_EVENTS_AUTH_USERNAME" desc:"The username to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthPassword         string `yaml:"password" env:"OC_EVENTS_AUTH_PASSWORD;ANTIVIRUS_EVENTS_AUTH_PASSWORD" desc:"The password to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
}

// Scanner provides configuration options for the virus scanner
type Scanner struct {
	Type ScannerType `yaml:"type" env:"ANTIVIRUS_SCANNER_TYPE" desc:"The antivirus scanner to use. Supported values are 'clamav' and 'icap'." introductionVersion:"1.0.0"`

	ClamAV ClamAV // only if Type == clamav
	ICAP   ICAP   // only if Type == icap
}

// ClamAV provides configuration option for clamav
type ClamAV struct {
	Socket  string        `yaml:"socket" env:"ANTIVIRUS_CLAMAV_SOCKET" desc:"The socket clamav is running on. Note the default value is an example which needs adaption according your OS." introductionVersion:"1.0.0"`
	Timeout time.Duration `yaml:"scan_timeout" env:"ANTIVIRUS_CLAMAV_SCAN_TIMEOUT" desc:"Scan timeout for the ClamAV client. Defaults to '5m' (5 minutes). See the Environment Variable Types description for more details." introductionVersion:"2.1.0"`
}

// ICAP provides configuration options for icap
type ICAP struct {
	Timeout time.Duration `yaml:"scan_timeout" env:"ANTIVIRUS_ICAP_SCAN_TIMEOUT" desc:"Scan timeout for the ICAP client. Defaults to '5m' (5 minutes). See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	URL     string        `yaml:"url" env:"ANTIVIRUS_ICAP_URL" desc:"URL of the ICAP server." introductionVersion:"1.0.0"`
	Service string        `yaml:"service" env:"ANTIVIRUS_ICAP_SERVICE" desc:"The name of the ICAP service." introductionVersion:"1.0.0"`
}
