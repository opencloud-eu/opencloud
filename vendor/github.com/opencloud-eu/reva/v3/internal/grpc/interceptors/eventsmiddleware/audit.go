package eventsmiddleware

import (
	"context"
	"strings"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/reva/v3/pkg/audit"
	"github.com/opencloud-eu/reva/v3/pkg/events/raw"
	"github.com/opencloud-eu/reva/v3/pkg/storagespace"
)

// The audit events of the CS3 calls, one file per area: files, folders, trash
// bin and versions in audit_files.go, shares and space members in
// audit_share.go, public links in audit_link.go and spaces in audit_space.go.

// auditConfig is the audit part of the middleware config.
type auditConfig struct {
	Enabled bool   `mapstructure:"audit_enabled"`
	Mode    string `mapstructure:"audit_mode"`
	Service string `mapstructure:"audit_service"`
	Version string `mapstructure:"audit_service_version"`
}

// emitterFromConfig returns the audit emitter, a discarding one when audit is off.
func emitterFromConfig(m map[string]interface{}) (audit.Emitter, error) {
	var ac auditConfig
	if err := mapstructure.Decode(m, &ac); err != nil {
		return nil, err
	}
	var nc raw.Config
	if err := mapstructure.Decode(m, &nc); err != nil {
		return nil, err
	}
	name, _ := m["name"].(string)
	return audit.EmitterFromConfig(context.Background(), name+"-audit", audit.EmitterConfig{
		Enabled:        ac.Enabled,
		Mode:           ac.Mode,
		Service:        ac.Service,
		ServiceVersion: ac.Version,
		Nats:           nc,
	}, audit.Options{})
}

// outcome maps a CS3 status to the audit outcome and reason.
func outcome(s *rpc.Status) (audit.Outcome, string) {
	switch s.GetCode() {
	case rpc.Code_CODE_OK:
		return audit.OutcomeSuccess, ""
	case rpc.Code_CODE_PERMISSION_DENIED:
		return audit.OutcomeDenied, "permission_denied"
	case rpc.Code_CODE_UNAUTHENTICATED:
		return audit.OutcomeDenied, "unauthenticated"
	default:
		return audit.OutcomeFailure, strings.ToLower(strings.TrimPrefix(s.GetCode().String(), "CODE_"))
	}
}

// auditEvent turns a CS3 call into an audit event. item is filled by the
// storage driver with the type, ID and name of the item the call acts on. The
// type is needed for move, delete, restore and purge; when it is unknown,
// e.g. because the item doesn't exist, no event is emitted for these calls.
func auditEvent(req, res interface{}, item *storagespace.ItemSlot) (audit.Event, bool) {
	if item == nil {
		item = &storagespace.ItemSlot{}
	}
	// every area knows its own calls and returns false for all others
	for _, area := range []func(req, res interface{}, item *storagespace.ItemSlot) (audit.Event, bool){
		fileEvent, shareEvent, linkEvent, spaceEvent,
	} {
		ev, ok := area(req, res, item)
		if !ok {
			continue
		}
		if s, ok := res.(su); ok && ev.Outcome == "" {
			ev.Outcome, ev.Reason = outcome(s.GetStatus())
		}
		if ev.Target.ID == "" {
			// a failed call may not tell us more than this
			ev.Target.ID = "unknown"
		}
		return ev, true
	}
	return audit.Event{}, false
}
