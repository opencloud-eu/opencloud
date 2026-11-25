package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"

	mgrpcc "github.com/go-micro/plugins/v4/client/grpc"
	mtracer "github.com/go-micro/plugins/v4/wrapper/trace/opentelemetry"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"go-micro.dev/v4/client"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// ClientOptions represent options (e.g. tls settings) for the grpc clients
type ClientOptions struct {
	tlsMode     string
	caCert      string
	tlsInsecure bool
	tp          trace.TracerProvider
}

// Option is used to pass client options
type ClientOption func(opts *ClientOptions)

// WithTLSMode allows setting the TLSMode option for grpc clients
func WithTLSMode(v string) ClientOption {
	return func(o *ClientOptions) {
		o.tlsMode = v
	}
}

// WithTLSCACert allows setting the CA Certificate for grpc clients
func WithTLSCACert(v string) ClientOption {
	return func(o *ClientOptions) {
		o.caCert = v
	}
}

// WithTLSInsecure allows the TLS mode 'insecure' for grpc clients, which
// skips the verification of the server certificate
func WithTLSInsecure(v bool) ClientOption {
	return func(o *ClientOptions) {
		o.tlsInsecure = v
	}
}

// WithTraceProvider allows to set the trace Provider for grpc clients
func WithTraceProvider(tp trace.TracerProvider) ClientOption {
	return func(o *ClientOptions) {
		if tp != nil {
			o.tp = tp
		} else {
			o.tp = noop.NewTracerProvider()
		}
	}
}

func GetClientOptions(t *shared.GRPCClientTLS) []ClientOption {
	opts := []ClientOption{
		WithTLSMode(t.Mode),
		WithTLSCACert(t.CACert),
		WithTLSInsecure(t.Insecure),
	}
	return opts
}

func NewClient(opts ...ClientOption) (client.Client, error) {
	var options ClientOptions
	for _, opt := range opts {
		opt(&options)
	}

	reg := registry.GetRegistry()
	var tlsConfig *tls.Config
	cOpts := []client.Option{
		client.Registry(reg),
		client.Wrap(mtracer.NewClientWrapper(
			mtracer.WithTraceProvider(options.tp),
		)),
	}
	switch options.tlsMode {
	case "insecure":
		// Skipping the server certificate check is only meant for development
		// setups, so it has to be allowed explicitly.
		if !options.tlsInsecure {
			return nil, errors.New("grpc client TLS mode 'insecure' disables server certificate verification and requires OC_INSECURE=true; use OC_GRPC_CLIENT_TLS_MODE=on with a trusted certificate otherwise")
		}
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
		cOpts = append(cOpts, mgrpcc.AuthTLS(tlsConfig))
	case "on":
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		// Note: If caCert is empty we use the system's default set of trusted CAs
		if options.caCert != "" {
			certs := x509.NewCertPool()
			pemData, err := os.ReadFile(options.caCert)
			if err != nil {
				return nil, err
			}
			if !certs.AppendCertsFromPEM(pemData) {
				return nil, errors.New("could not initialize client, adding CA cert failed")
			}
			tlsConfig.RootCAs = certs
		}
		cOpts = append(cOpts, mgrpcc.AuthTLS(tlsConfig))
		// case "off":
		// default:
	}

	return mgrpcc.NewClient(cOpts...), nil
}
