package syncs

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// WaitCloseSignals for some huang program.
//
// Usage:
//
//	// do something. eg: start a http server
//
//	syncs.WaitCloseSignals(func(sig os.Signal) {
//		// do something on shutdown. eg: close db, flush logs
//	})
func WaitCloseSignals(onClose func(sig os.Signal), sigCh ...chan os.Signal) {
	var signals chan os.Signal
	if len(sigCh) > 0 && sigCh[0] != nil {
		signals = sigCh[0]
	} else {
		signals = make(chan os.Signal, 1)
	}

	signal.Notify(signals, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)

	// block until a signal is received.
	onClose(<-signals)
	close(signals)
}

// SignalHandler returns an actor, i.e. an execute and interrupt func, that
// terminates with SignalError when the process receives one of the provided
// signals, or the parent context is canceled.
//
// from https://github.com/oklog/run/blob/master/actors.go
func SignalHandler(ctx context.Context, signals ...os.Signal) (execute func() error, interrupt func(error)) {
	ctx, cancel := context.WithCancel(ctx)
	return func() error {
			c := make(chan os.Signal, 1)
			signal.Notify(c, signals...)
			defer signal.Stop(c)

			select {
			case sig := <-c:
				return SignalError{Signal: sig}
			case <-ctx.Done():
				return ctx.Err()
			}
		}, func(error) {
			cancel()
		}
}

// SignalError is returned by the signal handler's execute function
// when it terminates due to a received signal.
type SignalError struct {
	Signal os.Signal
}

// Error implements the error interface.
func (e SignalError) Error() string {
	return fmt.Sprintf("received signal %s", e.Signal)
}
