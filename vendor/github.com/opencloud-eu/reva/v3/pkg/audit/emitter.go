package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	ctxpkg "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

// Mode decides what happens when an event can't be stored.
type Mode string

// ModeBestEffort publishes asynchronously through a bounded buffer and never
// blocks a request. Events that don't fit or fail are dropped and counted. It
// is the only mode for now: a strict mode has to refuse a request before it
// acts, the events middleware only sees a call after it happened.
const ModeBestEffort Mode = "best_effort"

// ErrUnknownMode is returned for a mode other than best_effort.
var ErrUnknownMode = errors.New("audit: unknown mode, only best_effort is supported")

// ErrInvalidEvent is returned when an event misses required fields.
var ErrInvalidEvent = errors.New("audit: invalid event")

// Target is the object an event acts on.
type Target struct {
	ID      string
	SpaceID string
	Name    string
	Path    string
	// Type overrides the target type of the definition.
	Type string
}

// Change holds the changed attributes of an update. Never put secrets in it.
type Change struct {
	Before map[string]any
	After  map[string]any
}

// Actor overrides the actor taken from the context, e.g. for the CLI, the
// system or a public link.
type Actor struct {
	Type ActorType
	ID   string
	Name string
}

// Event is what a service reports.
type Event struct {
	Definition *Definition
	Outcome    Outcome
	// Reason is a machine readable reason for failures and denials, e.g. permission_denied.
	Reason string
	Target Target
	Change *Change
	// Actor, when set, replaces the actor from the context.
	Actor *Actor
	// Tenant, when set, replaces the tenant of the actor, e.g. the tenant of the resource.
	Tenant string
	// Time defaults to now.
	Time time.Time
}

// Emitter reports audit events.
type Emitter interface {
	Emit(ctx context.Context, ev Event) error
}

// Publisher stores a message, in production a JetStream publish that waits for the ack.
type Publisher interface {
	Publish(ctx context.Context, msg *nats.Msg) error
}

// Service describes the emitting service.
type Service struct {
	Name       string
	Version    string
	InstanceID string
}

// Options configure an emitter.
type Options struct {
	Service Service
	// BufferSize is the size of the buffer, default 1000.
	BufferSize int
	// PublishTimeout limits one publish, default 5s.
	PublishTimeout time.Duration
	Logger         *zerolog.Logger
}

// NatsEmitter builds records and publishes them.
type NatsEmitter struct {
	pub  Publisher
	opts Options
	log  zerolog.Logger

	queue chan *nats.Msg
	// mu guards closed: Emit holds it for reading while it queues, Close for
	// writing while it closes the queue, so no send hits a closed channel.
	mu     sync.RWMutex
	closed bool
	wg     sync.WaitGroup
	once   sync.Once
	now    func() time.Time
}

// NewEmitter returns an emitter and starts its worker; call Close to flush the
// buffer on shutdown.
func NewEmitter(pub Publisher, opts Options) *NatsEmitter {
	if opts.BufferSize <= 0 {
		opts.BufferSize = 1000
	}
	if opts.PublishTimeout <= 0 {
		opts.PublishTimeout = 5 * time.Second
	}
	log := zerolog.Nop()
	if opts.Logger != nil {
		log = *opts.Logger
	}
	e := &NatsEmitter{pub: pub, opts: opts, log: log, now: time.Now}
	registerMetrics()
	e.queue = make(chan *nats.Msg, opts.BufferSize)
	e.wg.Add(1)
	go e.work()
	return e
}

