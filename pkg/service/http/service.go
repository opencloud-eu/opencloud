package http

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/broker"
	"github.com/opencloud-eu/opencloud/pkg/registry"

	mhttps "github.com/go-micro/plugins/v4/server/http"
	mtracer "github.com/go-micro/plugins/v4/wrapper/trace/opentelemetry"
	occrypto "github.com/opencloud-eu/opencloud/pkg/crypto"
	"go-micro.dev/v4"
	"go-micro.dev/v4/server"
)

// Service simply wraps the go-micro web service.
type Service struct {
	micro.Service
}

// NewService initializes a new http service.
func NewService(opts ...Option) (Service, error) {
	noopBroker := broker.NoOp{}
	sopts := newOptions(opts...)
	var mServer server.Server
	if sopts.TLSConfig.Enabled {
		var cert tls.Certificate
		var err error
		if sopts.TLSConfig.Cert != "" {
			cert, err = tls.LoadX509KeyPair(sopts.TLSConfig.Cert, sopts.TLSConfig.Key)
			if err != nil {
				sopts.Logger.Error().Err(err).
					Str("cert", sopts.TLSConfig.Cert).
					Str("key", sopts.TLSConfig.Key).
					Msg("error loading server certifcate and key")
				return Service{}, fmt.Errorf("error loading server certificate and key: %w", err)
			}
		} else {
			// Generate a self-signed server certificate on the fly. This requires the clients
			// to connect with InsecureSkipVerify.
			sopts.Logger.Warn().Str("address", sopts.Address).
				Msg("No server certificate configured. Generating a temporary self-signed certificate")
			cert, err = occrypto.GenTempCertForAddr(sopts.Address)
			if err != nil {
				return Service{}, fmt.Errorf("error creating temporary self-signed certificate: %w", err)
			}
		}
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
			NextProtos:   []string{"h2", "http/1.1"},
		}
		mServer = mhttps.NewServer(server.TLSConfig(tlsConfig))
	} else {
		mServer = mhttps.NewServer()
	}

	wopts := []micro.Option{
		micro.Server(mServer),
		micro.Broker(noopBroker),
		micro.Address(sopts.Address),
		micro.Name(strings.Join([]string{sopts.Namespace, sopts.Name}, ".")),
		micro.Version(sopts.Version),
		micro.Context(sopts.Context),
		// TODO: clarify if this is actually used on the go-micro side
		//micro.Flags(sopts.Flags...),
		micro.Registry(registry.GetRegistry()),
		micro.RegisterTTL(registry.GetRegisterTTL()),
		micro.RegisterInterval(registry.GetRegisterInterval()),
		micro.WrapClient(mtracer.NewClientWrapper(
			mtracer.WithTraceProvider(sopts.TraceProvider),
		)),
		micro.WrapHandler(mtracer.NewHandlerWrapper(
			mtracer.WithTraceProvider(sopts.TraceProvider),
		)),
		micro.WrapSubscriber(mtracer.NewSubscriberWrapper(
			mtracer.WithTraceProvider(sopts.TraceProvider),
		)),
	}
	if sopts.TLSConfig.Enabled {
		wopts = append(wopts, micro.Metadata(map[string]string{"use_tls": "true"}))
	}

	return Service{micro.NewService(wopts...)}, nil
}
