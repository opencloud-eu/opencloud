// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func NewNatsKeyValue(c Config) (jetstream.KeyValue, error) {
	nodes := strings.Join(c.Nodes, ",")
	if nodes == "" {
		return nil, errors.New("at least one node is required")
	}
	opts := []nats.Option{}
	if c.AuthUsername != "" || c.AuthPassword != "" {
		opts = append(opts, nats.UserInfo(c.AuthUsername, c.AuthPassword))
	}

	if c.TLSEnabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: c.TLSInsecure,
		}
		if c.TLSRootCACertificate != "" {
			caCert, err := os.ReadFile(c.TLSRootCACertificate)
			if err == nil {
				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(caCert)
				tlsConfig.RootCAs = caCertPool
			}
		}
		opts = append(opts, nats.Secure(tlsConfig))
	}

	var js jetstream.JetStream
	o := func() error {
		nc, err := nats.Connect(nodes, opts...)
		if err != nil {
			return err
		}

		js, err = jetstream.New(nc)
		return err
	}

	err := backoff.Retry(o, backoff.NewExponentialBackOff())
	if err != nil {
		return nil, err
	}

	return NewNatsKeyValueFromJetStream(c, js)
}

func NewNatsKeyValueFromJetStream(c Config, js jetstream.JetStream) (jetstream.KeyValue, error) {
	var err error
	var kv jetstream.KeyValue
	o := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		kv, err = js.KeyValue(ctx, c.Database)
		if err != nil {
			kvConfig := jetstream.KeyValueConfig{
				Bucket: c.Database,
				TTL:    c.TTL,
			}
			if c.DisablePersistence {
				kvConfig.Storage = jetstream.MemoryStorage
			}
			if c.Size > 0 {
				kvConfig.MaxBytes = int64(c.Size)
			}
			kv, err = js.CreateKeyValue(ctx, kvConfig)
			if err != nil {
				panic(err)
			}
		}

		return nil
	}
	err = backoff.Retry(o, backoff.NewExponentialBackOff())
	if err != nil {
		return nil, err
	}

	return kv, nil
}
