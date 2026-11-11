package svc

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/opencloud-eu/reva/v2/pkg/audit"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
)

// emitAudit reports audit events. Failures are counted and logged by the
// emitter; in strict mode the error is returned to the caller.
func (g Graph) emitAudit(ctx context.Context, evs ...audit.Event) error {
	if g.auditEmitter == nil {
		return nil
	}
	for _, ev := range evs {
		if err := g.auditEmitter.Emit(ctx, ev); err != nil {
			return err
		}
	}
	return nil
}

// auditEvents returns the audit events for a domain event graph publishes.
// Graph publishes a domain event after every successful user, group and role
// change, so this covers all of their success paths.
func auditEvents(ev any) []audit.Event {
	switch e := ev.(type) {
	case events.UserCreated:
		return []audit.Event{{Definition: audit.UserCreate, Target: audit.Target{ID: e.UserID}}}
	case events.UserDeleted:
		return []audit.Event{{Definition: audit.UserDelete, Target: audit.Target{ID: e.UserID}}}
	case events.UserSoftDeleted:
		return []audit.Event{{Definition: audit.UserSoftDelete, Target: audit.Target{ID: e.UserID}}}
	case events.UserFeatureChanged:
		return userFeatureEvents(e)
	case events.GroupCreated:
		return []audit.Event{{Definition: audit.GroupCreate, Target: audit.Target{ID: e.GroupID}}}
	case events.GroupDeleted:
		return []audit.Event{{Definition: audit.GroupDelete, Target: audit.Target{ID: e.GroupID}}}
	case events.GroupMemberAdded:
		return []audit.Event{{Definition: audit.GroupMemberAdd, Target: audit.Target{ID: e.GroupID},
			Change: &audit.Change{After: map[string]any{"member": e.UserID}}}}
	case events.GroupMemberRemoved:
		return []audit.Event{{Definition: audit.GroupMemberRemove, Target: audit.Target{ID: e.GroupID},
			Change: &audit.Change{Before: map[string]any{"member": e.UserID}}}}
	default:
		return nil
	}
}

// userFeatureEvents splits a feature change into the audit events it stands for.
func userFeatureEvents(e events.UserFeatureChanged) []audit.Event {
	target := audit.Target{ID: e.UserID}
	var evs []audit.Event
	update := &audit.Change{Before: map[string]any{}, After: map[string]any{}}
	for _, f := range e.Features {
		switch f.Name {
		case "password":
			// the user changed their own password
			evs = append(evs, audit.Event{Definition: audit.UserPasswordChange, Target: target})
		case "passwordChanged":
			// an admin set the password of the user
			evs = append(evs, audit.Event{Definition: audit.UserPasswordReset, Target: target})
		case "accountEnabled":
			d := audit.UserDisable
			if f.Value == "true" {
				d = audit.UserEnable
			}
			evs = append(evs, audit.Event{Definition: d, Target: target})
		case "roleChanged":
			ch := &audit.Change{After: map[string]any{"app_role": f.Value}}
			if f.OldValue != nil && *f.OldValue != "" {
				ch.Before = map[string]any{"app_role": *f.OldValue}
			}
			evs = append(evs, audit.Event{Definition: audit.AppRoleAssign, Target: target, Change: ch})
		default:
			update.After[f.Name] = f.Value
			if f.OldValue != nil {
				update.Before[f.Name] = *f.OldValue
			}
		}
	}
	if len(update.After) > 0 {
		if len(update.Before) == 0 {
			update.Before = nil
		}
		evs = append(evs, audit.Event{Definition: audit.UserUpdate, Target: target, Change: update})
	}
	return evs
}

// auditOnError reports failed and denied requests of a route. Successes are
// reported where graph publishes its domain event, so they are not counted
// twice. It must run before requireAdmin, which refuses non-admins.
func (g Graph) auditOnError(d *audit.Definition, idParam string) func(http.Handler) http.Handler {
	return g.auditRoute(d, idParam, false)
}

// auditAlways reports every request of a route, for actions without a domain event.
func (g Graph) auditAlways(d *audit.Definition, idParam string) func(http.Handler) http.Handler {
	return g.auditRoute(d, idParam, true)
}

func (g Graph) auditRoute(d *audit.Definition, idParam string, success bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			outcome, reason := outcomeOf(ww.Status())
			if outcome == audit.OutcomeSuccess && !success {
				return
			}
			target := audit.Target{ID: "unknown"}
			switch {
			case idParam != "":
				target.ID = chi.URLParam(r, idParam)
			case d == audit.UserPasswordChange:
				if u, ok := revactx.ContextGetUser(r.Context()); ok {
					target.ID = u.GetId().GetOpaqueId()
				}
			}
			if err := g.emitAudit(r.Context(), audit.Event{Definition: d, Outcome: outcome, Reason: reason, Target: target}); err != nil {
				g.logger.Error().Err(err).Str("event", d.Name).Msg("could not emit audit event")
			}
		})
	}
}

// outcomeOf maps an HTTP status to the audit outcome and reason.
func outcomeOf(status int) (audit.Outcome, string) {
	switch {
	case status == 0 || status < 400:
		return audit.OutcomeSuccess, ""
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return audit.OutcomeDenied, "permission_denied"
	case status == http.StatusNotFound:
		return audit.OutcomeFailure, "not_found"
	case status == http.StatusConflict:
		return audit.OutcomeFailure, "conflict"
	case status < 500:
		return audit.OutcomeFailure, "invalid_request"
	default:
		return audit.OutcomeFailure, "error"
	}
}
