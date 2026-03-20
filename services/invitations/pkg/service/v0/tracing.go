package service

import (
	"context"

	"github.com/opencloud-eu/opencloud/services/invitations/pkg/invitations"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// NewTracing returns a service that instruments traces.
func NewTracing(next Service, tp trace.TracerProvider) Service {
	return tracing{
		next: next,
		tp:   tp,
	}
}

type tracing struct {
	next Service
	tp   trace.TracerProvider
}

func (t tracing) List(ctx context.Context, userId string) ([]*invitations.Invitation, error) {
	spanOpts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.KeyValue{
				Key: "invitation", Value: attribute.StringValue("list"),
			}),
	}
	ctx, span := t.tp.Tracer("invitations").Start(ctx, "List", spanOpts...)
	defer span.End()

	return t.next.List(ctx, userId)
}

func (t tracing) GetByInvitedEmail(ctx context.Context, email string) (*invitations.Invitation, error) {
	spanOpts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.KeyValue{
				Key: "invitation", Value: attribute.StringValue("get"),
			}),
	}
	ctx, span := t.tp.Tracer("invitations").Start(ctx, "Get", spanOpts...)
	defer span.End()

	return t.next.GetByInvitedEmail(ctx, email)
}

func (t tracing) GetByInviteId(ctx context.Context, id string) (*invitations.Invitation, error) {
	spanOpts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.KeyValue{
				Key: "invitation", Value: attribute.StringValue("get"),
			}),
	}
	ctx, span := t.tp.Tracer("invitations").Start(ctx, "Get", spanOpts...)
	defer span.End()

	return t.next.GetByInviteId(ctx, id)
}

// Invite implements the Service interface.
func (t tracing) Invite(ctx context.Context, invitation *invitations.Invitation) (*invitations.Invitation, error) {
	spanOpts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.KeyValue{
				Key: "invitation", Value: attribute.StringValue(invitation.InvitedUserEmailAddress),
			}),
	}
	ctx, span := t.tp.Tracer("invitations").Start(ctx, "Invite", spanOpts...)
	defer span.End()

	return t.next.Invite(ctx, invitation)
}
