package config

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

type Config struct {
	Commons  *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service
	Service  Service         `yaml:"-"`
	LogLevel string          `yaml:"loglevel" env:"OC_LOG_LEVEL;APP_PROVIDER_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Debug    Debug           `yaml:"debug"`

	GRPC GRPCConfig `yaml:"grpc"`

	TokenManager *TokenManager `yaml:"token_manager"`
	Reva         *shared.Reva  `yaml:"reva"`

	ExternalAddr string  `yaml:"external_addr" env:"APP_PROVIDER_EXTERNAL_ADDR" desc:"Address of the app provider, where the GATEWAY service can reach it." introductionVersion:"1.0.0"`
	Driver       string  `yaml:"driver" env:"APP_PROVIDER_DRIVER" desc:"Driver, the APP PROVIDER services uses. Only 'wopi' is supported as of now." introductionVersion:"1.0.0"`
	Drivers      Drivers `yaml:"drivers"`

	Context context.Context `yaml:"-"`
}

type Debug struct {
	Addr   string `yaml:"addr" env:"APP_PROVIDER_DEBUG_ADDR" desc:"Bind address of the debug server, where metrics, health, config and debug endpoints will be exposed." introductionVersion:"1.0.0"`
	Token  string `yaml:"token" env:"APP_PROVIDER_DEBUG_TOKEN" desc:"Token to secure the metrics endpoint" introductionVersion:"1.0.0"`
	Pprof  bool   `yaml:"pprof" env:"APP_PROVIDER_DEBUG_PPROF" desc:"Enables pprof, which can be used for profiling" introductionVersion:"1.0.0"`
	Zpages bool   `yaml:"zpages" env:"APP_PROVIDER_DEBUG_ZPAGES" desc:"Enables zpages, which can  be used for collecting and viewing traces in-memory." introductionVersion:"1.0.0"`
}

type Service struct {
	Name string `yaml:"name" env:"APP_PROVIDER_SERVICE_NAME" desc:"The name of the service. This needs to be changed when using more than one app provider. Each app provider configured needs to be identified by a unique service name. Possible examples are: 'app-provider-collabora', 'app-provider-onlyoffice', 'app-provider-office365'." introductionVersion:"1.0.0"`
}

type GRPCConfig struct {
	Addr      string                 `yaml:"addr" env:"APP_PROVIDER_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"1.0.0"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
	Namespace string                 `yaml:"-"`
	Protocol  string                 `yaml:"protocol" env:"OC_GRPC_PROTOCOL;APP_PROVIDER_GRPC_PROTOCOL" desc:"The transport protocol of the GPRC service." introductionVersion:"1.0.0"`
}

type Drivers struct {
	WOPI WOPIDriver `yaml:"wopi" desc:"Driver for the CS3org WOPI server"`
}

type WOPIDriver struct {
	AppAPIKey                 string `yaml:"app_api_key" env:"APP_PROVIDER_WOPI_APP_API_KEY" desc:"API key for the wopi app." introductionVersion:"1.0.0"`
	AppDesktopOnly            bool   `yaml:"app_desktop_only" env:"APP_PROVIDER_WOPI_APP_DESKTOP_ONLY" desc:"Offer this app only on desktop." introductionVersion:"1.0.0"`
	AppIconURI                string `yaml:"app_icon_uri" env:"APP_PROVIDER_WOPI_APP_ICON_URI" desc:"URI to an app icon to be used by clients." introductionVersion:"1.0.0"`
	AppInternalURL            string `yaml:"app_internal_url" env:"APP_PROVIDER_WOPI_APP_INTERNAL_URL" desc:"Internal URL to the app, like in your DMZ." introductionVersion:"1.0.0"`
	AppName                   string `yaml:"app_name" env:"APP_PROVIDER_WOPI_APP_NAME" desc:"Human readable app name." introductionVersion:"1.0.0"`
	AppURL                    string `yaml:"app_url" env:"APP_PROVIDER_WOPI_APP_URL" desc:"URL for end users to access the app." introductionVersion:"1.0.0"`
	AppDisableChat            bool   `yaml:"app_disable_chat" env:"APP_PROVIDER_WOPI_DISABLE_CHAT;OC_WOPI_DISABLE_CHAT" desc:"Disable the chat functionality of the office app." introductionVersion:"1.0.0"`
	Insecure                  bool   `yaml:"insecure" env:"APP_PROVIDER_WOPI_INSECURE" desc:"Disable TLS certificate validation for requests to the WOPI server and the web office application. Do not set this in production environments." introductionVersion:"1.0.0"`
	IopSecret                 string `yaml:"wopi_server_iop_secret" env:"APP_PROVIDER_WOPI_WOPI_SERVER_IOP_SECRET" desc:"Shared secret of the CS3org WOPI server." introductionVersion:"1.0.0"`
	WopiURL                   string `yaml:"wopi_server_external_url" env:"APP_PROVIDER_WOPI_WOPI_SERVER_EXTERNAL_URL" desc:"External url of the CS3org WOPI server." introductionVersion:"1.0.0"`
	WopiFolderURLBaseURL      string `yaml:"wopi_folder_url_base_url" env:"OC_URL;APP_PROVIDER_WOPI_FOLDER_URL_BASE_URL" desc:"Base url to navigate back from the app to the containing folder in the file list." introductionVersion:"1.0.0"`
	WopiFolderURLPathTemplate string `yaml:"wopi_folder_url_path_template" env:"APP_PROVIDER_WOPI_FOLDER_URL_PATH_TEMPLATE" desc:"Path template to navigate back from the app to the containing folder in the file list. Supported template variables are {{.ResourceInfo.ResourceID}}, {{.ResourceInfo.Mtime.Seconds}}, {{.ResourceInfo.Name}}, {{.ResourceInfo.Path}}, {{.ResourceInfo.Type}}, {{.ResourceInfo.Id.SpaceId}}, {{.ResourceInfo.Id.StorageId}}, {{.ResourceInfo.Id.OpaqueId}}, {{.ResourceInfo.MimeType}}" introductionVersion:"1.0.0"`
}
