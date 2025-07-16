package raw

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/pkg/errors"
)

// Config is the configuration needed for a NATS event stream
type Config struct {
	Endpoint             string        `mapstructure:"address"`          // Endpoint of the nats server
	Cluster              string        `mapstructure:"clusterID"`        // CluserID of the nats cluster
	TLSInsecure          bool          `mapstructure:"tls-insecure"`     // Whether to verify TLS certificates
	TLSRootCACertificate string        `mapstructure:"tls-root-ca-cert"` // The root CA certificate used to validate the TLS certificate
	EnableTLS            bool          `mapstructure:"enable-tls"`       // Enable TLS
	AuthUsername         string        `mapstructure:"username"`         // Username for authentication
	AuthPassword         string        `mapstructure:"password"`         // Password for authentication
	MaxAckPending        int           `mapstructure:"max-ack-pending"`  // Maximum number of unacknowledged messages
	AckWait              time.Duration `mapstructure:"ack-wait"`         // Time to wait for an ack
}

type RawEvent struct {
	Timestamp time.Time
	Metadata  map[string]string
	ID        string
	Topic     string
	Payload   []byte

	msg jetstream.Msg
}

type Event struct {
	events.Event

	msg jetstream.Msg
}

func (re *Event) Ack() error {
	if re.msg == nil {
		return errors.New("cannot ack event without message")
	}
	return re.msg.Ack()
}

func (re *Event) InProgress() error {
	if re.msg == nil {
		return errors.New("cannot mark event as in progress without message")
	}
	return re.msg.InProgress()
}

type Stream interface {
	Consume(group string, evs ...events.Unmarshaller) (<-chan Event, error)
}

type RawStream struct {
	Js jetstream.Stream

	c Config
}

func FromConfig(ctx context.Context, name string, cfg Config) (Stream, error) {
	var s Stream
	b := backoff.NewExponentialBackOff()

	connect := func() error {
		var tlsConf *tls.Config
		if cfg.EnableTLS {
			var rootCAPool *x509.CertPool
			if cfg.TLSRootCACertificate != "" {
				rootCrtFile, err := os.Open(cfg.TLSRootCACertificate)
				if err != nil {
					return err
				}

				rootCAPool, err = newCertPoolFromPEM(rootCrtFile)
				if err != nil {
					return err
				}
				cfg.TLSInsecure = false
			}

			tlsConf = &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: cfg.TLSInsecure,
				RootCAs:            rootCAPool,
			}
		}

		nopts := nats.GetDefaultOptions()
		nopts.Name = name
		if tlsConf != nil {
			nopts.Secure = true
			nopts.TLSConfig = tlsConf
		}

		if len(cfg.Endpoint) > 0 {
			nopts.Servers = []string{cfg.Endpoint}
		}

		if cfg.AuthUsername != "" && cfg.AuthPassword != "" {
			nopts.User = cfg.AuthUsername
			nopts.Password = cfg.AuthPassword
		}

		conn, err := nopts.Connect()
		if err != nil {
			return err
		}

		jsConn, err := jetstream.New(conn)
		if err != nil {
			return err
		}

		js, err := jsConn.Stream(ctx, events.MainQueueName)
		if err != nil {
			return err
		}

		s = &RawStream{
			Js: js,
			c:  cfg,
		}
		return nil
	}
	err := backoff.Retry(connect, b)
	if err != nil {
		return s, errors.Wrap(err, "could not connect to nats jetstream")
	}
	return s, nil
}

func (s *RawStream) Consume(group string, evs ...events.Unmarshaller) (<-chan Event, error) {
	c, err := s.consumeRaw(group)
	if err != nil {
		return nil, err
	}

	registeredEvents := map[string]events.Unmarshaller{}
	for _, e := range evs {
		typ := reflect.TypeOf(e)
		registeredEvents[typ.String()] = e
	}

	outchan := make(chan Event)
	go func() {
		for {
			e := <-c
			eventType := e.Metadata[events.MetadatakeyEventType]
			ev, ok := registeredEvents[eventType]
			if !ok {
				_ = e.msg.Ack() // Discard. We are not interested in this event type
				continue
			}

			event, err := ev.Unmarshal(e.Payload)
			if err != nil {
				continue
			}

			outchan <- Event{
				Event: events.Event{
					Type:        eventType,
					ID:          e.Metadata[events.MetadatakeyEventID],
					TraceParent: e.Metadata[events.MetadatakeyTraceParent],
					InitiatorID: e.Metadata[events.MetadatakeyInitiatorID],
					Event:       event,
				},
				msg: e.msg,
			}
		}
	}()
	return outchan, nil
}

func (s *RawStream) consumeRaw(group string) (<-chan RawEvent, error) {
	consumer, err := s.Js.CreateOrUpdateConsumer(context.Background(), jetstream.ConsumerConfig{
		Durable:       group,
		DeliverPolicy: jetstream.DeliverNewPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy, // Require manual acknowledgment
		MaxAckPending: s.c.MaxAckPending,           // Maximum number of unacknowledged messages
		AckWait:       s.c.AckWait,                 // Time to wait for an ack
	})
	if err != nil {
		return nil, err
	}

	channel := make(chan RawEvent)
	callback := func(msg jetstream.Msg) {
		var rawEvent RawEvent
		if err := json.Unmarshal(msg.Data(), &rawEvent); err != nil {
			fmt.Printf("error unmarshalling event: %v\n", err)
			return
		}
		rawEvent.msg = msg
		channel <- rawEvent
	}
	_, err = consumer.Consume(callback)
	if err != nil {
		return nil, err
	}

	return channel, nil
}