// Emit builds the record and queues it for publishing. It never blocks: when
// the buffer is full, the record is dropped and counted.
func (e *NatsEmitter) Emit(ctx context.Context, ev Event) error {
	rec, err := e.Record(ctx, ev)
	if err != nil {
		name := "unknown"
		if ev.Definition != nil {
			name = ev.Definition.Name
		}
		e.dropped(name, "invalid")
		e.log.Error().Err(err).Msg("audit event not emitted")
		return err
	}
	msg, err := message(rec, trace.SpanContextFromContext(ctx))
	if err != nil {
		e.dropped(rec.EventName, "marshal")
		e.log.Error().Err(err).Str("event", rec.EventName).Msg("audit event not emitted")
		return err
	}
	emittedTotal.WithLabelValues(e.opts.Service.Name, rec.EventName, string(rec.Outcome)).Inc()

	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.closed {
		e.dropped(rec.EventName, "closed")
		e.log.Error().Str("event", rec.EventName).Str("event_id", rec.EventID).Msg("audit emitter closed, event dropped")
		return nil
	}
	select {
	case e.queue <- msg:
	default:
		e.dropped(rec.EventName, "buffer_full")
		e.log.Error().Str("event", rec.EventName).Str("event_id", rec.EventID).Msg("audit buffer full, event dropped")
	}
	return nil
}

// Record builds and validates the record for an event without publishing it.
func (e *NatsEmitter) Record(ctx context.Context, ev Event) (*Record, error) {
	d := ev.Definition
	if d == nil {
		return nil, fmt.Errorf("%w: no definition", ErrInvalidEvent)
	}
	outcome := ev.Outcome
	if outcome == "" {
		outcome = OutcomeSuccess
	}
	if outcome != OutcomeSuccess && ev.Reason == "" {
		return nil, fmt.Errorf("%w: %s %s without reason", ErrInvalidEvent, d.Name, outcome)
	}
	if ev.Target.ID == "" {
		return nil, fmt.Errorf("%w: %s without target id", ErrInvalidEvent, d.Name)
	}

	t := ev.Time
	if t.IsZero() {
		t = e.now()
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	rec := &Record{
		EventID:           id.String(),
		EventName:         d.Name,
		EventCode:         d.Code(outcome),
		SchemaVersion:     SchemaVersion,
		Time:              formatTime(t),
		Outcome:           outcome,
		OutcomeReason:     ev.Reason,
		Severity:          d.severity(outcome),
		TargetType:        d.TargetType,
		TargetID:          ev.Target.ID,
		TargetSpaceID:     ev.Target.SpaceID,
		TargetName:        ev.Target.Name,
		TargetPath:        ev.Target.Path,
		ServiceName:       e.opts.Service.Name,
		ServiceVersion:    e.opts.Service.Version,
		ServiceInstanceID: e.opts.Service.InstanceID,
		OCSFClassUID:      d.OCSFClass,
		OCSFActivityID:    d.activity(outcome),
	}
	if ev.Target.Type != "" {
		rec.TargetType = ev.Target.Type
	}
	if ev.Change != nil {
		rec.ChangeBefore = ev.Change.Before
		rec.ChangeAfter = ev.Change.After
	}

	setActor(ctx, rec, ev.Actor)
	if ev.Tenant != "" {
		rec.TenantID = ev.Tenant
	}

	if rc, ok := ContextGetRequest(ctx); ok {
		rec.ClientAddress = rc.ClientAddress
		rec.SessionID = rc.SessionID
		rec.AuthMethod = rc.AuthMethod
		rec.AuthClientID = rc.ClientID
		rec.UserAgent = rc.UserAgent
	}
	if rec.UserAgent == "" {
		rec.UserAgent = clientUserAgent(ctx)
	}
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		rec.TraceID = sc.TraceID().String()
		rec.SpanID = sc.SpanID().String()
	}
	return rec, nil
}

// clientUserAgent reads the x-user-agent metadata the reva HTTP services set.
// Unlike ctxpkg.ContextGetUserAgentString it never falls back to the gRPC
// user-agent, that is the one of the calling service, e.g. grpc-go/1.84.0.
func clientUserAgent(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	if lst := md.Get(ctxpkg.UserAgentHeader); len(lst) > 0 {
		return lst[0]
	}
	return ""
}

