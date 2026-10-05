package queue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// Priority identifies the antivirus queue a scan is published to.
type Priority string

const (
	PriorityHigh Priority = "high"
	PriorityLow  Priority = "low"
)

type rateRequest struct {
	ID       string    `json:"id,omitempty"`
	At       time.Time `json:"at"`
	Priority Priority  `json:"priority,omitempty"`
}

type rateState struct {
	Recent    []rateRequest `json:"recent"`
	BulkUntil time.Time     `json:"bulk_until"`
}

// UnmarshalJSON accepts rate states written before event IDs were stored.
func (s *rateState) UnmarshalJSON(data []byte) error {
	var stored struct {
		Recent    json.RawMessage `json:"recent"`
		BulkUntil time.Time       `json:"bulk_until"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}
	s.BulkUntil = stored.BulkUntil
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
// the same per-user rate threshold.
type Classifier struct {
	store     jetstream.KeyValue
	threshold int
	window    time.Duration
	cooldown  time.Duration
}

func newClassifier(store jetstream.KeyValue, threshold int, window, cooldown time.Duration) *Classifier {
	return &Classifier{
		store:     store,
		threshold: threshold,
		window:    window,
		cooldown:  cooldown,
	}
}

// Classify records one scan request and returns its priority. Requests beyond
// the threshold start a cooldown during which that user's requests stay low.
func (c *Classifier) Classify(ctx context.Context, userID, eventID string, eventTime, now time.Time) (Priority, error) {
	if userID == "" {
		// Keep system-generated uploads working normally when no actor is present.
		return PriorityHigh, nil
	}
	if eventTime.IsZero() {
		eventTime = now
	}

	key := classifierKey(userID)
	for attempt := 0; attempt < 32; attempt++ {
		entry, err := c.store.Get(ctx, key)
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			state, priority := c.advance(rateState{}, eventID, eventTime, now)
			data, marshalErr := json.Marshal(state)
			if marshalErr != nil {
				return "", marshalErr
			}
			if _, err = c.store.Create(ctx, key, data); errors.Is(err, jetstream.ErrKeyExists) {
				continue
			} else if err != nil {
				return "", fmt.Errorf("create antivirus rate state: %w", err)
			}
			return priority, nil
		}
		if err != nil {
			return "", fmt.Errorf("read antivirus rate state: %w", err)
		}

		var state rateState
		if err := json.Unmarshal(entry.Value(), &state); err != nil {
			return "", fmt.Errorf("decode antivirus rate state: %w", err)
		}
		state, priority := c.advance(state, eventID, eventTime, now)
		data, err := json.Marshal(state)
		if err != nil {
			return "", err
		}
		if _, err = c.store.Update(ctx, key, data, entry.Revision()); errors.Is(err, jetstream.ErrKeyRevisionMismatch) {
			continue
		} else if err != nil {
			return "", fmt.Errorf("update antivirus rate state: %w", err)
		}
		return priority, nil
	}

	return "", errors.New("antivirus rate state remained contended")
}

// CurrentPriority reports the user's current rate class without recording a
// new request. It lets workers demote high-lane jobs that were queued before
// the threshold crossing was observable.
func (c *Classifier) CurrentPriority(ctx context.Context, userID string, now time.Time) (Priority, error) {
	if userID == "" {
		return PriorityHigh, nil
	}
	entry, err := c.store.Get(ctx, classifierKey(userID))
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return PriorityHigh, nil
	}
	if err != nil {
		return "", fmt.Errorf("read antivirus rate state: %w", err)
	}
	var state rateState
	if err := json.Unmarshal(entry.Value(), &state); err != nil {
		return "", fmt.Errorf("decode antivirus rate state: %w", err)
	}
	if now.Before(state.BulkUntil) {
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

func (c *Classifier) advance(state rateState, eventID string, eventTime, now time.Time) (rateState, Priority) {
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
	newRequest := !eventTime.Before(cutoff)
	if newRequest {
		state.Recent = append(state.Recent, rateRequest{ID: eventID, At: eventTime})
	}
	sort.Slice(state.Recent, func(i, j int) bool {
		if state.Recent[i].At.Equal(state.Recent[j].At) {
			return state.Recent[i].ID < state.Recent[j].ID
		}
		return state.Recent[i].At.Before(state.Recent[j].At)
	})
	if maxRecent := c.threshold + 1; len(state.Recent) > maxRecent {
		state.Recent = state.Recent[len(state.Recent)-maxRecent:]
	}

	if newRequest && len(state.Recent) > c.threshold {
		state.BulkUntil = now.Add(c.cooldown)
	}
	priority := PriorityHigh
	if now.Before(state.BulkUntil) {
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
