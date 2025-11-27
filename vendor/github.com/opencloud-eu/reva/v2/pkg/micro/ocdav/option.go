// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package ocdav

import (
	"context"
	"crypto/tls"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/config"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storage/favorite"
	"github.com/rs/zerolog"
	"go-micro.dev/v4/broker"
	"go.opentelemetry.io/otel/trace"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	TLSConfig *tls.Config
	Broker    broker.Broker
	Address   string
	Logger    zerolog.Logger
	Context   context.Context
	// Metrics   *metrics.Metrics
	// Flags     []cli.Flag
	Name      string
	JWTSecret string

	FavoriteManager favorite.Manager
	GatewaySelector pool.Selectable[gateway.GatewayAPIClient]

	TracesExporter string

	TraceProvider trace.TracerProvider

	MetricsEnabled   bool
	MetricsNamespace string
	MetricsSubsystem string

	// ocdav.* is internal so we need to set config options individually
	config             config.Config
	lockSystem         ocdav.LockSystem
	AllowCredentials   bool
	AllowedOrigins     []string
	AllowedHeaders     []string
	AllowedMethods     []string
	AllowDepthInfinity bool

	RegisterTTL      time.Duration
	RegisterInterval time.Duration
}

// newOptions initializes the available default options.
func newOptions(opts ...Option) Options {
	opt := Options{}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// TLSConfig provides a function to set the TLSConfig option.
func TLSConfig(config *tls.Config) Option {
	return func(o *Options) {
		o.TLSConfig = config
	}
}

// Broker provides a function to set the Broker option.
func Broker(b broker.Broker) Option {
	return func(o *Options) {
		o.Broker = b
	}
}

// Address provides a function to set the address option.
func Address(val string) Option {
	return func(o *Options) {
		o.Address = val
	}
}

func AllowDepthInfinity(val bool) Option {
	return func(o *Options) {
		o.AllowDepthInfinity = val
	}
}

// JWTSecret provides a function to set the jwt secret option.
func JWTSecret(s string) Option {
	return func(o *Options) {
		o.JWTSecret = s
	}
}

// MachineAuthAPIKey provides a function to set the machine auth api key option.
func MachineAuthAPIKey(s string) Option {
	return func(o *Options) {
		o.config.MachineAuthAPIKey = s
	}
}

// Context provides a function to set the context option.
func Context(val context.Context) Option {
	return func(o *Options) {
		o.Context = val
	}
}

// Logger provides a function to set the logger option.
func Logger(val zerolog.Logger) Option {
	return func(o *Options) {
		o.Logger = val
	}
}

// Name provides a function to set the Name option.
func Name(val string) Option {
	return func(o *Options) {
		o.Name = val
	}
}

// Prefix provides a function to set the prefix config option.
func Prefix(val string) Option {
	return func(o *Options) {
		o.config.Prefix = val
	}
}

// FilesNamespace provides a function to set the FilesNamespace config option.
func FilesNamespace(val string) Option {
	return func(o *Options) {
		o.config.FilesNamespace = val
	}
}

// WebdavNamespace provides a function to set the WebdavNamespace config option.
func WebdavNamespace(val string) Option {
	return func(o *Options) {
		o.config.WebdavNamespace = val
	}
}

// SharesNamespace provides a function to set the SharesNamespace config option.
func SharesNamespace(val string) Option {
	return func(o *Options) {
		o.config.SharesNamespace = val
	}
}

// OCMNamespace provides a function to set the OCMNamespace config option.
func OCMNamespace(val string) Option {
	return func(o *Options) {
		o.config.OCMNamespace = val
	}
}

// GatewaySvc provides a function to set the GatewaySvc config option.
func GatewaySvc(val string) Option {
	return func(o *Options) {
		o.config.GatewaySvc = val
	}
}

// Timeout provides a function to set the Timeout config option.
func Timeout(val int64) Option {
	return func(o *Options) {
		o.config.Timeout = val
	}
}

// Insecure provides a function to set the Insecure config option.
func Insecure(val bool) Option {
	return func(o *Options) {
		o.config.Insecure = val
	}
}

// PublicURL provides a function to set the PublicURL config option.
func PublicURL(val string) Option {
	return func(o *Options) {
		o.config.PublicURL = val
	}
}

// FavoriteManager provides a function to set the FavoriteManager option.
func FavoriteManager(val favorite.Manager) Option {
	return func(o *Options) {
		o.FavoriteManager = val
	}
}

// GatewaySelector provides a function to set the GatewaySelector option.
func GatewaySelector(val pool.Selectable[gateway.GatewayAPIClient]) Option {
	return func(o *Options) {
		o.GatewaySelector = val
	}
}

// LockSystem provides a function to set the LockSystem option.
func LockSystem(val ocdav.LockSystem) Option {
	return func(o *Options) {
		o.lockSystem = val
	}
}

// WithTracesExporter option
func WithTracesExporter(exporter string) Option {
	return func(o *Options) {
		o.TracesExporter = exporter
	}
}

// WithTracingExporter option
// Deprecated: unused
func WithTracingExporter(exporter string) Option {
	return func(o *Options) {}
}

// WithTraceProvider option
func WithTraceProvider(provider trace.TracerProvider) Option {
	return func(o *Options) {
		o.TraceProvider = provider
	}
}

// Version provides a function to set the Version config option.
func Version(val string) Option {
	return func(o *Options) {
		o.config.Version = val
	}
}

// VersionString provides a function to set the VersionString config option.
func VersionString(val string) Option {
	return func(o *Options) {
		o.config.VersionString = val
	}
}

// Edition provides a function to set the Edition config option.
func Edition(val string) Option {
	return func(o *Options) {
		o.config.Edition = val
	}
}

// Product provides a function to set the Product config option.
func Product(val string) Option {
	return func(o *Options) {
		o.config.Product = val
	}
}

// ProductName provides a function to set the ProductName config option.
func ProductName(val string) Option {
	return func(o *Options) {
		o.config.ProductName = val
	}
}

// ProductVersion provides a function to set the ProductVersion config option.
func ProductVersion(val string) Option {
	return func(o *Options) {
		o.config.ProductVersion = val
	}
}

// MetricsEnabled provides a function to set the MetricsEnabled config option.
func MetricsEnabled(val bool) Option {
	return func(o *Options) {
		o.MetricsEnabled = val
	}
}

// MetricsNamespace provides a function to set the MetricsNamespace config option.
func MetricsNamespace(val string) Option {
	return func(o *Options) {
		o.MetricsNamespace = val
	}
}

// MetricsSubsystem provides a function to set the MetricsSubsystem config option.
func MetricsSubsystem(val string) Option {
	return func(o *Options) {
		o.MetricsSubsystem = val
	}
}

// AllowCredentials provides a function to set the AllowCredentials option.
func AllowCredentials(val bool) Option {
	return func(o *Options) {
		o.AllowCredentials = val
	}
}

// AllowedOrigins provides a function to set the AllowedOrigins option.
func AllowedOrigins(val []string) Option {
	return func(o *Options) {
		o.AllowedOrigins = val
	}
}

// AllowedMethods provides a function to set the AllowedMethods option.
func AllowedMethods(val []string) Option {
	return func(o *Options) {
		o.AllowedMethods = val
	}
}

// AllowedHeaders provides a function to set the AllowedHeaders option.
func AllowedHeaders(val []string) Option {
	return func(o *Options) {
		o.AllowedHeaders = val
	}
}

// ItemNameInvalidChars provides a function to set forbidden characters in file or folder names
func ItemNameInvalidChars(chars []string) Option {
	return func(o *Options) {
		o.config.NameValidation.InvalidChars = chars
	}
}

// ItemNameMaxLength provides a function to set the maximum length of a file or folder name
func ItemNameMaxLength(i int) Option {
	return func(o *Options) {
		o.config.NameValidation.MaxLength = i
	}
}

// RegisterTTL provides a function to set the RegisterTTL option.
func RegisterTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.RegisterTTL = ttl
	}
}

// RegisterInterval provides a function to set the RegisterInterval option.
func RegisterInterval(interval time.Duration) Option {
	return func(o *Options) {
		o.RegisterInterval = interval
	}
}

// URLSigningSharedSecret provides a function to set the URLSigningSharedSecret config option.
func URLSigningSharedSecret(secret string) Option {
	return func(o *Options) {
		o.config.URLSigningSharedSecret = secret
	}
}
