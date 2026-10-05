package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	revaevents "github.com/opencloud-eu/reva/v2/pkg/events"
)

func (q *Queue) subscribeInput() error {
	var err error
	// Use the raw NATS message here so the source can be acknowledged only after
	// its scan request has been persisted to the priority stream. The generic
	// Reva events.Consume wrapper does not expose this message's Ack handle.
	q.input, err = q.legacyJS.QueueSubscribe(
		revaevents.MainQueueName,
		inputConsumer,
		func(msg *nats.Msg) {
			select {
			case q.inputMsgs <- msg:
			case <-q.ctx.Done():
				_ = msg.NakWithDelay(inputRetryDelay)
			}
		},
		nats.Durable(inputConsumer),
		nats.ManualAck(),
		nats.DeliverNew(),
	)
	if err != nil {
		return fmt.Errorf("subscribe antivirus durable input: %w", err)
	}
	return nil
}

func (q *Queue) intake() {
	defer q.inputWG.Done()
	for {
		select {
		case <-q.ctx.Done():
			return
		case msg := <-q.inputMsgs:
			q.handleInput(msg)
		}
	}
}

func (q *Queue) readJobs(messages jetstream.MessagesContext, ready chan<- *Lease) {
	defer q.readerWG.Done()
	for {
		msg, err := messages.Next(jetstream.NextContext(q.ctx))
		if err != nil {
			if q.ctx.Err() != nil || errors.Is(err, jetstream.ErrMsgIteratorClosed) {
				return
			}
			if errors.Is(err, nats.ErrTimeout) || errors.Is(err, jetstream.ErrNoMessages) {
				continue
			}
			q.report(fmt.Errorf("read antivirus job: %w", err))
			if !q.waitForRetry() {
				return
			}
			continue
		}

		var job Job
		if err := json.Unmarshal(msg.Data(), &job); err != nil {
			_ = msg.Term()
			q.report(fmt.Errorf("discard malformed antivirus job: %w", err))
			continue
		}
		select {
		case ready <- &Lease{Job: job, msg: msg}:
		case <-q.ctx.Done():
			_ = msg.NakWithDelay(inputRetryDelay)
			return
		}
	}
}

