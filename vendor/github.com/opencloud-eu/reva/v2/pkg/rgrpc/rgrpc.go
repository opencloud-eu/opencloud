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

package rgrpc

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sort"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/internal/grpc/interceptors/appctx"
	"github.com/opencloud-eu/reva/v2/internal/grpc/interceptors/auth"
	"github.com/opencloud-eu/reva/v2/internal/grpc/interceptors/log"
	"github.com/opencloud-eu/reva/v2/internal/grpc/interceptors/recovery"
	"github.com/opencloud-eu/reva/v2/internal/grpc/interceptors/token"
	"github.com/opencloud-eu/reva/v2/internal/grpc/interceptors/useragent"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	rtrace "github.com/opencloud-eu/reva/v2/pkg/trace"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	mtls "go-micro.dev/v4/util/tls"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// UnaryInterceptors is a map of registered unary grpc interceptors.
var UnaryInterceptors = map[string]NewUnaryInterceptor{}

// StreamInterceptors is a map of registered streaming grpc interceptor
var StreamInterceptors = map[string]NewStreamInterceptor{}

// NewUnaryInterceptor is the type that unary interceptors need to register.
type NewUnaryInterceptor func(m map[string]interface{}) (grpc.UnaryServerInterceptor, int, error)

// NewStreamInterceptor is the type that stream interceptors need to register.
type NewStreamInterceptor func(m map[string]interface{}) (grpc.StreamServerInterceptor, int, error)

// RegisterUnaryInterceptor registers a new unary interceptor.
func RegisterUnaryInterceptor(name string, newFunc NewUnaryInterceptor) {
	UnaryInterceptors[name] = newFunc
}

// RegisterStreamInterceptor registers a new stream interceptor.
func RegisterStreamInterceptor(name string, newFunc NewStreamInterceptor) {
	StreamInterceptors[name] = newFunc
}

// Services is a map of service name and its new function.
var Services = map[string]NewService{}

// Register registers a new gRPC service with name and new function.
func Register(name string, newFunc NewService) {
	Services[name] = newFunc
}

// NewService is the function that gRPC services need to register at init time.
// It returns an io.Closer to close the service and a list of service endpoints that need to be unprotected.
type NewService func(conf map[string]interface{}, ss *grpc.Server, log *zerolog.Logger) (Service, error)

// Service represents a grpc service.
type Service interface {
	Register(ss *grpc.Server)
	io.Closer
	UnprotectedEndpoints() []string
}

type unaryInterceptorTriple struct {
	Name        string
	Priority    int
	Interceptor grpc.UnaryServerInterceptor
}

type streamInterceptorTriple struct {
	Name        string
	Priority    int
	Interceptor grpc.StreamServerInterceptor
}

type tlsSettings struct {
	Enabled         bool   `mapstructure:"enabled"`
	CertificateFile string `mapstructure:"certificate"`
	KeyFile         string `mapstructure:"key"`
	tlsConfig       *tls.Config
}

type config struct {
	Network          string                            `mapstructure:"network"`
	Address          string                            `mapstructure:"address"`
	TLSSettings      tlsSettings                       `mapstructure:"tls_settings"`
	ShutdownDeadline int                               `mapstructure:"shutdown_deadline"`
	Services         map[string]map[string]interface{} `mapstructure:"services"`
	Interceptors     map[string]map[string]interface{} `mapstructure:"interceptors"`
	EnableReflection bool                              `mapstructure:"enable_reflection"`
}

func (c *config) init() {
	if c.Network == "" {
		c.Network = "tcp"
	}

	if c.Address == "" {
		c.Address = sharedconf.GetGatewaySVC("0.0.0.0:19000")
	}
}

// Server is a gRPC server.
type Server struct {
	s              *grpc.Server
	conf           *config
	listener       net.Listener
	log            zerolog.Logger
	tracerProvider trace.TracerProvider
	services       map[string]Service
}

