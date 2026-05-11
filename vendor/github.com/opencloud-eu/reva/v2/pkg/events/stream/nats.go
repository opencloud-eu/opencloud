package stream

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"os"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/go-micro/plugins/v4/events/natsjs"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/raw"
	"github.com/opencloud-eu/reva/v2/pkg/logger"
)

// NatsConfig is the configuration needed for a NATS event stream
type NatsConfig struct {
	Endpoint             string `mapstructure:"address"`          // Endpoint of the nats server
	Cluster              string `mapstructure:"clusterID"`        // CluserID of the nats cluster
	TLSInsecure          bool   `mapstructure:"tls-insecure"`     // Whether to verify TLS certificates
	TLSRootCACertificate string `mapstructure:"tls-root-ca-cert"` // The root CA certificate used to validate the TLS certificate
	EnableTLS            bool   `mapstructure:"enable-tls"`       // Enable TLS
	AuthUsername         string `mapstructure:"username"`         // Username for authentication
	AuthPassword         string `mapstructure:"password"`         // Password for authentication

}

// NatsFromConfig returns a nats stream from the given config
func NatsFromConfig(connName string, disableDurability bool, cfg NatsConfig) (events.Stream, error) {
	var tlsConf *tls.Config
	if cfg.EnableTLS {
		var rootCAPool *x509.CertPool
		if cfg.TLSRootCACertificate != "" {
			rootCrtFile, err := os.Open(cfg.TLSRootCACertificate)
			if err != nil {
				return nil, err
			}

			rootCAPool, err = newCertPoolFromPEM(rootCrtFile)
			if err != nil {
				return nil, err
			}
			cfg.TLSInsecure = false
		}

		tlsConf = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: cfg.TLSInsecure, //nolint:gosec
			RootCAs:            rootCAPool,
		}
	}

	opts := []natsjs.Option{
		natsjs.TLSConfig(tlsConf),
		natsjs.Address(cfg.Endpoint),
		natsjs.ClusterID(cfg.Cluster),
		natsjs.SynchronousPublish(true),
		natsjs.Name(connName),
		natsjs.Authenticate(cfg.AuthUsername, cfg.AuthPassword),
	}

	if disableDurability {
		opts = append(opts, natsjs.DisableDurableStreams())
	}

	s, err := Nats(opts...)
	if err != nil {
		return nil, err
	}

	// apply a MaxAge to the main queue to prevent it from filling up
	ctx := context.Background()
	jsConn, err := raw.JetStream(ctx, connName, raw.Config{
		Endpoint:             cfg.Endpoint,
		Cluster:              cfg.Cluster,
		TLSInsecure:          cfg.TLSInsecure,
		TLSRootCACertificate: cfg.TLSRootCACertificate,
		EnableTLS:            cfg.EnableTLS,
		AuthUsername:         cfg.AuthUsername,
		AuthPassword:         cfg.AuthPassword,
	})
	if err != nil {
		return nil, err
	}
	streamCfg := jetstream.StreamConfig{
		Name:   "main-queue",
		MaxAge: 7 * 24 * time.Hour,
	}
	_, err = jsConn.CreateStream(ctx, streamCfg)
	if err != nil {
		// If the stream already exists, update its configuration
		if err == jetstream.ErrStreamNameAlreadyInUse {
			_, _ = jsConn.UpdateStream(ctx, streamCfg)
		}
	}

	return s, nil
}

// nats returns a nats streaming client
// retries exponentially to connect to a nats server
func Nats(opts ...natsjs.Option) (events.Stream, error) {
	b := backoff.NewExponentialBackOff()
	var stream events.Stream
	o := func() error {
		n := b.NextBackOff()
		s, err := natsjs.NewStream(opts...)
		if err != nil && n > time.Second {
			logger.New().Error().Err(err).Msgf("can't connect to nats (jetstream) server, retrying in %s", n)
		}
		stream = s
		return err
	}

	err := backoff.Retry(o, b)
	return stream, err
}

// newCertPoolFromPEM reads certificates from io.Reader and returns a x509.CertPool
// containing those certificates.
func newCertPoolFromPEM(crts ...io.Reader) (*x509.CertPool, error) {
	certPool := x509.NewCertPool()

	var buf bytes.Buffer
	for _, c := range crts {
		if _, err := io.Copy(&buf, c); err != nil {
			return nil, err
		}
		if !certPool.AppendCertsFromPEM(buf.Bytes()) {
			return nil, errors.New("failed to append cert from PEM")
		}
		buf.Reset()
	}

	return certPool, nil
}
