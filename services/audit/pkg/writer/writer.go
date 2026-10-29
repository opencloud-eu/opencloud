// Package writer consumes the audit stream and writes every record to the
// configured outputs. A message is acked only after all outputs have written
// it, so a crash or a failing output replays it instead of losing it.
package writer

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/reva/v2/pkg/audit"
)

// ConsumerName is the durable consumer of the audit service on the audit stream.
const ConsumerName = "audit"

// Sink writes one audit record, a single line of JSON without the newline.
type Sink interface {
	Name() string
	Write(record []byte) error
}

// LineSink writes one record per line to w, e.g. stdout.
type LineSink struct {
	name string
	mu   sync.Mutex
	w    *bufio.Writer
}

// NewLineSink returns a sink that writes to w.
func NewLineSink(name string, w io.Writer) *LineSink {
	return &LineSink{name: name, w: bufio.NewWriter(w)}
}

// Name of the sink, used in logs and metrics.
func (s *LineSink) Name() string { return s.name }

// Write writes the record and a newline and flushes, so nothing waits in a
// buffer when the message is acked.
func (s *LineSink) Write(record []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.w.Write(record); err != nil {
		return err
	}
	if err := s.w.WriteByte('\n'); err != nil {
		return err
	}
	return s.w.Flush()
}

// Message is the part of a JetStream message the writer needs.
type Message interface {
	Data() []byte
	Ack() error
	NakWithDelay(time.Duration) error
	Term() error
}

// Writer writes audit records to its sinks.
type Writer struct {
	sinks []Sink
	log   log.Logger
	// RetryDelay before a message is delivered again after an output failed.
	RetryDelay time.Duration
}

// New returns a writer for the given sinks.
func New(logger log.Logger, sinks ...Sink) *Writer {
	registerMetrics()
	return &Writer{sinks: sinks, log: logger, RetryDelay: 5 * time.Second}
}

// Handle writes one message to all sinks and acks it. A message that is not
// a record is terminated, it would fail forever. If an output fails, the
// message is delivered again later.
func (w *Writer) Handle(msg Message) {
	data := msg.Data()
	var rec audit.Record
	if err := json.Unmarshal(data, &rec); err != nil || rec.EventID == "" || rec.EventName == "" {
		recordsTotal.WithLabelValues("invalid").Inc()
		w.log.Error().Err(err).Msg("audit stream message is not an audit record, dropped")
		_ = msg.Term()
		return
	}

	for _, s := range w.sinks {
		if err := s.Write(data); err != nil {
			sinkErrorsTotal.WithLabelValues(s.Name()).Inc()
			w.log.Error().Err(err).Str("sink", s.Name()).Str("event_id", rec.EventID).Msg("audit record not written, retrying")
			_ = msg.NakWithDelay(w.RetryDelay)
			return
		}
	}
	if err := msg.Ack(); err != nil {
		// the record is written, a redelivery only causes a duplicate with the same event.id
		w.log.Error().Err(err).Str("event_id", rec.EventID).Msg("audit record written but not acked")
	}
	recordsTotal.WithLabelValues("written").Inc()
}

// Run consumes the audit stream until ctx is done.
func (w *Writer) Run(ctx context.Context, js jetstream.JetStream, cfg audit.StreamConfig) error {
	stream, err := audit.EnsureStream(ctx, js, cfg)
	if err != nil {
		return err
	}
	cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       ConsumerName,
		Description:   "audit service, writes the audit records to the outputs",
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxAckPending: 1000,
	})
	if err != nil {
		return err
	}

	go w.reportLag(ctx, cons)

	iter, err := cons.Messages()
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		iter.Stop()
	}()

	for {
		msg, err := iter.Next()
		if err != nil {
			if errors.Is(err, jetstream.ErrMsgIteratorClosed) || ctx.Err() != nil {
				return nil
			}
			w.log.Error().Err(err).Msg("reading the audit stream failed")
			continue
		}
		w.Handle(msg)
	}
}

func (w *Writer) reportLag(ctx context.Context, cons jetstream.Consumer) {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			info, err := cons.Info(ctx)
			if err != nil {
				continue
			}
			consumerLag.Set(float64(info.NumPending))
		}
	}
}
