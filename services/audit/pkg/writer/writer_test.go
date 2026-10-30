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
	"github.com/opencloud-eu/reva/v2/pkg/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMsg struct {
	data                 []byte
	acked, naked, termed bool
}

func (m *fakeMsg) Data() []byte                     { return m.data }
func (m *fakeMsg) Ack() error                       { m.acked = true; return nil }
func (m *fakeMsg) NakWithDelay(time.Duration) error { m.naked = true; return nil }
func (m *fakeMsg) Term() error                      { m.termed = true; return nil }

type failingSink struct{ calls int }

func (s *failingSink) Name() string { return "failing" }
func (s *failingSink) Write([]byte) error {
	s.calls++
	return errors.New("disk full")
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

	w.Handle(msg)

	assert.True(t, msg.acked)
	assert.Equal(t, record+"\n", out.String())
}

func TestHandleTerminatesInvalidMessages(t *testing.T) {
	var out syncBuffer
	w := New(log.NopLogger(), NewLineSink("stdout", &out))

	for _, data := range []string{"not json", `{"event.name":"x"}`, `{}`} {
		msg := &fakeMsg{data: []byte(data)}
		w.Handle(msg)
		assert.True(t, msg.termed, data)
		assert.False(t, msg.acked, data)
	}
	assert.Empty(t, out.String())
}

func TestHandleRetriesWhenAnOutputFails(t *testing.T) {
	var out syncBuffer
	failing := &failingSink{}
	w := New(log.NopLogger(), failing, NewLineSink("stdout", &out))
	msg := &fakeMsg{data: []byte(record)}

	w.Handle(msg)

	assert.True(t, msg.naked)
	assert.False(t, msg.acked, "a record that is not written everywhere is not acked")
	assert.Empty(t, out.String(), "the next outputs are not written either")
	assert.Equal(t, 1, failing.calls)
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
		Mode:    audit.ModeStrict,
	})
	require.NoError(t, emitter.Emit(ctx, audit.Event{
		Definition: audit.UserCreate,
		Actor:      &audit.Actor{Type: audit.ActorCLI, ID: "admin"},
		Target:     audit.Target{ID: "u-1", Name: "alice"},
	}))

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
