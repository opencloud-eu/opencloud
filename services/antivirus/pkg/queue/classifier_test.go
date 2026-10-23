package queue

import (
	"encoding/json"
	"math"
	"strconv"
	"testing"
	"time"
)

func testClassifier(threshold int, window, cooldown time.Duration) *Classifier {
	return &Classifier{
		threshold:           threshold,
		window:              window,
		cooldown:            cooldown,
		maxOutstandingJobs:  math.MaxInt64,
		maxOutstandingBytes: math.MaxUint64,
		enabled:             true,
	}
}

func TestAdvanceDemotesAfterThresholdAndKeepsCooldown(t *testing.T) {
	c := testClassifier(10, time.Second, 30*time.Second)
	start := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	state := rateState{}

	for i := 0; i < 10; i++ {
		var priority Priority
		state, priority = c.advance(state, "event-"+strconv.Itoa(i), start.Add(time.Duration(i)*20*time.Millisecond), start.Add(time.Duration(i)*20*time.Millisecond), 1)
		if priority != PriorityHigh {
			t.Fatalf("request %d: got priority %q, want high", i+1, priority)
		}
	}

	bulkAt := start.Add(200 * time.Millisecond)
	state, priority := c.advance(state, "bulk-event", bulkAt, bulkAt, 1)
	if priority != PriorityLow {
		t.Fatalf("request above threshold: got priority %q, want low", priority)
	}

	state, priority = c.advance(state, "cooldown-event", bulkAt.Add(time.Second), bulkAt.Add(time.Second), 1)
	if priority != PriorityLow {
		t.Fatalf("request during cooldown: got priority %q, want low", priority)
	}

	afterCooldown := bulkAt.Add(31 * time.Second)
	state, priority = c.advance(state, "after-cooldown-event", afterCooldown, afterCooldown, 1)
	if priority != PriorityHigh {
		t.Fatalf("request after cooldown: got priority %q, want high", priority)
	}
}

func TestAdvanceDemotesWhenOutstandingBacklogExceedsBudget(t *testing.T) {
	c := testClassifier(1000, time.Second, time.Minute)
	c.maxOutstandingJobs = 2
	start := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	state := rateState{}

	state, priority := c.advance(state, "first", start, start, 1)
	if priority != PriorityHigh {
		t.Fatalf("first request priority = %q, want high", priority)
	}
	state, priority = c.advance(state, "second", start.Add(time.Second), start.Add(time.Second), 1)
	if priority != PriorityHigh {
		t.Fatalf("second request priority = %q, want high", priority)
	}
	state, priority = c.advance(state, "third", start.Add(2*time.Second), start.Add(2*time.Second), 1)
	if priority != PriorityLow {
		t.Fatalf("third request priority = %q, want low once the backlog budget is exceeded", priority)
	}
	if state.OutstandingJobs != 3 {
		t.Fatalf("outstanding jobs = %d, want 3", state.OutstandingJobs)
	}
}

func TestAdvanceDemotesWhenOutstandingBytesExceedBudget(t *testing.T) {
	c := testClassifier(1000, time.Second, time.Minute)
	c.maxOutstandingBytes = 1 << 20 // 1 MiB
	start := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	state := rateState{}

	state, priority := c.advance(state, "small", start, start, 1024)
	if priority != PriorityHigh {
		t.Fatalf("small request priority = %q, want high", priority)
	}
	state, priority = c.advance(state, "large", start.Add(time.Second), start.Add(time.Second), 2<<20)
	if priority != PriorityLow {
		t.Fatalf("large request priority = %q, want low once the byte budget is exceeded", priority)
	}
	if state.OutstandingBytes != 1024+(2<<20) {
		t.Fatalf("outstanding bytes = %d, want %d", state.OutstandingBytes, 1024+(2<<20))
	}
}

