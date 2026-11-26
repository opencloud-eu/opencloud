package writer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	nserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/reva/v3/pkg/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMsg struct {
	data                 []byte
	seq                  uint64
	acked, naked, termed bool
	inProgress           int
}

func (m *fakeMsg) Data() []byte      { return m.data }
func (m *fakeMsg) Ack() error        { m.acked = true; return nil }
func (m *fakeMsg) Nak() error        { m.naked = true; return nil }
func (m *fakeMsg) InProgress() error { m.inProgress++; return nil }
func (m *fakeMsg) Term() error       { m.termed = true; return nil }
func (m *fakeMsg) Metadata() (*jetstream.MsgMetadata, error) {
	if m.seq == 0 {
		return nil, errors.New("no metadata")
	}
	return &jetstream.MsgMetadata{Sequence: jetstream.SequencePair{Stream: m.seq}}, nil
}

type failingSink struct{ calls int }

func (s *failingSink) Name() string { return "failing" }
func (s *failingSink) Write([]byte) error {
	s.calls++
	return errors.New("disk full")
}

// flakySink fails the first fails writes, then writes to out.
type flakySink struct {
	mu    sync.Mutex
	fails int
	calls int
	out   *syncBuffer
}

func (s *flakySink) Name() string { return "flaky" }
func (s *flakySink) Write(record []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.calls <= s.fails {
		return errors.New("collector unavailable")
	}
	if s.out != nil {
		_, _ = s.out.Write(append(record, '\n'))
	}
	return nil
}

// syncBuffer is a bytes.Buffer safe for the writer goroutine and the test.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

const record = `{"event.id":"019a3c1e-7b2f-7c4e-9a10-5d2f0e6b8a41","event.name":"opencloud.file.download"}`

func TestHandleWritesAndAcks(t *testing.T) {
	var out syncBuffer
	w := New(log.NopLogger(), NewLineSink("stdout", &out))
	msg := &fakeMsg{data: []byte(record)}

	w.Handle(context.Background(), msg)

	assert.True(t, msg.acked)
	assert.Equal(t, record+"\n", out.String())
}

func TestHandleTerminatesInvalidMessages(t *testing.T) {
	var out syncBuffer
	w := New(log.NopLogger(), NewLineSink("stdout", &out))

	for _, data := range []string{"not json", `{"event.name":"x"}`, `{}`} {
		msg := &fakeMsg{data: []byte(data)}
		w.Handle(context.Background(), msg)
		assert.True(t, msg.termed, data)
		assert.False(t, msg.acked, data)
	}
	assert.Empty(t, out.String())
}

func TestHandleRetriesOnlyTheOutputThatFailed(t *testing.T) {
	var out syncBuffer
	flaky := &flakySink{fails: 2}
	w := New(log.NopLogger(), NewLineSink("stdout", &out), flaky)
	w.RetryDelay = time.Millisecond
	msg := &fakeMsg{data: []byte(record)}

	w.Handle(context.Background(), msg)

	assert.True(t, msg.acked)
	assert.False(t, msg.naked)
	assert.Equal(t, 3, flaky.calls)
	assert.Equal(t, 2, msg.inProgress, "the message is kept during the retries")
	assert.Equal(t, record+"\n", out.String(), "the output that worked gets the record once")
}

