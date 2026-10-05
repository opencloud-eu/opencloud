package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	resourceOrderKeyPrefix = "resource-"
	resourceCompletedLimit = 64
	resourceCASRetries     = 32
)

var errResourceLeaseLost = errors.New("antivirus resource lease lost")

type resourceJob struct {
	ID       string `json:"id"`
	UploadID string `json:"upload_id,omitempty"`
	Sequence uint64 `json:"sequence"`
}

type resourceState struct {
	Pending         []resourceJob `json:"pending,omitempty"`
	RunningJobID    string        `json:"running_job_id,omitempty"`
	RunningUploadID string        `json:"running_upload_id,omitempty"`
	RunningToken    string        `json:"running_token,omitempty"`
	LeaseUntil      time.Time     `json:"lease_until,omitempty"`
	Completed       []string      `json:"completed,omitempty"`
}

type resourceClaim uint8

const (
	resourceReady resourceClaim = iota
	resourceBlocked
	resourceCompleted
)

// resourceOrder persists a FIFO per ResourceID, independent of the global
// priority lane. This prevents a newer save from overtaking an older upload
// for the same file.
type resourceOrder struct {
	store       jetstream.KeyValue
	leasePeriod time.Duration
}

func newResourceOrder(store jetstream.KeyValue, leasePeriod time.Duration) *resourceOrder {
	return &resourceOrder{store: store, leasePeriod: leasePeriod}
}

func (o *resourceOrder) Register(ctx context.Context, resourceKey, jobID, uploadID string, sequence uint64) error {
	if resourceKey == "" {
		return nil
	}
	key := resourceOrderKey(resourceKey)
	for attempt := 0; attempt < resourceCASRetries; attempt++ {
		entry, err := o.store.Get(ctx, key)
		missing := errors.Is(err, jetstream.ErrKeyNotFound)
		state := resourceState{}
		if err != nil && !missing {
			return fmt.Errorf("read antivirus resource order: %w", err)
		}
		if !missing {
			if err := json.Unmarshal(entry.Value(), &state); err != nil {
				return fmt.Errorf("decode antivirus resource order: %w", err)
			}
		}
		if slices.Contains(state.Completed, jobID) {
			return nil
		}
		found := false
		for index := range state.Pending {
			item := &state.Pending[index]
			if item.ID == jobID {
				item.UploadID = uploadID
				found = true
				break
			}
			// A postprocessing retry has a new event ID but belongs to the
			// same upload. Keep its original position in this resource's FIFO.
			if uploadID != "" && item.UploadID == uploadID {
				item.ID = jobID
				item.Sequence = min(item.Sequence, sequence)
				found = true
				break
			}
		}
		if !found {
			state.Pending = append(state.Pending, resourceJob{ID: jobID, UploadID: uploadID, Sequence: sequence})
		}
		sort.Slice(state.Pending, func(i, j int) bool {
			if state.Pending[i].Sequence == state.Pending[j].Sequence {
				return state.Pending[i].ID < state.Pending[j].ID
			}
			return state.Pending[i].Sequence < state.Pending[j].Sequence
		})
		data, err := json.Marshal(state)
		if err != nil {
			return err
		}
		if missing {
			if _, err := o.store.Create(ctx, key, data); errors.Is(err, jetstream.ErrKeyExists) {
				continue
			} else if err != nil {
				return fmt.Errorf("create antivirus resource order: %w", err)
			}
			return o.writeUploadIndex(ctx, uploadID, resourceKey)
		}
		if _, err := o.store.Update(ctx, key, data, entry.Revision()); errors.Is(err, jetstream.ErrKeyRevisionMismatch) {
			continue
		} else if err != nil {
			return fmt.Errorf("update antivirus resource order: %w", err)
		}
		return o.writeUploadIndex(ctx, uploadID, resourceKey)
	}
	return errors.New("antivirus resource order remained contended while registering a job")
}

func (o *resourceOrder) writeUploadIndex(ctx context.Context, uploadID, resourceKey string) error {
	if uploadID == "" {
		return nil
	}
	key := uploadOrderKey(uploadID)
	for attempt := 0; attempt < resourceCASRetries; attempt++ {
		entry, err := o.store.Get(ctx, key)
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			if _, err := o.store.Create(ctx, key, []byte(resourceKey)); errors.Is(err, jetstream.ErrKeyExists) {
				continue
			} else if err != nil {
				return fmt.Errorf("create antivirus upload order index: %w", err)
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("read antivirus upload order index: %w", err)
		}
		if string(entry.Value()) == resourceKey {
			return nil
		}
		if _, err := o.store.Update(ctx, key, []byte(resourceKey), entry.Revision()); errors.Is(err, jetstream.ErrKeyRevisionMismatch) {
			continue
		} else if err != nil {
			return fmt.Errorf("update antivirus upload order index: %w", err)
		}
		return nil
	}
	return errors.New("antivirus upload order index remained contended")
}

