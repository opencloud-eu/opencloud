package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	revaevents "github.com/opencloud-eu/reva/v2/pkg/events"
	microevents "go-micro.dev/v4/events"
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
		case ready <- &Lease{Job: job, msg: msg, queue: q}:
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
	updateMetrics := func(priority string, consumer jetstream.Consumer) {
		info, err := consumer.Info(q.ctx)
		if err != nil {
			q.reportMonitor(fmt.Errorf("read %s antivirus queue metrics: %w", priority, err))
			return
		}
		q.metrics.JobsPending.WithLabelValues(priority).Set(float64(info.NumPending))
		q.metrics.JobsInFlight.WithLabelValues(priority).Set(float64(info.NumAckPending))
	}
	metricsTicker := time.NewTicker(monitorInterval)
	defer metricsTicker.Stop()
	updateMetrics(string(PriorityHigh), q.high)
	updateMetrics(string(PriorityLow), q.low)
	for {
		select {
		case <-q.ctx.Done():
			return
		case <-metricsTicker.C:
			updateMetrics(string(PriorityHigh), q.high)
			updateMetrics(string(PriorityLow), q.low)
		}
	}
}

func (q *Queue) handleInput(msg *nats.Msg) {
	var envelope microevents.Event
	if err := json.Unmarshal(msg.Data, &envelope); err != nil {
		q.report(fmt.Errorf("decode antivirus input event: %w", err))
		_ = msg.Term()
		return
	}

	if envelope.Metadata[revaevents.MetadatakeyEventType] != reflect.TypeFor[revaevents.StartPostprocessingStep]().String() {
		// Not a scan request, e.g. UploadReady or CleanUpload. Acknowledge it so
		// it does not block the durable consumer.
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

	priority := PriorityHigh
	if q.priorityEnabled {
		priority, err = q.classifier.Classify(q.ctx, userIdentity(ev.ExecutingUser), jobID, envelope.Timestamp, time.Now(), ev.Filesize)
		if err != nil {
			q.report(err)
			_ = msg.NakWithDelay(inputRetryDelay)
			return
		}
	}
	job := Job{
		ID:          jobID,
		Event:       ev,
		TraceParent: envelope.Metadata[revaevents.MetadatakeyTraceParent],
		InitiatorID: envelope.Metadata[revaevents.MetadatakeyInitiatorID],
		Priority:    priority,
		EnqueuedAt:  time.Now().UTC(),
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

func userIdentity(u *user.User) string {
	if u == nil || u.GetId() == nil {
		return ""
	}
	id := u.GetId()
	return fmt.Sprintf("%s\x00%s\x00%s", id.GetType().String(), id.GetIdp(), id.GetOpaqueId())
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
	ack, err := q.js.Publish(ctx, subject, data, jetstream.WithMsgID(messageID))
	if err != nil {
		return fmt.Errorf("persist antivirus job %q: %w", job.ID, err)
	}
	if !ack.Duplicate {
		q.metrics.JobsEnqueued.WithLabelValues(string(job.Priority)).Inc()
	}
	return nil
}
