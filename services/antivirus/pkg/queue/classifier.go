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

type rateState struct {
	Recent    []time.Time `json:"recent"`
	BulkUntil time.Time   `json:"bulk_until"`
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
func (c *Classifier) Classify(ctx context.Context, userID string, eventTime, now time.Time) (Priority, error) {
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
			state, priority := c.advance(rateState{}, eventTime, now)
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
		state, priority := c.advance(state, eventTime, now)
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
	for _, at := range state.Recent {
		if !at.Before(cutoff) {
			count++
		}
	}
	if count > c.threshold {
		return PriorityLow, nil
	}
	return PriorityHigh, nil
}

func (c *Classifier) advance(state rateState, eventTime, now time.Time) (rateState, Priority) {
	latestEvent := eventTime
	for _, at := range state.Recent {
		if at.After(latestEvent) {
			latestEvent = at
		}
	}
	cutoff := latestEvent.Add(-c.window)
	recent := state.Recent[:0]
	for _, at := range state.Recent {
		if !at.Before(cutoff) {
			recent = append(recent, at)
		}
	}
	state.Recent = recent
	if !eventTime.Before(cutoff) {
		state.Recent = append(state.Recent, eventTime)
	}
	sort.Slice(state.Recent, func(i, j int) bool {
		return state.Recent[i].Before(state.Recent[j])
	})
	if maxRecent := c.threshold + 1; len(state.Recent) > maxRecent {
		state.Recent = state.Recent[len(state.Recent)-maxRecent:]
	}

	if len(state.Recent) > c.threshold {
		state.BulkUntil = now.Add(c.cooldown)
	}
	if now.Before(state.BulkUntil) {
		return state, PriorityLow
	}
	return state, PriorityHigh
}

func classifierKey(userID string) string {
	sum := sha256.Sum256([]byte(userID))
	return hex.EncodeToString(sum[:])
}
