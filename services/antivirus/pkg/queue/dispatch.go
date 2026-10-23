package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/nats-io/nats.go"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	revaevents "github.com/opencloud-eu/reva/v2/pkg/events"
	microevents "go-micro.dev/v4/events"
	"go.opentelemetry.io/otel/propagation"
)

// Next returns a high-priority job whenever one is available. Low-priority
// work is only selected when the local high-priority lane is empty.
func (q *Queue) Next(ctx context.Context) (*Lease, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		select {
		case lease := <-q.highReady:
			ready, handled, err := q.claimHigh(ctx, lease)
			if err != nil {
				return nil, err
			}
			if handled {
				continue
			}
			q.recordClaim(ready)
			return ready, nil
		default:
		}
		select {
		case lease := <-q.highReady:
			ready, handled, err := q.claimHigh(ctx, lease)
			if err != nil {
				return nil, err
			}
			if handled {
				continue
			}
			q.recordClaim(ready)
			return ready, nil
		case lease := <-q.lowReady:
			q.recordClaim(lease)
			return lease, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// NextHighPriority is used by reserved workers that must never start a
// low-priority scan. It keeps one unit of scanner capacity available for
// interactive/high-priority requests.
func (q *Queue) NextHighPriority(ctx context.Context) (*Lease, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		select {
		case lease := <-q.highReady:
			ready, handled, err := q.claimHigh(ctx, lease)
			if err != nil {
				return nil, err
			}
			if handled {
				continue
			}
			q.recordClaim(ready)
			return ready, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// claimHigh returns (lease, false, nil) when the job may run, or
// (nil, true, nil) when the user turned out to be a heavy uploader and the job
// was moved to the low-priority lane.
func (q *Queue) claimHigh(ctx context.Context, lease *Lease) (*Lease, bool, error) {
	if !q.priorityEnabled {
		return lease, false, nil
	}
	priority, err := q.classifier.CurrentPriority(ctx, userIdentity(lease.Job.Event.ExecutingUser), time.Now())
	if err != nil {
		_ = lease.NakWithDelay(inputRetryDelay)
		return nil, false, err
	}
	if priority == PriorityLow {
		lowJob := lease.Job
		lowJob.Priority = PriorityLow
		if err := q.enqueueWithMessageID(ctx, lowJob, lowJob.ID+"-low"); err != nil {
			_ = lease.NakWithDelay(inputRetryDelay)
			return nil, false, err
		}
		q.metrics.JobsDemoted.WithLabelValues(string(PriorityHigh)).Inc()
		if err := lease.ackRaw(); err != nil {
			q.report(fmt.Errorf("acknowledge demoted antivirus job: %w", err))
		}
		return nil, true, nil
	}
	return lease, false, nil
}

func (q *Queue) recordClaim(lease *Lease) {
	if lease.Job.EnqueuedAt.IsZero() {
		return
	}
	q.metrics.QueueWait.WithLabelValues(string(lease.Job.Priority)).Observe(time.Since(lease.Job.EnqueuedAt).Seconds())
}

// finishOutstanding removes a finished scan from the per-user backlog
// accounting. Failures are reported but never block the acknowledgement.
func (q *Queue) finishOutstanding(job Job) {
	if q.classifier == nil || !q.classifier.enabled {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), classifierOpTimeout)
	defer cancel()
	if err := q.classifier.Complete(ctx, userIdentity(job.Event.ExecutingUser), job.Event.Filesize); err != nil {
		q.report(fmt.Errorf("update antivirus rate state: %w", err))
	}
}

// PublishFinished writes a stable-ID completion event to the existing main
// stream. JetStream deduplication protects the postprocessing consumer from a
// duplicate when a worker crashes after publishing but before acknowledging
// its job, within the main stream's configured duplicate window.
func (q *Queue) PublishFinished(ctx context.Context, job Job, ev revaevents.PostprocessingStepFinished) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	carrier := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(ctx, carrier)
	initiatorID, _ := ctxpkg.ContextGetInitiator(ctx)
	eventID := job.ID + "-finished"
	metadata := map[string]string{
		revaevents.MetadatakeyEventType:   reflect.TypeFor[revaevents.PostprocessingStepFinished]().String(),
		revaevents.MetadatakeyEventID:     eventID,
		revaevents.MetadatakeyTraceParent: carrier.Get("traceparent"),
		revaevents.MetadatakeyInitiatorID: initiatorID,
	}
	message, err := json.Marshal(microevents.Event{
		ID:        eventID,
		Topic:     revaevents.MainQueueName,
		Timestamp: time.Now().UTC(),
		Metadata:  metadata,
		Payload:   payload,
	})
	if err != nil {
		return err
	}
	_, err = q.legacyJS.PublishMsg(&nats.Msg{
		Subject: revaevents.MainQueueName,
		Data:    message,
	}, nats.MsgId(eventID))
	if err != nil {
		return fmt.Errorf("publish antivirus completion: %w", err)
	}
	return nil
}

func (q *Queue) report(err error) {
	if err != nil && q.onError != nil && q.ctx.Err() == nil {
		q.onError(err)
	}
}

func (q *Queue) reportMonitor(err error) {
	if err == nil || q.ctx.Err() != nil {
		return
	}
	now := time.Now().UnixNano()
	last := q.lastMonitorErr.Load()
	if now-last < int64(monitorInterval) || !q.lastMonitorErr.CompareAndSwap(last, now) {
		return
	}
	q.report(err)
}

// AckWait returns the configured JetStream acknowledgement timeout.
func (q *Queue) AckWait() time.Duration { return q.ackWait }

// Close releases the input subscription and NATS connection.
func (q *Queue) Close() {
	q.closeOnce.Do(func() {
		q.cancel()
		if q.input != nil {
			_ = q.input.Unsubscribe()
		}
		if q.highMessages != nil {
			q.highMessages.Stop()
		}
		if q.lowMessages != nil {
			q.lowMessages.Stop()
		}
		q.inputWG.Wait()
		q.readerWG.Wait()
		q.monitorWG.Wait()
		if q.conn != nil {
			q.conn.Close()
		}
	})
}
