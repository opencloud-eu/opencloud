package queue

import (
	"testing"
	"time"
)

func TestAdvanceDemotesAfterThresholdAndKeepsCooldown(t *testing.T) {
	c := &Classifier{threshold: 10, window: time.Second, cooldown: 30 * time.Second}
	start := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	state := rateState{}

	for i := 0; i < 10; i++ {
		var priority Priority
		state, priority = c.advance(state, start.Add(time.Duration(i)*20*time.Millisecond), start.Add(time.Duration(i)*20*time.Millisecond))
		if priority != PriorityHigh {
			t.Fatalf("request %d: got priority %q, want high", i+1, priority)
		}
	}

	bulkAt := start.Add(200 * time.Millisecond)
	state, priority := c.advance(state, bulkAt, bulkAt)
	if priority != PriorityLow {
		t.Fatalf("request above threshold: got priority %q, want low", priority)
	}

	state, priority = c.advance(state, bulkAt.Add(time.Second), bulkAt.Add(time.Second))
	if priority != PriorityLow {
		t.Fatalf("request during cooldown: got priority %q, want low", priority)
	}

	afterCooldown := bulkAt.Add(31 * time.Second)
	state, priority = c.advance(state, afterCooldown, afterCooldown)
	if priority != PriorityHigh {
		t.Fatalf("request after cooldown: got priority %q, want high", priority)
	}
}

func TestAdvanceUsesEventTimeAndBoundsRateState(t *testing.T) {
	c := &Classifier{threshold: 10, window: time.Second, cooldown: time.Minute}
	start := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	state := rateState{}
	for i := 0; i < 100; i++ {
		state, _ = c.advance(state, start.Add(time.Duration(i)*time.Millisecond), start.Add(time.Hour))
	}

	if got, want := len(state.Recent), c.threshold+1; got != want {
		t.Fatalf("recent event count = %d, want bounded count %d", got, want)
	}
	if !state.BulkUntil.Equal(start.Add(time.Hour + time.Minute)) {
		t.Fatalf("bulk cooldown = %s, want %s", state.BulkUntil, start.Add(time.Hour+time.Minute))
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
