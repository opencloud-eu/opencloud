package writer

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeExporter struct {
	records []sdklog.Record
	err     error
}

func (e *fakeExporter) Export(_ context.Context, records []sdklog.Record) error {
	if e.err != nil {
		return e.err
	}
	e.records = append(e.records, records...)
	return nil
}
func (e *fakeExporter) Shutdown(context.Context) error   { return nil }
func (e *fakeExporter) ForceFlush(context.Context) error { return nil }

const fullRecord = `{"event.id":"019a3c1e-7b2f-7c4e-9a10-5d2f0e6b8a41","event.name":"opencloud.auth.login","event.code":"A001F","time":"2026-10-28T09:14:03.512Z","outcome":"failure","severity":"warning","client.address":"203.0.113.7","change.after":{"name":"x"},"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7","ocsf.class_uid":3002}`

func TestOTLPSinkExportsAnEvent(t *testing.T) {
	exp := &fakeExporter{}
	sink := newOTLPSink(exp, OTLPConfig{Service: "audit", Version: "5.1.0"})

	require.NoError(t, sink.Write([]byte(fullRecord)))
	require.Len(t, exp.records, 1)
	r := exp.records[0]

	assert.Equal(t, "opencloud.auth.login", r.EventName())
	assert.Equal(t, time.Date(2026, 10, 28, 9, 14, 3, 512_000_000, time.UTC), r.Timestamp().UTC())
	assert.Equal(t, otellog.SeverityWarn, r.Severity())
	assert.Equal(t, "warning", r.SeverityText())
	assert.Equal(t, fullRecord, r.Body().AsString(), "the body is the whole record")
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", r.TraceID().String())
	assert.Equal(t, "00f067aa0ba902b7", r.SpanID().String())

	attrs := map[string]attribute.Value{}
	r.WalkAttributes(func(kv attribute.KeyValue) bool {
		attrs[string(kv.Key)] = kv.Value
		return true
	})
	assert.Equal(t, "203.0.113.7", attrs["client.address"].AsString())
	assert.Equal(t, int64(3002), attrs["ocsf.class_uid"].AsInt64(), "numbers stay numbers")
	assert.Equal(t, `{"name":"x"}`, attrs["change.after"].AsString(), "nested values become JSON")

	svc, ok := r.Resource().Set().Value("service.name")
	require.True(t, ok)
	assert.Equal(t, "audit", svc.AsString())
}

func TestOTLPSinkReportsExportErrors(t *testing.T) {
	exp := &fakeExporter{err: errors.New("collector unavailable")}
	sink := newOTLPSink(exp, OTLPConfig{Service: "audit"})

	assert.Error(t, sink.Write([]byte(fullRecord)))
}

func TestWriterRetriesWhenTheCollectorIsDown(t *testing.T) {
	exp := &fakeExporter{err: errors.New("collector unavailable")}
	w := New(log.NopLogger(), newOTLPSink(exp, OTLPConfig{Service: "audit"}))
	msg := &fakeMsg{data: []byte(fullRecord)}

	w.Handle(msg)

	assert.True(t, msg.naked)
	assert.False(t, msg.acked)
}
