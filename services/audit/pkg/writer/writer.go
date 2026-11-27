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
	"github.com/opencloud-eu/reva/v3/pkg/audit"
)

// ConsumerName is the durable consumer of the audit service on the audit stream.
const ConsumerName = "audit"

// maxAckPending limits the messages delivered to the audit service and not
// acked yet.
const maxAckPending = 1000

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
	Nak() error
	InProgress() error
	Term() error
	Metadata() (*jetstream.MsgMetadata, error)
}

// Writer writes audit records to its sinks.
type Writer struct {
	sinks []Sink
	log   log.Logger
	// RetryDelay before a record is written again to an output that failed.
	// Together with the export timeout it must stay below the ack wait.
	RetryDelay time.Duration
	written    *seqWindow
}

// New returns a writer for the given sinks.
func New(logger log.Logger, sinks ...Sink) *Writer {
	registerMetrics()
	return &Writer{sinks: sinks, log: logger, RetryDelay: 5 * time.Second, written: newSeqWindow(4 * maxAckPending)}
}

// Handle writes one message to all sinks and acks it. A message that is not
// a record is terminated, it would fail forever. If an output fails, Handle
// keeps the message and writes it again after RetryDelay, only to the outputs
// that failed, until all have it or ctx is done. The other outputs get the
// record once, and the records stay in order.
//
// While Handle retries, messages fetched ahead can run out of their ack wait
// and NATS sends them again. Handle acks such a copy of a record it already
// wrote without writing it again. Handle is not safe for concurrent use.
func (w *Writer) Handle(ctx context.Context, msg Message) {
	data := msg.Data()
	var rec audit.Record
	if err := json.Unmarshal(data, &rec); err != nil || rec.EventID == "" || rec.EventName == "" {
		recordsTotal.WithLabelValues("invalid").Inc()
		w.log.Error().Err(err).Msg("audit stream message is not an audit record, dropped")
		_ = msg.Term()
		return
	}

	var seq uint64
	if md, err := msg.Metadata(); err == nil {
		seq = md.Sequence.Stream
	}
	if seq > 0 && w.written.has(seq) {
		_ = msg.Ack()
		recordsTotal.WithLabelValues("duplicate").Inc()
		return
	}

	pending := w.sinks
	for {
		var failed []Sink
		for _, s := range pending {
			if err := s.Write(data); err != nil {
				sinkErrorsTotal.WithLabelValues(s.Name()).Inc()
				w.log.Error().Err(err).Str("sink", s.Name()).Str("event_id", rec.EventID).Msg("audit record not written, retrying")
				failed = append(failed, s)
			}
		}
		if len(failed) == 0 {
			break
		}
		pending = failed
		select {
		case <-ctx.Done():
			// the next start writes it again, the other outputs then get a duplicate
			_ = msg.Nak()
			return
		case <-time.After(w.RetryDelay):
		}
		// keep the message, NATS would send it again after the ack wait
		if err := msg.InProgress(); err != nil {
			w.log.Error().Err(err).Str("event_id", rec.EventID).Msg("could not extend the ack wait of the audit record")
		}
	}
	if seq > 0 {
		w.written.add(seq)
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
		MaxAckPending: maxAckPending,
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

	var delay time.Duration
	for {
		msg, err := iter.Next()
		if err != nil {
			if errors.Is(err, jetstream.ErrMsgIteratorClosed) || ctx.Err() != nil {
				return nil
			}
			// without a pause a persistent error fills the log as fast as it repeats
			delay = nextReadDelay(delay, w.RetryDelay)
			w.log.Error().Err(err).Dur("retry_in", delay).Msg("reading the audit stream failed")
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(delay):
			}
			continue
		}
		delay = 0
		w.Handle(ctx, msg)
	}
}

// nextReadDelay is the pause after a failed read of the stream: 100ms after the
// first failure, then doubled up to limit.
func nextReadDelay(d, limit time.Duration) time.Duration {
	if d == 0 {
		return min(100*time.Millisecond, limit)
	}
	return min(2*d, limit)
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
			consumerLag.Set(lag(info))
		}
	}
}

// lag counts the records not written yet: the ones not delivered and the ones
// delivered but not acked, e.g. while an output is down.
func lag(info *jetstream.ConsumerInfo) float64 {
	return float64(info.NumPending) + float64(info.NumAckPending)
}

// seqWindow remembers the last stream sequences written. A copy NATS sends
// again is at most maxAckPending messages behind the original.
type seqWindow struct {
	set  map[uint64]struct{}
	ring []uint64
	next int
}

func newSeqWindow(size int) *seqWindow {
	return &seqWindow{set: make(map[uint64]struct{}, size), ring: make([]uint64, size)}
}

func (w *seqWindow) has(seq uint64) bool {
	_, ok := w.set[seq]
	return ok
}

func (w *seqWindow) add(seq uint64) {
	if old := w.ring[w.next]; old != 0 {
		delete(w.set, old)
	}
	w.ring[w.next] = seq
	w.set[seq] = struct{}{}
	w.next = (w.next + 1) % len(w.ring)
}