func TestHandleGivesTheMessageBackWhenStopped(t *testing.T) {
	failing := &failingSink{}
	w := New(log.NopLogger(), failing)
	w.RetryDelay = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	msg := &fakeMsg{data: []byte(record)}

	done := make(chan struct{})
	go func() { w.Handle(ctx, msg); close(done) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	assert.True(t, msg.naked)
	assert.False(t, msg.acked, "a record that is not written everywhere is not acked")
	assert.Equal(t, 1, failing.calls)
}

func TestHandleSkipsACopyOfAWrittenRecord(t *testing.T) {
	var out syncBuffer
	w := New(log.NopLogger(), NewLineSink("stdout", &out))
	first := &fakeMsg{data: []byte(record), seq: 7}
	again := &fakeMsg{data: []byte(record), seq: 7}
	next := &fakeMsg{data: []byte(record), seq: 8}

	w.Handle(context.Background(), first)
	w.Handle(context.Background(), again)
	w.Handle(context.Background(), next)

	assert.True(t, again.acked, "the copy is acked")
	assert.Equal(t, 2, strings.Count(out.String(), "\n"), "but not written again")
}

func TestSeqWindowForgetsOldSequences(t *testing.T) {
	sw := newSeqWindow(2)
	sw.add(1)
	sw.add(2)
	sw.add(3)
	assert.False(t, sw.has(1))
	assert.True(t, sw.has(2))
	assert.True(t, sw.has(3))
}

func TestLagCountsUnackedRecords(t *testing.T) {
	assert.Equal(t, float64(7), lag(&jetstream.ConsumerInfo{NumPending: 3, NumAckPending: 4}))
}

func TestRunEndToEnd(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, err := nserver.NewServer(&nserver.Options{
		DontListen: true,
		JetStream:  true,
		StoreDir:   t.TempDir(),
		NoLog:      true,
		NoSigs:     true,
	})
	require.NoError(t, err)
	go srv.Start()
	require.True(t, srv.ReadyForConnections(5*time.Second))
	defer srv.Shutdown()

	nc, err := nats.Connect("", nats.InProcessServer(srv))
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)

	// the stream exists before the first service publishes
	_, err = audit.EnsureStream(ctx, js, audit.StreamConfig{})
	require.NoError(t, err)

	emitter := audit.NewEmitter(audit.JetStreamPublisher{JS: js}, audit.Options{
		Service: audit.Service{Name: "graph"},
	})
	require.NoError(t, emitter.Emit(ctx, audit.Event{
		Definition: audit.UserCreate,
		Actor:      &audit.Actor{Type: audit.ActorCLI, ID: "admin"},
		Target:     audit.Target{ID: "u-1", Name: "alice"},
	}))
	// the emitter publishes in the background, Close waits until it is stored
	emitter.Close()

	var out syncBuffer
	w := New(log.NopLogger(), NewLineSink("stdout", &out))
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx, js, audit.StreamConfig{}) }()

	require.Eventually(t, func() bool { return strings.Count(out.String(), "\n") == 1 }, 5*time.Second, 20*time.Millisecond)

	var rec audit.Record
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out.String())), &rec))
	assert.Equal(t, "opencloud.user.create", rec.EventName)
	assert.Equal(t, audit.ActorCLI, rec.ActorType)
	assert.Equal(t, "u-1", rec.TargetID)

	// a second publish with the same event id is dropped by the stream
	m := nats.NewMsg(audit.Subject("", audit.CategoryIdentity))
	m.Header.Set(nats.MsgIdHdr, rec.EventID)
	m.Data = []byte(strings.TrimSpace(out.String()))
	ack, err := js.PublishMsg(ctx, m)
	require.NoError(t, err)
	assert.True(t, ack.Duplicate)

	require.Eventually(t, func() bool {
		cons, err := js.Consumer(ctx, audit.StreamName, ConsumerName)
		if err != nil {
			return false
		}
		info, err := cons.Info(ctx)
		return err == nil && info.NumAckPending == 0 && info.NumPending == 0
	}, 5*time.Second, 20*time.Millisecond, "the record is acked")
	assert.Equal(t, 1, strings.Count(out.String(), "\n"))

	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("writer did not stop")
	}
}

func TestRunWritesEveryRecordOnceAfterAnOutage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, err := nserver.NewServer(&nserver.Options{
		DontListen: true,
		JetStream:  true,
		StoreDir:   t.TempDir(),
		NoLog:      true,
		NoSigs:     true,
	})
	require.NoError(t, err)
	go srv.Start()
	require.True(t, srv.ReadyForConnections(5*time.Second))
	defer srv.Shutdown()

	nc, err := nats.Connect("", nats.InProcessServer(srv))
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	_, err = audit.EnsureStream(ctx, js, audit.StreamConfig{})
	require.NoError(t, err)

	emitter := audit.NewEmitter(audit.JetStreamPublisher{JS: js}, audit.Options{
		Service: audit.Service{Name: "graph"},
	})
	for _, id := range []string{"u-1", "u-2", "u-3"} {
		require.NoError(t, emitter.Emit(ctx, audit.Event{
			Definition: audit.UserCreate,
			Actor:      &audit.Actor{Type: audit.ActorCLI, ID: "admin"},
			Target:     audit.Target{ID: id},
		}))
	}
	emitter.Close()

	var stdout, collector syncBuffer
	w := New(log.NopLogger(), NewLineSink("stdout", &stdout), &flakySink{fails: 3, out: &collector})
	w.RetryDelay = 10 * time.Millisecond
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx, js, audit.StreamConfig{}) }()

	require.Eventually(t, func() bool { return strings.Count(collector.String(), "\n") == 3 }, 5*time.Second, 20*time.Millisecond)
	require.Eventually(t, func() bool {
		cons, err := js.Consumer(ctx, audit.StreamName, ConsumerName)
		if err != nil {
			return false
		}
		info, err := cons.Info(ctx)
		return err == nil && info.NumAckPending == 0 && info.NumPending == 0
	}, 5*time.Second, 20*time.Millisecond, "all records are acked")

	for _, out := range []*syncBuffer{&stdout, &collector} {
		var ids []string
		for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
			var rec audit.Record
			require.NoError(t, json.Unmarshal([]byte(line), &rec))
			ids = append(ids, rec.TargetID)
		}
		assert.Equal(t, []string{"u-1", "u-2", "u-3"}, ids, "every record once and in order")
	}

	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("writer did not stop")
	}
}