func (q *Queue) waitForRetry() bool {
	timer := time.NewTimer(inputRetryDelay)
	defer timer.Stop()
	select {
	case <-q.ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (q *Queue) monitor() {
	defer q.monitorWG.Done()
	updatePending := func() {
		info, err := q.high.Info(q.ctx)
		if err != nil {
			q.reportMonitor(fmt.Errorf("read high-priority antivirus queue state: %w", err))
			return
		}
		q.highPending.Store(info.NumPending > 0)
	}
	updateInputAckFloor := func() {
		info, err := q.legacyJS.ConsumerInfo(revaevents.MainQueueName, inputConsumer)
		if err != nil {
			q.reportMonitor(fmt.Errorf("read antivirus input acknowledgment floor: %w", err))
			return
		}
		q.inputAckFloor.Store(info.AckFloor.Stream)
	}
	updateMetrics := func(priority string, consumer jetstream.Consumer) {
		info, err := consumer.Info(q.ctx)
		if err != nil {
			q.reportMonitor(fmt.Errorf("read %s antivirus queue metrics: %w", priority, err))
			return
		}
		q.metrics.JobsPending.WithLabelValues(priority).Set(float64(info.NumPending))
		q.metrics.JobsInFlight.WithLabelValues(priority).Set(float64(info.NumAckPending))
	}
	priorityTicker := time.NewTicker(priorityPollInterval)
	defer priorityTicker.Stop()
	metricsTicker := time.NewTicker(monitorInterval)
	defer metricsTicker.Stop()
	updatePending()
	updateInputAckFloor()
	updateMetrics(string(PriorityHigh), q.high)
	updateMetrics(string(PriorityLow), q.low)
	for {
		select {
		case <-q.ctx.Done():
			return
		case <-priorityTicker.C:
			updatePending()
			updateInputAckFloor()
		case <-metricsTicker.C:
			updatePending()
			updateInputAckFloor()
			updateMetrics(string(PriorityHigh), q.high)
			updateMetrics(string(PriorityLow), q.low)
		}
	}
}

type eventEnvelope struct {
	ID        string            `json:"ID"`
	Timestamp time.Time         `json:"Timestamp"`
	Metadata  map[string]string `json:"Metadata"`
	Payload   []byte            `json:"Payload"`
}

func (q *Queue) handleInput(msg *nats.Msg) {
	var envelope eventEnvelope
	if err := json.Unmarshal(msg.Data, &envelope); err != nil {
		q.report(fmt.Errorf("decode antivirus input event: %w", err))
		_ = msg.Term()
		return
	}

	eventType := envelope.Metadata[revaevents.MetadatakeyEventType]
	switch eventType {
	case reflect.TypeOf(revaevents.UploadReady{}).String():
		var ev revaevents.UploadReady
		if err := json.Unmarshal(envelope.Payload, &ev); err != nil {
			q.report(fmt.Errorf("decode antivirus UploadReady event: %w", err))
			_ = msg.Term()
			return
		}
		if !q.inputPredecessorsAcked(msg) {
			_ = msg.NakWithDelay(resourceBlockedWait)
			return
		}
		if err := q.resourceOrder.FinishUpload(q.ctx, ev.UploadID); err != nil {
			q.report(err)
			_ = msg.NakWithDelay(inputRetryDelay)
			return
		}
		if err := msg.AckSync(); err != nil {
			q.report(fmt.Errorf("acknowledge UploadReady event: %w", err))
		}
		return
	case reflect.TypeOf(revaevents.CleanUpload{}).String():
		var ev revaevents.CleanUpload
		if err := json.Unmarshal(envelope.Payload, &ev); err != nil {
			q.report(fmt.Errorf("decode antivirus CleanUpload event: %w", err))
			_ = msg.Term()
			return
		}
		if !q.inputPredecessorsAcked(msg) {
			_ = msg.NakWithDelay(resourceBlockedWait)
			return
		}
		if err := q.resourceOrder.FinishUpload(q.ctx, ev.UploadID); err != nil {
			q.report(err)
			_ = msg.NakWithDelay(inputRetryDelay)
			return
		}
		if err := msg.AckSync(); err != nil {
			q.report(fmt.Errorf("acknowledge CleanUpload event: %w", err))
		}
		return
	}

	if eventType != reflect.TypeOf(revaevents.StartPostprocessingStep{}).String() {
		if err := msg.Ack(); err != nil {
			q.report(fmt.Errorf("acknowledge unrelated antivirus input: %w", err))
		}
		return
	}

	var ev revaevents.StartPostprocessingStep
	if err := json.Unmarshal(envelope.Payload, &ev); err != nil {
		q.report(fmt.Errorf("decode antivirus postprocessing step: %w", err))
		_ = msg.Term()
		return
	}
	if ev.StepToStart != revaevents.PPStepAntivirus {
		if err := msg.Ack(); err != nil {
			q.report(fmt.Errorf("acknowledge non-virus postprocessing step: %w", err))
		}
		return
	}

	metadata, err := msg.Metadata()
	if err != nil {
		q.report(fmt.Errorf("read antivirus input stream metadata: %w", err))
		_ = msg.NakWithDelay(inputRetryDelay)
		return
	}
	jobID := envelope.Metadata[revaevents.MetadatakeyEventID]
	if jobID == "" {
		jobID = envelope.ID
	}
	if jobID == "" {
		jobID = fmt.Sprintf("main-%d", metadata.Sequence.Stream)
	}

	now := time.Now()
	eventTime := envelope.Timestamp
	priority, err := q.classifier.Classify(q.ctx, userIdentity(ev.ExecutingUser), eventTime, now)
	if err != nil {
		q.report(err)
		_ = msg.NakWithDelay(inputRetryDelay)
		return
	}
	job := Job{
		ID:             jobID,
		SourceSequence: metadata.Sequence.Stream,
		ResourceKey:    resourceKey(ev.ResourceID),
		Event:          ev,
		TraceParent:    envelope.Metadata[revaevents.MetadatakeyTraceParent],
		InitiatorID:    envelope.Metadata[revaevents.MetadatakeyInitiatorID],
		Priority:       priority,
		EnqueuedAt:     time.Now().UTC(),
	}
	if err := q.resourceOrder.Register(q.ctx, job.ResourceKey, job.ID, ev.UploadID, job.SourceSequence); err != nil {
		q.report(err)
		_ = msg.NakWithDelay(inputRetryDelay)
		return
	}
	if err := q.enqueue(q.ctx, job); err != nil {
		q.report(err)
		_ = msg.NakWithDelay(inputRetryDelay)
		return
	}
	if err := msg.AckSync(); err != nil {
		q.report(fmt.Errorf("acknowledge durably enqueued antivirus job: %w", err))
	}
}

func (q *Queue) inputPredecessorsAcked(msg *nats.Msg) bool {
	metadata, err := msg.Metadata()
	if err != nil {
		q.report(fmt.Errorf("read antivirus input sequence: %w", err))
		return false
	}
	sequence := metadata.Sequence.Stream
	return sequence == 0 || q.inputAckFloor.Load() >= sequence-1
}

func userIdentity(u *user.User) string {
	if u == nil || u.GetId() == nil {
		return ""
	}
	id := u.GetId()
	return fmt.Sprintf("%s\x00%s\x00%s", id.GetType().String(), id.GetIdp(), id.GetOpaqueId())
}

func resourceKey(id *provider.ResourceId) string {
	if id == nil || id.GetSpaceId() == "" || id.GetOpaqueId() == "" {
		return ""
	}
	identity := fmt.Sprintf("%s\x00%s\x00%s", id.GetStorageId(), id.GetSpaceId(), id.GetOpaqueId())
	return classifierKey(identity)
}

func (q *Queue) enqueue(ctx context.Context, job Job) error {
	return q.enqueueWithMessageID(ctx, job, job.ID)
}

func (q *Queue) enqueueWithMessageID(ctx context.Context, job Job, messageID string) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("encode antivirus job: %w", err)
	}
	subject := lowSubject
	if job.Priority == PriorityHigh {
		subject = highSubject
	}
	if _, err := q.js.Publish(ctx, subject, data, jetstream.WithMsgID(messageID)); err != nil {
		return fmt.Errorf("persist antivirus job %q: %w", job.ID, err)
	}
	q.metrics.JobsEnqueued.WithLabelValues(string(job.Priority)).Inc()
	return nil
}

// Next returns a high-priority job whenever one is available. Low-priority
// work is fetched only after checking the high-priority lane.