// NewServer returns a new Server.
func NewServer(m interface{}, log zerolog.Logger, tp trace.TracerProvider) (*Server, error) {
	var err error
	conf := &config{}
	if err := mapstructure.Decode(m, conf); err != nil {
		return nil, err
	}

	conf.init()

	if conf.TLSSettings.Enabled {
		var cert tls.Certificate
		switch {
		case conf.TLSSettings.CertificateFile == "" && conf.TLSSettings.KeyFile == "":
			// Generate a self-signed server certificate on the fly. This requires the clients
			// to connect with InsecureSkipVerify.
			subj := []string{conf.Address}
			if host, _, err := net.SplitHostPort(conf.Address); err == nil && host != "" {
				subj = []string{host}
			}

			log.Warn().Str("address", conf.Address).Str("network", conf.Network).
				Msg("No server certificate configured. Generating a temporary self-signed certificate")

			cert, err = mtls.Certificate(subj...)
			if err != nil {
				return nil, err
			}
		default:
			cert, err = tls.LoadX509KeyPair(
				conf.TLSSettings.CertificateFile,
				conf.TLSSettings.KeyFile,
			)
			if err != nil {
				return nil, err
			}
		}
		conf.TLSSettings.tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
	}

	server := &Server{conf: conf, log: log, tracerProvider: tp, services: map[string]Service{}}

	return server, nil
}

// Start starts the server.
func (s *Server) Start(ln net.Listener) error {
	if err := s.registerServices(); err != nil {
		err = errors.Wrap(err, "unable to register services")
		return err
	}

	s.listener = ln
	s.log.Info().Msgf("grpc server listening at %s:%s", s.Network(), s.Address())
	err := s.s.Serve(s.listener)
	if err != nil {
		err = errors.Wrap(err, "serve failed")
		return err
	}
	return nil
}

func (s *Server) isInterceptorEnabled(name string) bool {
	for k := range s.conf.Interceptors {
		if k == name {
			return true
		}
	}
	return false
}

func (s *Server) isServiceEnabled(svcName string) bool {
	for key := range Services {
		if key == svcName {
			return true
		}
	}
	return false
}

func (s *Server) registerServices() error {
	for svcName := range s.conf.Services {
		if s.isServiceEnabled(svcName) {
			newFunc := Services[svcName]
			svc, err := newFunc(s.conf.Services[svcName], s.s, &s.log)
			if err != nil {
				return errors.Wrapf(err, "rgrpc: grpc service %s could not be started,", svcName)
			}
			s.services[svcName] = svc
			s.log.Info().Msgf("rgrpc: grpc service enabled: %s", svcName)
		} else {
			message := fmt.Sprintf("rgrpc: grpc service %s does not exist", svcName)
			return errors.New(message)
		}
	}

	// obtain list of unprotected endpoints
	unprotected := []string{}
	for _, svc := range s.services {
		unprotected = append(unprotected, svc.UnprotectedEndpoints()...)
	}

	opts, err := s.getInterceptors(unprotected)
	if err != nil {
		return err
	}

	if s.conf.TLSSettings.tlsConfig != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(s.conf.TLSSettings.tlsConfig)))
	}
	opts = append(opts, grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionAge: GetMaxConnectionAge(), // this forces clients to reconnect after 30 seconds, triggering a new DNS lookup to pick up new IPs
	}))

	grpcServer := grpc.NewServer(opts...)

	for _, svc := range s.services {
		svc.Register(grpcServer)
	}

	if s.conf.EnableReflection {
		s.log.Info().Msg("rgrpc: grpc server reflection enabled")
		reflection.Register(grpcServer)
	}

	s.s = grpcServer

	return nil
}

// TODO(labkode): make closing with deadline.
func (s *Server) cleanupServices() {
	for name, svc := range s.services {
		if err := svc.Close(); err != nil {
			s.log.Error().Err(err).Msgf("error closing service %q", name)
		} else {
			s.log.Info().Msgf("service %q correctly closed", name)
		}
	}
}