// FinishUpload removes all scan attempts for an upload after storage has
// emitted UploadReady. It also releases a scan lease if finalization raced its
// acknowledgement.
func (o *resourceOrder) FinishUpload(ctx context.Context, uploadID string) error {
	if uploadID == "" {
		return nil
	}
	indexKey := uploadOrderKey(uploadID)
	index, err := o.store.Get(ctx, indexKey)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read antivirus upload order index: %w", err)
	}
	resourceKey := string(index.Value())
	key := resourceOrderKey(resourceKey)
	updated := false
	for attempt := 0; attempt < resourceCASRetries; attempt++ {
		entry, err := o.store.Get(ctx, key)
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			updated = true
			break
		}
		if err != nil {
			return fmt.Errorf("read antivirus resource order at upload completion: %w", err)
		}
		var state resourceState
		if err := json.Unmarshal(entry.Value(), &state); err != nil {
			return fmt.Errorf("decode antivirus resource order at upload completion: %w", err)
		}
		kept := state.Pending[:0]
		removed := make([]string, 0, 1)
		for _, item := range state.Pending {
			if item.UploadID == uploadID {
				removed = append(removed, item.ID)
			} else {
				kept = append(kept, item)
			}
		}
		state.Pending = kept
		if state.RunningUploadID == uploadID {
			if state.RunningJobID != "" {
				removed = append(removed, state.RunningJobID)
			}
			state.RunningJobID = ""
			state.RunningUploadID = ""
			state.RunningToken = ""
			state.LeaseUntil = time.Time{}
		}
		state.Completed = append(state.Completed, removed...)
		if len(state.Completed) > resourceCompletedLimit {
			state.Completed = state.Completed[len(state.Completed)-resourceCompletedLimit:]
		}
		data, err := json.Marshal(state)
		if err != nil {
			return err
		}
		if _, err := o.store.Update(ctx, key, data, entry.Revision()); errors.Is(err, jetstream.ErrKeyRevisionMismatch) {
			continue
		} else if err != nil {
			return fmt.Errorf("finish antivirus resource order: %w", err)
		}
		updated = true
		break
	}
	if !updated {
		return errors.New("antivirus resource order remained contended while finishing an upload")
	}
	if err := o.store.Delete(ctx, indexKey); err != nil && !errors.Is(err, jetstream.ErrKeyNotFound) {
		return fmt.Errorf("delete antivirus upload order index: %w", err)
	}
	return nil
}

func (o *resourceOrder) Claim(ctx context.Context, job Job) (resourceClaim, string, error) {
	if job.ResourceKey == "" {
		return resourceReady, "", nil
	}
	key := resourceOrderKey(job.ResourceKey)
	leaseToken := uuid.NewString()
	for attempt := 0; attempt < resourceCASRetries; attempt++ {
		entry, err := o.store.Get(ctx, key)
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			// Jobs from an older queue format may not have a corresponding ordering
			// record. Keep them processable during upgrades.
			return resourceReady, "", nil
		}
		if err != nil {
			return resourceBlocked, "", fmt.Errorf("read antivirus resource order: %w", err)
		}
		var state resourceState
		if err := json.Unmarshal(entry.Value(), &state); err != nil {
			return resourceBlocked, "", fmt.Errorf("decode antivirus resource order: %w", err)
		}
		if slices.Contains(state.Completed, job.ID) {
			return resourceCompleted, "", nil
		}
		index := findJob(state.Pending, job.ID)
		if index < 0 {
			for _, pending := range state.Pending {
				if job.Event.UploadID != "" && pending.UploadID == job.Event.UploadID {
					// A retry for this upload has superseded this delivery.
					return resourceCompleted, "", nil
				}
			}
			return resourceReady, "", nil
		}
		if index != 0 || (state.RunningJobID != "" && time.Now().Before(state.LeaseUntil)) {
			return resourceBlocked, "", nil
		}
		state.RunningJobID = job.ID
		state.RunningUploadID = job.Event.UploadID
		state.RunningToken = leaseToken
		state.LeaseUntil = time.Now().Add(o.leasePeriod)
		data, err := json.Marshal(state)
		if err != nil {
			return resourceBlocked, "", err
		}
		if _, err := o.store.Update(ctx, key, data, entry.Revision()); errors.Is(err, jetstream.ErrKeyRevisionMismatch) {
			continue
		} else if err != nil {
			return resourceBlocked, "", fmt.Errorf("claim antivirus resource order: %w", err)
		}
		return resourceReady, leaseToken, nil
	}
	return resourceBlocked, "", errors.New("antivirus resource order remained contended while claiming a job")
}

