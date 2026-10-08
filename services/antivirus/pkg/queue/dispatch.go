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
		if q.highPriorityPending() {
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
			case <-time.After(consumerPollWait):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
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
			ready, handled, claimErr := q.claimResource(ctx, lease)
			if claimErr != nil {
				return nil, claimErr
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

func (q *Queue) highPriorityPending() bool {
	return len(q.highReady) > 0 || q.highPending.Load()
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

func (q *Queue) claimHigh(ctx context.Context, lease *Lease) (*Lease, bool, error) {
	if !lease.Job.EnqueuedAt.IsZero() {
		if delay := time.Until(lease.Job.EnqueuedAt.Add(q.priorityWindow)); delay > 0 {
			if err := lease.NakWithDelay(delay); err != nil {
				return nil, false, fmt.Errorf("defer antivirus job until its rate window closes: %w", err)
			}
			return nil, true, nil
		}
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
		if err := lease.Ack(); err != nil {
			q.report(fmt.Errorf("acknowledge demoted antivirus job: %w", err))
		}
		return nil, true, nil
	}
	ready, handled, err := q.claimResource(ctx, lease)
	return ready, handled, err
}

func (q *Queue) recordClaim(lease *Lease) {
	q.metrics.QueueWait.WithLabelValues(string(lease.Job.Priority)).Observe(time.Since(lease.Job.EnqueuedAt).Seconds())
}

func (q *Queue) claimResource(ctx context.Context, lease *Lease) (*Lease, bool, error) {
	if lease.Job.ResourceKey != "" && lease.Job.SourceSequence > 0 {
		if q.inputAckFloor.Load() < lease.Job.SourceSequence {
			if err := lease.NakWithDelay(resourceBlockedWait); err != nil {
				return nil, false, err
			}
			return nil, true, nil
		}
	}
	claim, token, err := q.resourceOrder.Claim(ctx, lease.Job)
	if err != nil {
		_ = lease.NakWithDelay(inputRetryDelay)
		return nil, false, err
	}
	switch claim {
	case resourceBlocked:
		if err := lease.NakWithDelay(resourceBlockedWait); err != nil {
			return nil, false, err
		}
		return nil, true, nil
	case resourceCompleted:
		if err := lease.msg.Ack(); err != nil {
			return nil, false, err
		}
		return nil, true, nil
	default:
		if token != "" {
			lease.resourceOrder = q.resourceOrder
			lease.resourceToken = token
		}
		return lease, false, nil
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
		revaevents.MetadatakeyEventType:   reflect.TypeOf(ev).String(),
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
