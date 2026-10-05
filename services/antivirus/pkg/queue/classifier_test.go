package queue

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"
)

func TestAdvanceDemotesAfterThresholdAndKeepsCooldown(t *testing.T) {
	c := &Classifier{threshold: 10, window: time.Second, cooldown: 30 * time.Second}
	start := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	state := rateState{}

	for i := 0; i < 10; i++ {
		var priority Priority
		state, priority = c.advance(state, "event-"+strconv.Itoa(i), start.Add(time.Duration(i)*20*time.Millisecond), start.Add(time.Duration(i)*20*time.Millisecond))
		if priority != PriorityHigh {
			t.Fatalf("request %d: got priority %q, want high", i+1, priority)
		}
	}

	bulkAt := start.Add(200 * time.Millisecond)
	state, priority := c.advance(state, "bulk-event", bulkAt, bulkAt)
	if priority != PriorityLow {
		t.Fatalf("request above threshold: got priority %q, want low", priority)
	}

	state, priority = c.advance(state, "cooldown-event", bulkAt.Add(time.Second), bulkAt.Add(time.Second))
	if priority != PriorityLow {
		t.Fatalf("request during cooldown: got priority %q, want low", priority)
	}

	afterCooldown := bulkAt.Add(31 * time.Second)
	state, priority = c.advance(state, "after-cooldown-event", afterCooldown, afterCooldown)
	if priority != PriorityHigh {
		t.Fatalf("request after cooldown: got priority %q, want high", priority)
	}
}

func TestAdvanceUsesEventTimeAndBoundsRateState(t *testing.T) {
	c := &Classifier{threshold: 10, window: time.Second, cooldown: time.Minute}
	start := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	state := rateState{}
	for i := 0; i < 100; i++ {
		state, _ = c.advance(state, "event-"+strconv.Itoa(i), start.Add(time.Duration(i)*time.Millisecond), start.Add(time.Hour))
	}

	if got, want := len(state.Recent), c.threshold+1; got != want {
		t.Fatalf("recent event count = %d, want bounded count %d", got, want)
	}
	if !state.BulkUntil.Equal(start.Add(time.Hour + time.Minute)) {
		t.Fatalf("bulk cooldown = %s, want %s", state.BulkUntil, start.Add(time.Hour+time.Minute))
	}
}

func TestAdvanceIsIdempotentForDuplicateEventID(t *testing.T) {
	c := &Classifier{threshold: 2, window: time.Second, cooldown: time.Minute}
	start := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	state := rateState{}

	state, priority := c.advance(state, "first", start, start)
	if priority != PriorityHigh {
		t.Fatalf("first event priority = %q, want high", priority)
	}
	state, priority = c.advance(state, "second", start.Add(10*time.Millisecond), start.Add(10*time.Millisecond))
	if priority != PriorityHigh {
		t.Fatalf("second event priority = %q, want high", priority)
	}
	state, priority = c.advance(state, "third", start.Add(20*time.Millisecond), start.Add(20*time.Millisecond))
	if priority != PriorityLow {
		t.Fatalf("third event priority = %q, want low", priority)
	}
	bulkUntil := state.BulkUntil

	state, priority = c.advance(state, "first", start, start.Add(30*time.Millisecond))
	if priority != PriorityHigh {
		t.Fatalf("duplicate event priority = %q, want original high", priority)
	}
	if len(state.Recent) != 3 {
		t.Fatalf("duplicate event changed request count to %d, want 3", len(state.Recent))
	}
	if !state.BulkUntil.Equal(bulkUntil) {
		t.Fatalf("duplicate event extended cooldown to %s, want %s", state.BulkUntil, bulkUntil)
	}
}

func TestRateStateUnmarshalLegacyRecentTimestamps(t *testing.T) {
	legacy, err := json.Marshal(struct {
		Recent    []time.Time `json:"recent"`
		BulkUntil time.Time   `json:"bulk_until"`
	}{Recent: []time.Time{time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)}})
	if err != nil {
		t.Fatal(err)
	}

	var state rateState
	if err := json.Unmarshal(legacy, &state); err != nil {
		t.Fatal(err)
	}
	if len(state.Recent) != 1 || state.Recent[0].ID != "" {
		t.Fatalf("legacy recent timestamps = %#v, want one migrated timestamp", state.Recent)
	}
	if !state.Recent[0].At.Equal(time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("migrated timestamp = %s, want original timestamp", state.Recent[0].At)
	}
}

func TestClassifierKeyHidesUserIdentity(t *testing.T) {
	const userID = "user-idp\x00opaque-user-id"
	key := classifierKey(userID)
	if key == userID || len(key) != 64 {
		t.Fatalf("classifier key should be an opaque SHA-256 hex value, got %q", key)
	}
	if key == classifierKey("another-user") {
		t.Fatal("different users produced the same classifier key")
	}
}
