package writer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"

	"github.com/opencloud-eu/reva/v2/pkg/audit"
)

// OTLPSink sends every record as an OpenTelemetry log record, an event with
// EventName set to event.name. The fields are attributes, the body is the
// record as JSON. Every record is exported on its own and synchronously, so
// the writer only acks it after the Collector took it.
type OTLPSink struct {
	provider *sdklog.LoggerProvider
	logger   otellog.Logger
}

// Exporter sends log records, in production the OTLP gRPC exporter.
type Exporter interface {
	Export(ctx context.Context, records []sdklog.Record) error
	Shutdown(ctx context.Context) error
	ForceFlush(ctx context.Context) error
}

// OTLPConfig configures the OTLP output.
type OTLPConfig struct {
	Endpoint string
	Insecure bool
	Timeout  time.Duration
	// Service names the audit service in the OTel resource.
	Service string
	Version string
}

// NewOTLPSink returns a sink exporting to an OTLP gRPC endpoint.
func NewOTLPSink(ctx context.Context, cfg OTLPConfig) (*OTLPSink, error) {
	opts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(cfg.Endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}
	exp, err := otlploggrpc.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return newOTLPSink(exp, cfg), nil
}

func newOTLPSink(exp Exporter, cfg OTLPConfig) *OTLPSink {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	res := resource.NewSchemaless(
		attribute.String("service.name", cfg.Service),
		attribute.String("service.version", cfg.Version),
	)
	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(&syncProcessor{exporter: exp, timeout: cfg.Timeout}),
	)
	return &OTLPSink{provider: provider, logger: provider.Logger("github.com/opencloud-eu/opencloud/services/audit")}
}

// Name of the sink.
func (s *OTLPSink) Name() string { return "otlp" }

// Write exports one record and returns the export error.
func (s *OTLPSink) Write(record []byte) error {
	var fields map[string]any
	if err := json.Unmarshal(record, &fields); err != nil {
		return err
	}

	var rec otellog.Record
	name, _ := fields["event.name"].(string)
	rec.SetEventName(name)
	rec.SetObservedTimestamp(time.Now())
	if ts, ok := fields["time"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			rec.SetTimestamp(t)
		}
	}
	sev, _ := fields["severity"].(string)
	rec.SetSeverity(severity(audit.Severity(sev)))
	rec.SetSeverityText(sev)
	rec.SetBody(attribute.StringValue(string(record)))
	rec.AddAttributes(attributes(fields)...)

	// the SDK takes the trace from the context
	ctx := context.Background()
	if sc, ok := spanContext(fields); ok {
		ctx = trace.ContextWithSpanContext(ctx, sc)
	}
	result := &exportResult{}
	s.logger.Emit(context.WithValue(ctx, exportResultKey{}, result), rec)
	return result.err
}

// Shutdown flushes and stops the exporter.
func (s *OTLPSink) Shutdown(ctx context.Context) error {
	return s.provider.Shutdown(ctx)
}

func severity(s audit.Severity) otellog.Severity {
	switch s {
	case audit.SeverityNotice:
		return otellog.SeverityInfo2
	case audit.SeverityWarning:
		return otellog.SeverityWarn
	case audit.SeverityAlert:
		return otellog.SeverityFatal
	default:
		return otellog.SeverityInfo
	}
}

// attributes turns the record fields into log attributes; nested values,
// like change.before, become JSON strings.
func attributes(fields map[string]any) []attribute.KeyValue {
	kvs := make([]attribute.KeyValue, 0, len(fields))
	for k, v := range fields {
		switch val := v.(type) {
		case string:
			kvs = append(kvs, attribute.String(k, val))
		case bool:
			kvs = append(kvs, attribute.Bool(k, val))
		case float64:
			if val == math.Trunc(val) {
				kvs = append(kvs, attribute.Int64(k, int64(val)))
			} else {
				kvs = append(kvs, attribute.Float64(k, val))
			}
		case nil:
		default:
			b, err := json.Marshal(val)
			if err != nil {
				b = []byte(fmt.Sprint(val))
			}
			kvs = append(kvs, attribute.String(k, string(b)))
		}
	}
	return kvs
}

func spanContext(fields map[string]any) (trace.SpanContext, bool) {
	tid, _ := fields["trace_id"].(string)
	sid, _ := fields["span_id"].(string)
	var cfg trace.SpanContextConfig
	if b, err := hex.DecodeString(tid); err == nil && len(b) == len(cfg.TraceID) {
		copy(cfg.TraceID[:], b)
	} else {
		return trace.SpanContext{}, false
	}
	if b, err := hex.DecodeString(sid); err == nil && len(b) == len(cfg.SpanID) {
		copy(cfg.SpanID[:], b)
	}
	cfg.TraceFlags = trace.FlagsSampled
	sc := trace.NewSpanContext(cfg)
	return sc, sc.IsValid()
}

type exportResultKey struct{}

// exportResult carries the export error back to Write, Logger.Emit has no
// return value.
type exportResult struct{ err error }

// syncProcessor exports every record right away and reports the error.
type syncProcessor struct {
	exporter Exporter
	timeout  time.Duration
}

func (p *syncProcessor) Enabled(context.Context, sdklog.EnabledParameters) bool { return true }

func (p *syncProcessor) OnEmit(ctx context.Context, r *sdklog.Record) error {
	exportCtx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	err := p.exporter.Export(exportCtx, []sdklog.Record{r.Clone()})
	if res, ok := ctx.Value(exportResultKey{}).(*exportResult); ok {
		res.err = err
	}
	return err
}

func (p *syncProcessor) Shutdown(ctx context.Context) error   { return p.exporter.Shutdown(ctx) }
func (p *syncProcessor) ForceFlush(ctx context.Context) error { return p.exporter.ForceFlush(ctx) }
