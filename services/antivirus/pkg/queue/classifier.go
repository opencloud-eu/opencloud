package queue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// Priority identifies the antivirus queue a scan is published to.
type Priority string

const (
	// PriorityHigh is the lane used for interactive/light uploads.
	PriorityHigh Priority = "high"
	// PriorityLow is the lane used for heavy/bulk uploads.
	PriorityLow Priority = "low"
)

const (
	classifierCASRetries = 32
	classifierOpTimeout  = 5 * time.Second
)

type rateRequest struct {
	ID       string    `json:"id,omitempty"`
	At       time.Time `json:"at"`
	Bytes    uint64    `json:"bytes,omitempty"`
	Priority Priority  `json:"priority,omitempty"`
}

type rateState struct {
	Recent           []rateRequest `json:"recent"`
	BulkUntil        time.Time     `json:"bulk_until"`
	OutstandingJobs  int64         `json:"outstanding_jobs,omitempty"`
	OutstandingBytes uint64        `json:"outstanding_bytes,omitempty"`
}

// UnmarshalJSON accepts rate states written before event IDs, sizes and the
// outstanding backlog counters were stored.
func (s *rateState) UnmarshalJSON(data []byte) error {
	var stored struct {
		Recent           json.RawMessage `json:"recent"`
		BulkUntil        time.Time       `json:"bulk_until"`
		OutstandingJobs  int64           `json:"outstanding_jobs"`
		OutstandingBytes uint64          `json:"outstanding_bytes"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}
	s.BulkUntil = stored.BulkUntil
	s.OutstandingJobs = stored.OutstandingJobs
	s.OutstandingBytes = stored.OutstandingBytes
	if len(stored.Recent) == 0 || string(stored.Recent) == "null" {
		s.Recent = nil
		return nil
	}
	if err := json.Unmarshal(stored.Recent, &s.Recent); err == nil {
		return nil
	}

	var legacy []time.Time
	if err := json.Unmarshal(stored.Recent, &legacy); err != nil {
		return err
	}
	s.Recent = make([]rateRequest, 0, len(legacy))
	for _, at := range legacy {
		s.Recent = append(s.Recent, rateRequest{At: at})
	}
	return nil
}

// Classifier uses a shared JetStream KV bucket so all antivirus replicas apply
// the same per-user threshold. A user is demoted when either their request
// rate inside the window exceeds the threshold, their outstanding scan backlog
// exceeds the configured job or byte budget, or they are still inside a
// cooldown started by an earlier burst.
type Classifier struct {
	store               jetstream.KeyValue
	threshold           int
	window              time.Duration
	cooldown            time.Duration
	maxOutstandingJobs  int64
	maxOutstandingBytes uint64
	enabled             bool
}

func newClassifier(store jetstream.KeyValue, threshold int, window, cooldown time.Duration, maxOutstandingJobs int64, maxOutstandingBytes uint64, enabled bool) *Classifier {
	return &Classifier{
		store:               store,
		threshold:           threshold,
		window:              window,
		cooldown:            cooldown,
		maxOutstandingJobs:  maxOutstandingJobs,
		maxOutstandingBytes: maxOutstandingBytes,
		enabled:             enabled,
	}
}

// Classify records one scan request and returns its priority. Requests beyond
// the rate threshold, the outstanding job budget or the outstanding byte
// budget are queued at low priority.
func (c *Classifier) Classify(ctx context.Context, userID, eventID string, eventTime, now time.Time, size uint64) (Priority, error) {
	if userID == "" || !c.enabled {
		// Keep system-generated uploads and the disabled mode working normally.
		return PriorityHigh, nil
	}
	if eventTime.IsZero() {
		eventTime = now
	}

	key := classifierKey(userID)
	for range classifierCASRetries {
		state, revision, found, err := c.load(ctx, key)
		if err != nil {
			return "", err
		}
		state, priority := c.advance(state, eventID, eventTime, now, size)
		data, err := json.Marshal(state)
		if err != nil {
			return "", err
		}
		if !found {
			if _, err := c.store.Create(ctx, key, data); errors.Is(err, jetstream.ErrKeyExists) {
				continue
			} else if err != nil {
				return "", fmt.Errorf("create antivirus rate state: %w", err)
			}
			return priority, nil
		}
		if _, err := c.store.Update(ctx, key, data, revision); errors.Is(err, jetstream.ErrKeyRevisionMismatch) {
			continue
		} else if err != nil {
			return "", fmt.Errorf("update antivirus rate state: %w", err)
		}
		return priority, nil
	}

	return "", errors.New("antivirus rate state remained contended")
}

// CurrentPriority reports the user's current class without recording a new
// request. It lets workers demote high-lane jobs that were queued before the
// threshold or backlog crossing was observable.
func (c *Classifier) CurrentPriority(ctx context.Context, userID string, now time.Time) (Priority, error) {
	if userID == "" || !c.enabled {
		return PriorityHigh, nil
	}
	state, _, found, err := c.load(ctx, classifierKey(userID))
	if err != nil {
		return "", err
	}
	if !found {
		return PriorityHigh, nil
	}
	if now.Before(state.BulkUntil) ||
		state.OutstandingJobs > c.maxOutstandingJobs ||
		state.OutstandingBytes > c.maxOutstandingBytes {
		return PriorityLow, nil
	}
	cutoff := now.Add(-c.window)
	count := 0
	for _, request := range state.Recent {
		if !request.At.Before(cutoff) {
			count++
		}
	}
	if count > c.threshold {
		return PriorityLow, nil
	}
	return PriorityHigh, nil
}

// Complete removes one finished scan from the user's outstanding backlog.
func (c *Classifier) Complete(ctx context.Context, userID string, size uint64) error {
	if userID == "" || !c.enabled {
		return nil
	}
	key := classifierKey(userID)
	for range classifierCASRetries {
		state, revision, found, err := c.load(ctx, key)
		if err != nil {
			return err
		}
		if !found {
			return nil
		}
		if state.OutstandingJobs > 0 {
			state.OutstandingJobs--
		}
		if state.OutstandingBytes >= size {
			state.OutstandingBytes -= size
		} else {
			state.OutstandingBytes = 0
		}
		data, err := json.Marshal(state)
		if err != nil {
			return err
		}
		if _, err := c.store.Update(ctx, key, data, revision); errors.Is(err, jetstream.ErrKeyRevisionMismatch) {
			continue
		} else if errors.Is(err, jetstream.ErrKeyNotFound) {
			return nil
		} else if err != nil {
			return fmt.Errorf("update antivirus rate state: %w", err)
		}
		return nil
	}
	return errors.New("antivirus rate state remained contended")
}

// load reads the per-user state. A corrupt value is deleted and rebuilt so a
// single bad record cannot wedge the service.
func (c *Classifier) load(ctx context.Context, key string) (rateState, uint64, bool, error) {
	entry, err := c.store.Get(ctx, key)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return rateState{}, 0, false, nil
	}
	if err != nil {
		return rateState{}, 0, false, fmt.Errorf("read antivirus rate state: %w", err)
	}
	var state rateState
	if err := json.Unmarshal(entry.Value(), &state); err != nil {
		if delErr := c.store.Delete(ctx, key); delErr != nil && !errors.Is(delErr, jetstream.ErrKeyNotFound) {
			return rateState{}, 0, false, fmt.Errorf("delete corrupt antivirus rate state: %w", delErr)
		}
		return rateState{}, 0, false, nil
	}
	return state, entry.Revision(), true, nil
}

func (c *Classifier) advance(state rateState, eventID string, eventTime, now time.Time, size uint64) (rateState, Priority) {
	// Clamp future timestamps so one skewed producer cannot disable
	// classification for a user until the state expires.
	if eventTime.After(now) {
		eventTime = now
	}
	latestEvent := eventTime
	for _, request := range state.Recent {
		if request.At.After(latestEvent) {
			latestEvent = request.At
		}
	}
	cutoff := latestEvent.Add(-c.window)
	recent := state.Recent[:0]
	var duplicatePriority Priority
	for _, request := range state.Recent {
		if !request.At.Before(cutoff) {
			recent = append(recent, request)
			if eventID != "" && request.ID == eventID {
				duplicatePriority = request.Priority
			}
		}
	}
	state.Recent = recent
	if duplicatePriority != "" {
		return state, duplicatePriority
	}

	// The outstanding backlog counts every accepted request, independent of
	// the rate window. This catches bulk uploads that stay below the rate
	// threshold but build a large queue.
	if state.OutstandingJobs < math.MaxInt64 {
		state.OutstandingJobs++
	}
	if state.OutstandingBytes <= math.MaxUint64-size {
		state.OutstandingBytes += size
	} else {
		state.OutstandingBytes = math.MaxUint64
	}

	newRequest := !eventTime.Before(cutoff)
	if newRequest {
		state.Recent = append(state.Recent, rateRequest{ID: eventID, At: eventTime, Bytes: size})
		sort.Slice(state.Recent, func(i, j int) bool {
			if state.Recent[i].At.Equal(state.Recent[j].At) {
				return state.Recent[i].ID < state.Recent[j].ID
			}
			return state.Recent[i].At.Before(state.Recent[j].At)
		})
		if maxRecent := c.threshold + 1; len(state.Recent) > maxRecent {
			state.Recent = state.Recent[len(state.Recent)-maxRecent:]
		}
		if len(state.Recent) > c.threshold {
			state.BulkUntil = now.Add(c.cooldown)
		}
	}

	priority := PriorityHigh
	switch {
	case now.Before(state.BulkUntil),
		state.OutstandingJobs > c.maxOutstandingJobs,
		state.OutstandingBytes > c.maxOutstandingBytes:
		priority = PriorityLow
	}
	for index := range state.Recent {
		if eventID != "" && state.Recent[index].ID == eventID {
			state.Recent[index].Priority = priority
			break
		}
	}
	return state, priority
}

func classifierKey(userID string) string {
	sum := sha256.Sum256([]byte(userID))
	return hex.EncodeToString(sum[:])
}