func (o *resourceOrder) Renew(ctx context.Context, job Job, token string) error {
	if job.ResourceKey == "" || token == "" {
		return nil
	}
	return o.updateLease(ctx, job, token, false)
}

func (o *resourceOrder) Validate(ctx context.Context, job Job, token string) error {
	if job.ResourceKey == "" || token == "" {
		return nil
	}
	entry, err := o.store.Get(ctx, resourceOrderKey(job.ResourceKey))
	if err != nil {
		return fmt.Errorf("read antivirus resource lease: %w", err)
	}
	var state resourceState
	if err := json.Unmarshal(entry.Value(), &state); err != nil {
		return fmt.Errorf("decode antivirus resource lease: %w", err)
	}
	if state.RunningJobID != job.ID || state.RunningToken != token || !time.Now().Before(state.LeaseUntil) {
		return errResourceLeaseLost
	}
	return nil
}

func (o *resourceOrder) Release(ctx context.Context, job Job, token string) error {
	if job.ResourceKey == "" || token == "" {
		return nil
	}
	return o.updateLease(ctx, job, token, true)
}

func (o *resourceOrder) updateLease(ctx context.Context, job Job, token string, release bool) error {
	key := resourceOrderKey(job.ResourceKey)
	for attempt := 0; attempt < resourceCASRetries; attempt++ {
		entry, err := o.store.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("read antivirus resource lease: %w", err)
		}
		var state resourceState
		if err := json.Unmarshal(entry.Value(), &state); err != nil {
			return fmt.Errorf("decode antivirus resource lease: %w", err)
		}
		if state.RunningJobID != job.ID || state.RunningToken != token {
			if release && slices.Contains(state.Completed, job.ID) {
				return nil
			}
			return errResourceLeaseLost
		}
		if release {
			state.RunningJobID = ""
			state.RunningUploadID = ""
			state.RunningToken = ""
			state.LeaseUntil = time.Time{}
		} else {
			state.LeaseUntil = time.Now().Add(o.leasePeriod)
		}
		data, err := json.Marshal(state)
		if err != nil {
			return err
		}
		if _, err := o.store.Update(ctx, key, data, entry.Revision()); errors.Is(err, jetstream.ErrKeyRevisionMismatch) {
			continue
		} else if err != nil {
			return fmt.Errorf("update antivirus resource lease: %w", err)
		}
		return nil
	}
	return errors.New("antivirus resource lease remained contended")
}

func (o *resourceOrder) Complete(ctx context.Context, job Job, token string) error {
	if job.ResourceKey == "" || token == "" {
		return nil
	}
	key := resourceOrderKey(job.ResourceKey)
	for attempt := 0; attempt < resourceCASRetries; attempt++ {
		entry, err := o.store.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("read antivirus resource queue: %w", err)
		}
		var state resourceState
		if err := json.Unmarshal(entry.Value(), &state); err != nil {
			return fmt.Errorf("decode antivirus resource queue: %w", err)
		}
		if slices.Contains(state.Completed, job.ID) {
			return nil
		}
		if state.RunningJobID != job.ID || state.RunningToken != token || len(state.Pending) == 0 || state.Pending[0].ID != job.ID {
			return errResourceLeaseLost
		}
		state.Pending = state.Pending[1:]
		state.RunningJobID = ""
		state.RunningUploadID = ""
		state.RunningToken = ""
		state.LeaseUntil = time.Time{}
		state.Completed = append(state.Completed, job.ID)
		if len(state.Completed) > resourceCompletedLimit {
			state.Completed = state.Completed[len(state.Completed)-resourceCompletedLimit:]
		}
		data, err := json.Marshal(state)
		if err != nil {
			return err
		}
		if _, err := o.store.Update(ctx, key, data, entry.Revision()); errors.Is(err, jetstream.ErrKeyRevisionMismatch) {
			continue
		} else if err != nil {
			return fmt.Errorf("complete antivirus resource job: %w", err)
		}
		return nil
	}
	return errors.New("antivirus resource queue remained contended while completing a job")
}

func resourceOrderKey(resourceKey string) string { return resourceOrderKeyPrefix + resourceKey }

func findJob(jobs []resourceJob, id string) int {
	for index, job := range jobs {
		if job.ID == id {
			return index
		}
	}
	return -1
}

func uploadOrderKey(uploadID string) string { return "upload-" + classifierKey(uploadID) }