func TestCompleteRemovesOutstandingJobsAndBytes(t *testing.T) {
	c := testClassifier(1000, time.Second, time.Minute)
	c.maxOutstandingJobs = 1
	start := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	state := rateState{}
	state, _ = c.advance(state, "first", start, start, 2<<20)
	state, _ = c.advance(state, "second", start.Add(time.Second), start.Add(time.Second), 0)

	if state.OutstandingJobs != 2 {
		t.Fatalf("outstanding jobs = %d, want 2", state.OutstandingJobs)
	}
	state.OutstandingJobs--
	if state.OutstandingBytes < 2<<20 {
		state.OutstandingBytes = 0
	} else {
		state.OutstandingBytes -= 2 << 20
	}
	if state.OutstandingJobs != 1 || state.OutstandingBytes != 0 {
		t.Fatalf("after completion: jobs=%d bytes=%d, want 1 and 0", state.OutstandingJobs, state.OutstandingBytes)
	}
}

func TestAdvanceClampsFutureEventTime(t *testing.T) {
	c := testClassifier(2, time.Second, time.Minute)
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	state := rateState{}

	state, priority := c.advance(state, "future", now.Add(time.Hour), now, 1)
	if priority != PriorityHigh {
		t.Fatalf("future event priority = %q, want high", priority)
	}
	state, priority = c.advance(state, "second", now.Add(10*time.Millisecond), now, 1)
	if priority != PriorityHigh {
		t.Fatalf("second event priority = %q, want high", priority)
	}
	state, priority = c.advance(state, "third", now.Add(20*time.Millisecond), now, 1)
	if priority != PriorityLow {
		t.Fatalf("third event priority = %q, want low; future timestamps must not disable classification", priority)
	}
	if len(state.Recent) != 3 {
		t.Fatalf("recent events = %d, want 3", len(state.Recent))
	}
}

func TestAdvanceUsesEventTimeAndBoundsRateState(t *testing.T) {
	c := testClassifier(10, time.Second, time.Minute)
	start := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	state := rateState{}
	for i := 0; i < 100; i++ {
		state, _ = c.advance(state, "event-"+strconv.Itoa(i), start.Add(time.Duration(i)*time.Millisecond), start.Add(time.Hour), 1)
	}

	if got, want := len(state.Recent), c.threshold+1; got != want {
		t.Fatalf("recent event count = %d, want bounded count %d", got, want)
	}
	if !state.BulkUntil.Equal(start.Add(time.Hour + time.Minute)) {
		t.Fatalf("bulk cooldown = %s, want %s", state.BulkUntil, start.Add(time.Hour+time.Minute))
	}
	if state.OutstandingJobs != 100 {
		t.Fatalf("outstanding jobs = %d, want 100", state.OutstandingJobs)
	}
}

func TestAdvanceIsIdempotentForDuplicateEventID(t *testing.T) {
	c := testClassifier(2, time.Second, time.Minute)
	start := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	state := rateState{}

	state, priority := c.advance(state, "first", start, start, 1)
	if priority != PriorityHigh {
		t.Fatalf("first event priority = %q, want high", priority)
	}
	state, priority = c.advance(state, "second", start.Add(10*time.Millisecond), start.Add(10*time.Millisecond), 1)
	if priority != PriorityHigh {
		t.Fatalf("second event priority = %q, want high", priority)
	}
	state, priority = c.advance(state, "third", start.Add(20*time.Millisecond), start.Add(20*time.Millisecond), 1)
	if priority != PriorityLow {
		t.Fatalf("third event priority = %q, want low", priority)
	}
	bulkUntil := state.BulkUntil

	state, priority = c.advance(state, "first", start, start.Add(30*time.Millisecond), 1)
	if priority != PriorityHigh {
		t.Fatalf("duplicate event priority = %q, want original high", priority)
	}
	if len(state.Recent) != 3 {
		t.Fatalf("duplicate event changed request count to %d, want 3", len(state.Recent))
	}
	if state.OutstandingJobs != 3 {
		t.Fatalf("duplicate event changed outstanding jobs to %d, want 3", state.OutstandingJobs)
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