func setActor(ctx context.Context, rec *Record, override *Actor) {
	if override != nil {
		rec.ActorType, rec.ActorID, rec.ActorName = override.Type, override.ID, override.Name
	}
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok || u == nil {
		if override == nil {
			rec.ActorType = ActorAnonymous
		}
		return
	}
	rec.TenantID = u.GetId().GetTenantId()
	if override != nil {
		return
	}
	// a public link acts with the user of its owner, the scope names the link
	if id := PublicShareID(ctx); id != "" {
		rec.ActorType, rec.ActorID = ActorPublicLink, id
		return
	}
	rec.ActorID = u.GetId().GetOpaqueId()
	rec.ActorName = u.GetDisplayName()
	if rec.ActorName == "" {
		rec.ActorName = u.GetUsername()
	}
	switch u.GetId().GetType() {
	case userpb.UserType_USER_TYPE_GUEST, userpb.UserType_USER_TYPE_LIGHTWEIGHT:
		rec.ActorType = ActorGuest
	case userpb.UserType_USER_TYPE_SERVICE, userpb.UserType_USER_TYPE_APPLICATION:
		rec.ActorType = ActorService
	default:
		rec.ActorType = ActorUser
	}
}

func (e *NatsEmitter) publish(ctx context.Context, msg *nats.Msg) error {
	ctx, cancel := context.WithTimeout(ctx, e.opts.PublishTimeout)
	defer cancel()
	start := time.Now()
	err := e.pub.Publish(ctx, msg)
	publishDuration.WithLabelValues(e.opts.Service.Name).Observe(time.Since(start).Seconds())
	if err != nil {
		e.dropped(msg.Header.Get("ce-type"), "publish")
		e.log.Error().Err(err).Str("event_id", msg.Header.Get(nats.MsgIdHdr)).Msg("audit event not stored")
	}
	return err
}

// dropped counts an event that is not stored. The event names come from the
// catalog, so the label stays bounded.
func (e *NatsEmitter) dropped(event, reason string) {
	droppedTotal.WithLabelValues(e.opts.Service.Name, event, reason).Inc()
}

func (e *NatsEmitter) work() {
	defer e.wg.Done()
	for msg := range e.queue {
		// the request is gone, the publish must not be cancelled with it
		_ = e.publish(context.Background(), msg)
	}
}

// Close flushes the buffer. Events emitted afterwards are dropped and counted.
func (e *NatsEmitter) Close() {
	e.once.Do(func() {
		e.mu.Lock()
		e.closed = true
		close(e.queue)
		e.mu.Unlock()
		e.wg.Wait()
	})
}

func message(rec *Record, sc trace.SpanContext) (*nats.Msg, error) {
	data, err := json.Marshal(rec)
	if err != nil {
		return nil, err
	}
	msg := nats.NewMsg(Subject(rec.TenantID, categoryOf(rec.EventName)))
	msg.Data = data
	// CloudEvents NATS binding, binary mode
	msg.Header.Set(nats.MsgIdHdr, rec.EventID)
	msg.Header.Set("ce-specversion", "1.0")
	msg.Header.Set("ce-id", rec.EventID)
	msg.Header.Set("ce-type", rec.EventName)
	msg.Header.Set("ce-source", "/opencloud/"+rec.ServiceName)
	msg.Header.Set("ce-time", rec.Time)
	msg.Header.Set("ce-partitionkey", rec.TenantID)
	msg.Header.Set("content-type", "application/json")
	if sc.IsValid() {
		msg.Header.Set("traceparent", fmt.Sprintf("00-%s-%s-%s", sc.TraceID(), sc.SpanID(), sc.TraceFlags()))
	}
	return msg, nil
}

func categoryOf(name string) string {
	if d, ok := Lookup(name); ok {
		return d.Category
	}
	return "unknown"
}

// Enabled reports whether e stores events, false for nil and Discard. Code
// that has to look something up for an event checks it first.
func Enabled(e Emitter) bool {
	if e == nil {
		return false
	}
	_, off := e.(Discard)
	return !off
}

// CloseEmitter flushes and closes e if it has a buffer, e.g. on shutdown.
func CloseEmitter(e Emitter) {
	if c, ok := e.(interface{ Close() }); ok {
		c.Close()
	}
}

// Discard is an emitter that drops everything, for tests and disabled audit.
type Discard struct{}

// Emit does nothing.
func (Discard) Emit(context.Context, Event) error { return nil }
