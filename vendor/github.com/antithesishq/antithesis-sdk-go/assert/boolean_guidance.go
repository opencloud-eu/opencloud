//go:build enable_antithesis_sdk

package assert

import (
	"sync"

	"github.com/antithesishq/antithesis-sdk-go/internal"
)

// TODO: Someday this tracker should also start deduplicating
// guidance, but there are some complicated policy questions
// to settle before we do that.
type booleanGuidance struct{}

// booleanGuidanceTracker keeps the per-id entries: a grow-only map with
// stable keys, read on every rich-assert evaluation and written once per
// guidance id.
var booleanGuidanceTracker sync.Map // message key -> *booleanGuidance

func getBooleanGuidanceEntry(messageKey string) *booleanGuidance {
	if entry, ok := booleanGuidanceTracker.Load(messageKey); ok {
		return entry.(*booleanGuidance)
	}
	entry, _ := booleanGuidanceTracker.LoadOrStore(messageKey, &booleanGuidance{})
	return entry.(*booleanGuidance)
}

func (tI *booleanGuidance) send_value(bgI *booleanGuidanceInfo) {
	if tI == nil {
		return
	}

	emitBooleanGuidance(bgI)
}

func emitBooleanGuidance(bgI *booleanGuidanceInfo) error {
	return internal.Json_data(map[string]any{"antithesis_guidance": bgI})
}