// Stop stops the server.
func (s *Server) Stop() error {
	s.s.Stop()
	s.cleanupServices()
	return nil
}

// GracefulStop gracefully stops the server.
func (s *Server) GracefulStop() error {
	s.s.GracefulStop()
	s.cleanupServices()
	return nil
}

// Network returns the network type.
func (s *Server) Network() string {
	return s.conf.Network
}

// Address returns the network address.
func (s *Server) Address() string {
	return s.conf.Address
}

func (s *Server) getInterceptors(unprotected []string) ([]grpc.ServerOption, error) {
	unaryTriples := []*unaryInterceptorTriple{}
	for name, newFunc := range UnaryInterceptors {
		if s.isInterceptorEnabled(name) {
			inter, prio, err := newFunc(s.conf.Interceptors[name])
			if err != nil {
				err = errors.Wrapf(err, "rgrpc: error creating unary interceptor: %s,", name)
				return nil, err
			}
			triple := &unaryInterceptorTriple{
				Name:        name,
				Priority:    prio,
				Interceptor: inter,
			}
			unaryTriples = append(unaryTriples, triple)
		}
	}

	// sort unary triples
	sort.SliceStable(unaryTriples, func(i, j int) bool {
		return unaryTriples[i].Priority < unaryTriples[j].Priority
	})

	authUnary, err := auth.NewUnary(s.conf.Interceptors["auth"], unprotected, s.tracerProvider)
	if err != nil {
		return nil, errors.Wrap(err, "rgrpc: error creating unary auth interceptor")
	}

	unaryInterceptors := []grpc.UnaryServerInterceptor{
		appctx.NewUnary(s.log, s.tracerProvider),
		token.NewUnary(),
		useragent.NewUnary(),
		log.NewUnary(),
		recovery.NewUnary(),
		authUnary,
	}

	for _, t := range unaryTriples {
		unaryInterceptors = append(unaryInterceptors, t.Interceptor)
		s.log.Info().Msgf("rgrpc: chaining grpc unary interceptor %s with priority %d", t.Name, t.Priority)
	}

	unaryChain := grpc_middleware.ChainUnaryServer(unaryInterceptors...)

	streamTriples := []*streamInterceptorTriple{}
	for name, newFunc := range StreamInterceptors {
		if s.isInterceptorEnabled(name) {
			inter, prio, err := newFunc(s.conf.Interceptors[name])
			if err != nil {
				err = errors.Wrapf(err, "rgrpc: error creating streaming interceptor: %s,", name)
				return nil, err
			}
			triple := &streamInterceptorTriple{
				Name:        name,
				Priority:    prio,
				Interceptor: inter,
			}
			streamTriples = append(streamTriples, triple)
		}
	}
	// sort stream triples
	sort.SliceStable(streamTriples, func(i, j int) bool {
		return streamTriples[i].Priority < streamTriples[j].Priority
	})

	authStream, err := auth.NewStream(s.conf.Interceptors["auth"], unprotected, s.tracerProvider)
	if err != nil {
		return nil, errors.Wrap(err, "rgrpc: error creating stream auth interceptor")
	}

	streamInterceptors := []grpc.StreamServerInterceptor{
		appctx.NewStream(s.log, s.tracerProvider),
		token.NewStream(),
		useragent.NewStream(),
		log.NewStream(),
		recovery.NewStream(),
		authStream,
	}

	for _, t := range streamTriples {
		streamInterceptors = append(streamInterceptors, t.Interceptor)
		s.log.Info().Msgf("rgrpc: chaining grpc streaming interceptor %s with priority %d", t.Name, t.Priority)
	}
	streamChain := grpc_middleware.ChainStreamServer(streamInterceptors...)

	opts := []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler(
			otelgrpc.WithTracerProvider(s.tracerProvider),
			otelgrpc.WithPropagators(rtrace.Propagator))),
		grpc.UnaryInterceptor(unaryChain),
		grpc.StreamInterceptor(streamChain),
	}

	return opts, nil
}
