package eventsmiddleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/audit"
	"github.com/opencloud-eu/reva/v3/pkg/storagespace"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
	"go.opentelemetry.io/otel/trace"
)

// linkEvent audits the calls on public links, also the token lookups that
// open a link.
func linkEvent(req, res interface{}, _ *storagespace.ItemSlot) (audit.Event, bool) {
	var ev audit.Event
	switch v := res.(type) {
	case *link.CreatePublicShareResponse:
		r := req.(*link.CreatePublicShareRequest)
		ev = audit.Event{Definition: audit.PublicLinkCreate, Target: shareTarget(v.GetShare().GetId().GetOpaqueId(), r.GetResourceInfo().GetId(), r.GetResourceInfo().GetName())}
		ev.Target.Type = "public_link"
		ev.Change = linkCreated(r, v)
	case *link.UpdatePublicShareResponse:
		r := req.(*link.UpdatePublicShareRequest)
		ev = audit.Event{Definition: audit.PublicLinkUpdate, Target: shareTarget(r.GetRef().GetId().GetOpaqueId(), v.GetShare().GetResourceId(), utils.ReadPlainFromOpaque(v.GetOpaque(), "resourcename"))}
		ev.Change = linkUpdated(r, v)
	case *link.GetPublicShareByTokenResponse:
		return publicLinkAccess(req.(*link.GetPublicShareByTokenRequest), v)
	case *link.RemovePublicShareResponse:
		r := req.(*link.RemovePublicShareRequest)
		var rid *provider.ResourceId
		_ = utils.ReadJSONFromOpaque(v.GetOpaque(), "resourceid", &rid)
		ev = audit.Event{Definition: audit.PublicLinkDelete, Target: shareTarget(r.GetRef().GetId().GetOpaqueId(), rid, utils.ReadPlainFromOpaque(v.GetOpaque(), "resourcename"))}
		ev.Change = linkRemoved(v.GetOpaque())
	default:
		return ev, false
	}
	return ev, true
}

// publicLinkAccess audits a token lookup. The share ID identifies the link;
// when the lookup failed there is none and a hash of the token is used, never
// the token itself. A denial without any credentials is the web client asking
// whether a link needs a password, not an attempt, and is left out.
func publicLinkAccess(r *link.GetPublicShareByTokenRequest, res *link.GetPublicShareByTokenResponse) (audit.Event, bool) {
	ev := audit.Event{Definition: audit.PublicLinkAccess}
	ev.Outcome, ev.Reason = outcome(res.GetStatus())

	if s := res.GetShare(); s.GetId().GetOpaqueId() != "" {
		ev.Target = shareTarget(s.GetId().GetOpaqueId(), s.GetResourceId(), "")
	} else {
		ev.Target = audit.Target{ID: "token:" + tokenHash(r.GetToken())}
	}
	ev.Target.Type = "public_link"
	ev.Actor = &audit.Actor{Type: audit.ActorPublicLink, ID: ev.Target.ID}

	if ev.Outcome == audit.OutcomeDenied {
		switch {
		case r.GetAuthentication().GetPassword() != "":
			ev.Reason = "invalid_password"
		// ocdav sends an empty signature when the request has no credentials
		case r.GetAuthentication().GetSignature().GetSignature() != "":
			ev.Reason = "invalid_signature"
		default:
			return ev, false
		}
	}
	return ev, true
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:8])
}

// accessWindow is how long repeated successful accesses of a link by the same
// client count as one access. Opening a link makes many requests, and every
// one of them looks the token up again.
const accessWindow = 30 * time.Minute

// failureWindow is how long a failed access is remembered. The proxy and ocdav
// check the credentials of the same request one after the other.
const failureWindow = 10 * time.Second

// maxRemembered limits the keys of one onceWithin. When it is full, an access
// is reported each time instead of growing the map.
const maxRemembered = 10000

// onceWithin remembers keys for a while, to report each access only once.
type onceWithin struct {
	mu        sync.Mutex
	window    time.Duration
	max       int
	seen      map[string]time.Time
	nextSweep time.Time
}

func newOnceWithin(window time.Duration, max int) *onceWithin {
	return &onceWithin{window: window, max: max, seen: map[string]time.Time{}}
}

// first reports whether key was not seen within the window, and remembers it.
// Expired keys are removed once per window, not on every call.
func (o *onceWithin) first(key string, now time.Time) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if t, ok := o.seen[key]; ok && now.Sub(t) < o.window {
		return false
	}
	if !now.Before(o.nextSweep) {
		for k, t := range o.seen {
			if now.Sub(t) >= o.window {
				delete(o.seen, k)
			}
		}
		o.nextSweep = now.Add(o.window)
	}
	if len(o.seen) < o.max {
		o.seen[key] = now
	}
	return true
}

// linkAccesses remembers successful and failed link accesses apart, failures
// only for a few seconds: someone trying passwords makes a new one each time.
type linkAccesses struct {
	successes *onceWithin
	failures  *onceWithin
}

func newLinkAccesses() *linkAccesses {
	return &linkAccesses{
		successes: newOnceWithin(accessWindow, maxRemembered),
		failures:  newOnceWithin(failureWindow, maxRemembered),
	}
}

// firstLinkAccess drops repeated successful accesses of a link by the same
// client, and successful ones without a client request. A failed one counts
// once per request: the proxy and ocdav both check the credentials of a
// request to public-files.
func firstLinkAccess(ctx context.Context, a *linkAccesses, ev audit.Event) bool {
	if ev.Definition != audit.PublicLinkAccess {
		return true
	}
	if ev.Outcome != audit.OutcomeSuccess {
		tid := trace.SpanContextFromContext(ctx).TraceID()
		if !tid.IsValid() {
			return true
		}
		return a.failures.first(ev.Target.ID+"|"+ev.Reason+"|"+tid.String(), time.Now())
	}
	rc, ok := audit.ContextGetRequest(ctx)
	if !ok || rc.IsZero() {
		// a service resolving the link, e.g. after an upload, is no client
		return false
	}
	return a.successes.first(ev.Target.ID+"|"+rc.ClientAddress+"|"+rc.SessionID, time.Now())
}
