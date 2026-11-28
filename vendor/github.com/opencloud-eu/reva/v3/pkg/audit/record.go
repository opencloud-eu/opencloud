// Package audit emits audit events as described in OpenCloud ADR 0006.
//
// Services describe what happened with an Event and hand it to an Emitter.
// The emitter fills in the actor, the request context, the trace and the
// service, validates the event against its Definition and publishes the
// resulting Record to the JetStream stream "audit".
package audit

import "time"

// SchemaVersion is the version of the Record schema. Changes are additive only.
const SchemaVersion = "1.0"

// TimeFormat is RFC 3339 in UTC with millisecond precision.
const TimeFormat = "2006-01-02T15:04:05.000Z07:00"

// Outcome is the result of the audited action.
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
	OutcomeDenied  Outcome = "denied"
)

// Severity of a record.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityNotice  Severity = "notice"
	SeverityWarning Severity = "warning"
	SeverityAlert   Severity = "alert"
)

// rank orders severities, so an outcome can raise but never lower it.
func (s Severity) rank() int {
	switch s {
	case SeverityNotice:
		return 1
	case SeverityWarning:
		return 2
	case SeverityAlert:
		return 3
	default:
		return 0
	}
}

// ActorType says who acted.
type ActorType string

const (
	ActorUser       ActorType = "user"
	ActorService    ActorType = "service"
	ActorPublicLink ActorType = "public_link"
	ActorGuest      ActorType = "guest"
	ActorAnonymous  ActorType = "anonymous"
	ActorSystem     ActorType = "system"
	ActorCLI        ActorType = "cli"
)

// Record is one audit event as it is written to the stream and the sinks:
// one line of JSON with OpenTelemetry attribute names where they exist.
type Record struct {
	EventID       string `json:"event.id"`
	EventName     string `json:"event.name"`
	EventCode     string `json:"event.code"`
	SchemaVersion string `json:"schema_version"`
	Time          string `json:"time"`

	Outcome       Outcome  `json:"outcome"`
	OutcomeReason string   `json:"outcome.reason,omitempty"`
	Severity      Severity `json:"severity"`

	TenantID string `json:"tenant.id"`

	ActorType ActorType `json:"actor.type"`
	ActorID   string    `json:"actor.id,omitempty"`
	ActorName string    `json:"actor.name,omitempty"`

	AuthMethod   string `json:"auth.method,omitempty"`
	AuthClientID string `json:"auth.client_id,omitempty"`
	SessionID    string `json:"session.id,omitempty"`

	ClientAddress string `json:"client.address,omitempty"`
	UserAgent     string `json:"user_agent.original,omitempty"`

	TargetType    string `json:"target.type"`
	TargetID      string `json:"target.id"`
	TargetSpaceID string `json:"target.space.id,omitempty"`
	TargetName    string `json:"target.name,omitempty"`
	TargetPath    string `json:"target.path,omitempty"`

	ChangeBefore map[string]any `json:"change.before,omitempty"`
	ChangeAfter  map[string]any `json:"change.after,omitempty"`

	TraceID string `json:"trace_id,omitempty"`
	SpanID  string `json:"span_id,omitempty"`

	ServiceName       string `json:"service.name"`
	ServiceVersion    string `json:"service.version,omitempty"`
	ServiceInstanceID string `json:"service.instance.id,omitempty"`

	OCSFClassUID   int `json:"ocsf.class_uid"`
	OCSFActivityID int `json:"ocsf.activity_id"`
}

func formatTime(t time.Time) string {
	return t.UTC().Format(TimeFormat)
}
