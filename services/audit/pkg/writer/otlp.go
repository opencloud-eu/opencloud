package writer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"

	"github.com/opencloud-eu/reva/v3/pkg/audit"
)

// OTLPSink sends every record as an OpenTelemetry log record, an event with
// EventName set to event.name. The fields are attributes, the body is the
// record as JSON. The resource names the service that emitted the event, the
// instrumentation scope the audit service. Every record is exported on its own
// and synchronously, so the writer only acks it after the Collector took it.
type OTLPSink struct {
	exporter Exporter
	cfg      OTLPConfig

	mu        sync.Mutex
	loggers   map[service]otellog.Logger
	providers []*sdklog.LoggerProvider
}

// service identifies the resource of a record.
type service struct{ name, version string }

// scopeName is the instrumentation scope of the records.
const scopeName = "github.com/opencloud-eu/opencloud/services/audit"

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
	// Service names the audit service, the resource of records without a
	// service and the version of the instrumentation scope.
	Service string
	Version string
}

// NewOTLPSink returns a sink exporting to an OTLP gRPC endpoint.
func NewOTLPSink(ctx context.Context, cfg OTLPConfig) (*OTLPSink, error) {
	if err := validEndpoint(cfg.Endpoint); err != nil {
		return nil, fmt.Errorf("invalid OTLP endpoint %q, expected host:port: %w", cfg.Endpoint, err)
	}
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

// validEndpoint checks the endpoint up front, the gRPC client would only fail
// on every export.
func validEndpoint(endpoint string) error {
	if strings.TrimSpace(endpoint) != endpoint {
		return errors.New("leading or trailing whitespace")
	}
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return err
	}
	if host == "" {
		return errors.New("missing host")
	}
	if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("invalid port %q", port)
	}
	return nil
}

func newOTLPSink(exp Exporter, cfg OTLPConfig) *OTLPSink {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &OTLPSink{exporter: exp, cfg: cfg, loggers: map[service]otellog.Logger{}}
}

// logger returns the logger for the resource of svc. The SDK binds the
// resource to the provider, so every service gets its own provider; they
// share the exporter.
func (s *OTLPSink) logger(svc service) otellog.Logger {
	if svc.name == "" {
		svc = service{name: s.cfg.Service, version: s.cfg.Version}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if l, ok := s.loggers[svc]; ok {
		return l
	}
	attrs := []attribute.KeyValue{attribute.String("service.name", svc.name)}
	if svc.version != "" {
		attrs = append(attrs, attribute.String("service.version", svc.version))
	}
	res := resource.NewSchemaless(attrs...)
	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(&syncProcessor{exporter: s.exporter, timeout: s.cfg.Timeout}),
	)
	l := provider.Logger(scopeName, otellog.WithInstrumentationVersion(s.cfg.Version))
	s.loggers[svc] = l
	s.providers = append(s.providers, provider)
	return l
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
	// the emitting service is the resource, not an attribute
	svcName, _ := fields["service.name"].(string)
	svcVersion, _ := fields["service.version"].(string)
	delete(fields, "service.name")
	delete(fields, "service.version")
	rec.AddAttributes(attributes(fields)...)

	// the SDK takes the trace from the context
	ctx := context.Background()
	if sc, ok := spanContext(fields); ok {
		ctx = trace.ContextWithSpanContext(ctx, sc)
	}
	result := &exportResult{}
	s.logger(service{name: svcName, version: svcVersion}).Emit(context.WithValue(ctx, exportResultKey{}, result), rec)
	return result.err
}

// Shutdown stops the providers and the exporter.
func (s *OTLPSink) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var errs []error
	for _, p := range s.providers {
		errs = append(errs, p.Shutdown(ctx))
	}
	errs = append(errs, s.exporter.Shutdown(ctx))
	return errors.Join(errs...)
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

// Shutdown does nothing, the providers share the exporter and the sink stops it.
func (p *syncProcessor) Shutdown(context.Context) error { return nil }

// ForceFlush does nothing, every record is exported right away.
func (p *syncProcessor) ForceFlush(context.Context) error { return nil }
