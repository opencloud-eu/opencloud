package runner

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	ogrpc "github.com/opencloud-eu/opencloud/pkg/service/grpc"
	ohttp "github.com/opencloud-eu/opencloud/pkg/service/http"
	"google.golang.org/grpc"
)

// NewGoMicroGrpcServerRunner creates a new runner based on the provided go-micro's
// GRPC service. The service is expected to be created via
// "github.com/owncloud/ocis/v2/ocis-pkg/service/grpc".NewService(...) function
//
// The runner will behave as described:
// * The task is to start a server and listen for connections. If the server
// can't start, the task will finish with that error.
// * The stopper will call the server's stop method and send the result to
// the task.
// * The stopper will run asynchronously because the stop method could take a
// while and we don't want to block
func NewGoMicroGrpcServerRunner(name string, server ogrpc.Service, opts ...Option) *Runner {
	httpCh := make(chan error, 1)
	r := New(name, func() error {
		// start the server and return if it fails
		if err := server.Server().Start(); err != nil {
			return err
		}
		return <-httpCh // wait for the result
	}, func() {
		// stop implies deregistering and waiting for request to finish,
		// so don't block
		go func() {
			httpCh <- server.Server().Stop() // stop and send result through channel
			close(httpCh)
		}()
	}, opts...)
	return r
}

// NewGoMicroHttpServerRunner creates a new runner based on the provided go-micro's
// HTTP service. The service is expected to be created via
// "github.com/owncloud/ocis/v2/ocis-pkg/service/http".NewService(...) function
//
// The runner will behave as described:
// * The task is to start a server and listen for connections. If the server
// can't start, the task will finish with that error.
// * The stopper will call the server's stop method and send the result to
// the task.
// * The stopper will run asynchronously because the stop method could take a
// while and we don't want to block
func NewGoMicroHttpServerRunner(name string, server ohttp.Service, opts ...Option) *Runner {
	httpCh := make(chan error, 1)
	r := New(name, func() error {
		// start the server and return if it fails
		if err := server.Server().Start(); err != nil {
			return err
		}
		return <-httpCh // wait for the result
	}, func() {
		// stop implies deregistering and waiting for request to finish,
		// so don't block
		go func() {
			httpCh <- server.Server().Stop() // stop and send result through channel
			close(httpCh)
		}()
	}, opts...)
	return r
}

// NewGolangHttpServerRunner creates a new runner based on the provided HTTP server.
// The HTTP server is expected to be created via
// "github.com/owncloud/ocis/v2/ocis-pkg/service/debug".NewService(...) function
// and it's expected to be a regular golang HTTP server
//
// The runner will behave as described:
// * The task starts a server and listen for connections. If the server
// can't start, the task will finish with that error. If the server is shutdown
// the task will wait for the shutdown to return that result (task won't finish
// immediately, but wait until shutdown returns)
// * The stopper will call the server's shutdown method and send the result to
// the task. The stopper will wait up to 5 secs for the shutdown.
// * The stopper will run asynchronously because the shutdown could take a
// while and we don't want to block
func NewGolangHttpServerRunner(name string, server *http.Server, opts ...Option) *Runner {
	debugCh := make(chan error, 1)
	r := New(name, func() error {
		// start listening and return if the error is NOT ErrServerClosed.
		// ListenAndServe will always return a non-nil error.
		// We need to wait and get the result of the Shutdown call.
		// App shouldn't exit until Shutdown has returned.
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		// wait for the shutdown and return the result
		return <-debugCh
	}, func() {
		// Since Shutdown might take some time, don't block
		go func() {
			// give 5 secs for the shutdown to finish
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			debugCh <- server.Shutdown(shutdownCtx)
			close(debugCh)
		}()
	}, opts...)

	return r
}

// NewGolangGrpcServerRunner creates a new runner based on the provided GRPC
// server. The GRPC server is expected to be a regular golang GRPC server,
// created via "google.golang.org/grpc".NewServer(...)
// A listener also needs to be provided for the server to listen there.
//
// The runner will just start the GRPC server in the listener, and the server
// will be gracefully stopped when interrupted
func NewGolangGrpcServerRunner(name string, server *grpc.Server, listener net.Listener, opts ...Option) *Runner {
	r := New(name, func() error {
		return server.Serve(listener)
	}, func() {
		// Since GracefulStop might take some time, don't block
		go func() {
			server.GracefulStop()
		}()
	}, opts...)

	return r
}
