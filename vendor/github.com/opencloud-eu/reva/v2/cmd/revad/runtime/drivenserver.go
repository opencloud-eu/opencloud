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
func NewDrivenHTTPServerWithOptions(mainConf map[string]interface{}, opts ...Option) RevaDrivenServer {
	if !isEnabledHTTP(mainConf) {
		return nil
	}
	options := newOptions(opts...)
	if srv := newServer(HTTP, mainConf, options); srv != nil {
		return srv
	}
	options.Logger.Fatal().Msg("nothing to do, no http enabled_services declared in config")
	return nil
}

// NewDrivenGRPCServerWithOptions runs a revad server w/o watcher with the given config file and options.
// Use it in cases where you want to run a revad server without the need for a watcher and the os signal handling as a part of another runtime.
// Returns nil if no grpc server is configured in the config file.
// The GracefulShutdownTimeout set to default 20 seconds and can be overridden in the core config.
// Logging a fatal error and exit with code 1 if the grpc server cannot be created.
func NewDrivenGRPCServerWithOptions(mainConf map[string]interface{}, opts ...Option) RevaDrivenServer {
	if !isEnabledGRPC(mainConf) {
		return nil
	}
	options := newOptions(opts...)
	if srv := newServer(GRPC, mainConf, options); srv != nil {
		return srv
	}
	options.Logger.Fatal().Msg("nothing to do, no grpc enabled_services declared in config")
	return nil
}

// Start starts the reva server, listening on the configured address and network.
func (s *server) Start() error {
	if s.srv == nil {
		err := fmt.Errorf("reva %s server not initialized", s.protocol)
		s.log.Fatal().Err(err).Send()
		return err
	}
	ln, err := net.Listen(s.srv.Network(), s.srv.Address())
	if err != nil {
		s.log.Fatal().Err(err).Send()
		return err
	}
	if err = s.srv.Start(ln); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			s.log.Error().Err(err).Msgf("reva %s server error", s.protocol)
		}
		return err
	}
	return nil
}

// Stop gracefully stops the reva server, waiting for the graceful shutdown timeout.
func (s *server) Stop() error {
	if s.srv == nil {
		return nil
	}
	done := make(chan struct{})
	go func() {
		s.log.Info().Msgf("gracefully stopping %s:%s reva %s server", s.srv.Network(), s.srv.Address(), s.protocol)
		if err := s.srv.GracefulStop(); err != nil {
			s.log.Error().Err(err).Msgf("error gracefully stopping reva %s server", s.protocol)
			s.srv.Stop()
		}
		close(done)
	}()

	select {
	case <-time.After(s.gracefulShutdownTimeout):
		s.log.Info().Msg("graceful shutdown timeout reached. running hard shutdown")
		err := s.srv.Stop()
		if err != nil {
			s.log.Error().Err(err).Msgf("error stopping reva %s server", s.protocol)
		}
		return nil
	case <-done:
		s.log.Info().Msgf("reva %s server gracefully stopped", s.protocol)
		return nil
	}
}

// newServer runs a revad server w/o watcher with the given config file and options.
func newServer(protocol int, mainConf map[string]interface{}, options Options) RevaDrivenServer {
	parseSharedConfOrDie(mainConf["shared"])
	coreConf := parseCoreConfOrDie(mainConf["core"])
	log := options.Logger

	if err := registry.Init(options.Registry); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize registry client")
		return nil
	}

	host, _ := os.Hostname()
	log.Info().Msgf("host info: %s", host)

	// Only initialize tracing if we didn't get a tracer provider.
	if options.TraceProvider == nil {
		log.Debug().Msg("no pre-existing tracer given, initializing tracing")
		options.TraceProvider = initTracing(coreConf)
	}
	initCPUCount(coreConf, log)

	gracefulShutdownTimeout := 20 * time.Second
	if coreConf.GracefulShutdownTimeout > 0 {
		gracefulShutdownTimeout = time.Duration(coreConf.GracefulShutdownTimeout) * time.Second
	}

	srv := &server{
		log:                     options.Logger,
		gracefulShutdownTimeout: gracefulShutdownTimeout,
	}
	switch protocol {
	case HTTP:
		s, err := getHTTPServer(mainConf["http"], options.Logger, options.TraceProvider)
		if err != nil {
			options.Logger.Fatal().Err(err).Msg("error creating http server")
			return nil
		}
		srv.srv = s
		srv.protocol = "http"
		return srv
	case GRPC:
		s, err := getGRPCServer(mainConf["grpc"], options.Logger, options.TraceProvider)
		if err != nil {
			options.Logger.Fatal().Err(err).Msg("error creating grpc server")
			return nil
		}
		srv.srv = s
		srv.protocol = "grpc"
		return srv
	}
	return nil
}
