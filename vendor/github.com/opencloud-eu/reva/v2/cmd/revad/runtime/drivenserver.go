package runtime

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/registry"
	"github.com/rs/zerolog"
)

const (
	HTTP = iota
	GRPC
)

// RevaDrivenServer is an interface that defines the methods for starting and stopping reva HTTP/GRPC services.
type RevaDrivenServer interface {
	Start() error
	Stop() error
}

// revaServer is an interface that defines the methods for starting and stopping a reva server.
type revaServer interface {
	Start(ln net.Listener) error
	Stop() error
	GracefulStop() error
	Network() string
	Address() string
}

// sever represents a generic reva server that implements the RevaDrivenServer interface.
type server struct {
	srv                     revaServer
	log                     *zerolog.Logger
	gracefulShutdownTimeout time.Duration
	protocol                string
}

// NewDrivenHTTPServerWithOptions runs a revad server w/o watcher with the given config file and options.
// Use it in cases where you want to run a revad server without the need for a watcher and the os signal handling as a part of another runtime.
// Returns nil if no http server is configured in the config file.
// The GracefulShutdownTimeout set to default 20 seconds and can be overridden in the core config.
// Logging a fatal error and exit with code 1 if the http server cannot be created.
func NewDrivenHTTPServerWithOptions(mainConf map[string]interface{}, opts ...Option) *server {
	if !isEnabledHTTP(mainConf) {
		return nil
	}
	options := newOptions(opts...)
	var srv *server
	var err error
	if srv, err = newServer(HTTP, mainConf, options); err != nil {
		options.Logger.Fatal().Err(err).Msg("failed to create http server")
	}
	return srv

}

// NewDrivenGRPCServerWithOptions runs a revad server w/o watcher with the given config file and options.
// Use it in cases where you want to run a revad server without the need for a watcher and the os signal handling as a part of another runtime.
// Returns nil if no grpc server is configured in the config file.
// The GracefulShutdownTimeout set to default 20 seconds and can be overridden in the core config.
// Logging a fatal error and exit with code 1 if the grpc server cannot be created.
func NewDrivenGRPCServerWithOptions(mainConf map[string]interface{}, opts ...Option) *server {
	if !isEnabledGRPC(mainConf) {
		return nil
	}
	options := newOptions(opts...)
	var srv *server
	var err error
	if srv, err = newServer(GRPC, mainConf, options); err != nil {
		options.Logger.Fatal().Err(err).Msg("failed to create grpc server")
	}
	return srv
}

// Start starts the reva server, listening on the configured address and network.
func (s *server) Start() error {
	if s.srv == nil {
		return fmt.Errorf("reva %s server not initialized", s.protocol)
	}
	ln, err := net.Listen(s.srv.Network(), s.srv.Address())
	if err != nil {
		return err
	}
	if err = s.srv.Start(ln); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			s.log.Error().Err(err).Msg("reva server error")
		}
		return err
	}
	// update logger with transport and address
	logger := s.log.With().Str("network.transport", s.srv.Network()).Str("network.local.address", s.srv.Address()).Logger()
	s.log = &logger
	return nil
}

// Stop gracefully stops the reva server, waiting for the graceful shutdown timeout.
func (s *server) Stop() error {
	if s.srv == nil {
		return nil
	}
	done := make(chan struct{})
	go func() {
		s.log.Info().Msg("gracefully stopping reva server")
		if err := s.srv.GracefulStop(); err != nil {
			s.log.Error().Err(err).Msg("error gracefully stopping reva server")
			err := s.srv.Stop()
			if err != nil {
				s.log.Error().Err(err).Msg("error stopping reva server")
			}
		}
		close(done)
	}()

	select {
	case <-time.After(s.gracefulShutdownTimeout):
		s.log.Info().Msg("graceful shutdown timeout reached. running hard shutdown")
		err := s.srv.Stop()
		if err != nil {
			s.log.Error().Err(err).Msg("error stopping reva server")
		}
		return nil
	case <-done:
		s.log.Info().Msg("reva server gracefully stopped")
		return nil
	}
}

// newServer runs a revad server w/o watcher with the given config file and options.
func newServer(protocol int, mainConf map[string]interface{}, options Options) (*server, error) {
	parseSharedConfOrDie(mainConf["shared"])
	coreConf := parseCoreConfOrDie(mainConf["core"])

	if err := registry.Init(options.Registry); err != nil {
		return nil, err
	}

	srv := &server{}

	// update logger with hostname
	host, _ := os.Hostname()
	logger := options.Logger.With().Str("host.name", host).Logger()
	srv.log = &logger

	// Only initialize tracing if we didn't get a tracer provider.
	if options.TraceProvider == nil {
		srv.log.Debug().Msg("no pre-existing tracer given, initializing tracing")
		options.TraceProvider = initTracing(coreConf)
	}
	initCPUCount(coreConf, srv.log)

	srv.gracefulShutdownTimeout = 20 * time.Second
	if coreConf.GracefulShutdownTimeout > 0 {
		srv.gracefulShutdownTimeout = time.Duration(coreConf.GracefulShutdownTimeout) * time.Second
	}

	switch protocol {
	case HTTP:
		s, err := getHTTPServer(mainConf["http"], srv.log, options.TraceProvider)
		if err != nil {
			return nil, err
		}
		srv.srv = s
		srv.protocol = "http"
		// update logger with protocol
		logger := srv.log.With().Str("protocol", "http").Logger()
		srv.log = &logger
		return srv, nil
	case GRPC:
		s, err := getGRPCServer(mainConf["grpc"], srv.log, options.TraceProvider)
		if err != nil {
			return nil, err
		}
		srv.srv = s
		srv.protocol = "grpc"
		// update logger with protocol
		logger := srv.log.With().Str("protocol", "grpc").Logger()
		srv.log = &logger
		return srv, nil
	}
	return nil, fmt.Errorf("unknown protocol: %d", protocol)
}
