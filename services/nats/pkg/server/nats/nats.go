package nats

import (
	"context"
	"time"

	nserver "github.com/nats-io/nats-server/v2/server"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/nats/pkg/logging"
	"github.com/rs/zerolog"
)

var NATSListenAndServeLoopTimer = 1 * time.Second

type NATSServer struct {
	ctx    context.Context
	server *nserver.Server
}

// NatsOption configures the new NATSServer instance
func NewNATSServer(ctx context.Context, logger log.Logger, opts ...NatsOption) (*NATSServer, error) {
	natsOpts := &nserver.Options{}

	for _, o := range opts {
		o(natsOpts)
	}

	// enable JetStream
	natsOpts.JetStream = true
	// The NATS server itself runs the signal handling. We set `natsOpts.NoSigs = true` because we want to handle signals ourselves
	natsOpts.NoSigs = true

	server, err := nserver.NewServer(natsOpts)
	if err != nil {
		return nil, err
	}

	nLogger := logging.NewLogWrapper(logger)
	server.SetLoggerV2(nLogger, logger.GetLevel() <= zerolog.DebugLevel, logger.GetLevel() <= zerolog.TraceLevel, false)

	return &NATSServer{
		ctx:    ctx,
		server: server,
	}, nil
}

// ListenAndServe runs the NATSServer in a blocking way until the server is shutdown or an error occurs
func (n *NATSServer) ListenAndServe() (err error) {
	go n.server.Start()
	<-n.ctx.Done()
	return nil
}

// Shutdown stops the NATSServer gracefully
func (n *NATSServer) Shutdown() {
	n.server.Shutdown()
	n.server.WaitForShutdown()
}
