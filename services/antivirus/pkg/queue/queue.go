package queue

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	revaevents "github.com/opencloud-eu/reva/v2/pkg/events"
	microevents "go-micro.dev/v4/events"
	"go.opentelemetry.io/otel/propagation"

	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/config"
)

const (
	inputConsumer = "antivirus"
	jobStream     = "OPENCLOUD_ANTIVIRUS_JOBS"
	rateBucket    = "ANTIVIRUS_USER_RATES"
	highSubject   = "opencloud.antivirus.jobs.high"
	lowSubject    = "opencloud.antivirus.jobs.low"
	highConsumer  = "antivirus-jobs-high"
	lowConsumer   = "antivirus-jobs-low"
)

var errHighJobDemoted = errors.New("high-priority job demoted before scan")

// Job is a durable copy of an antivirus postprocessing request.
type Job struct {
	ID          string
	Event       revaevents.StartPostprocessingStep
	TraceParent string
	InitiatorID string
	Priority    Priority
	EnqueuedAt  time.Time
}

// Lease is a claimed scan job. The message remains unacknowledged until the
// scan result has been durably published to the main event stream.
type Lease struct {
	Job Job
	msg jetstream.Msg
}

func (l *Lease) Ack() error                             { return l.msg.Ack() }
func (l *Lease) InProgress() error                      { return l.msg.InProgress() }
func (l *Lease) NakWithDelay(delay time.Duration) error { return l.msg.NakWithDelay(delay) }

// Queue moves antivirus requests out of the shared event stream into durable,
// strict-priority JetStream lanes.
type Queue struct {
	ctx            context.Context
	cancel         context.CancelFunc
	conn           *nats.Conn
	legacyJS       nats.JetStreamContext
	js             jetstream.JetStream
	jobs           jetstream.Stream
	classifier     *Classifier
	priorityWindow time.Duration
	high           jetstream.Consumer
	low            jetstream.Consumer
	ackWait        time.Duration
	input          *nats.Subscription
	inputMsgs      chan *nats.Msg
	inputWG        sync.WaitGroup
	monitorWG      sync.WaitGroup
	onError        func(error)
	closeOnce      sync.Once
}

// Open creates or binds the durable streams and consumers used by the
// antivirus priority scheduler.
func Open(ctx context.Context, cfg *config.Config, onError func(error)) (*Queue, error) {
	if cfg == nil {
		return nil, errors.New("antivirus config is nil")
	}
	if cfg.Workers < 1 {
		return nil, errors.New("ANTIVIRUS_WORKERS must be greater than zero")
	}
	if cfg.HighPriorityReservedWorkers < 0 || cfg.HighPriorityReservedWorkers >= cfg.Workers {
		return nil, errors.New("ANTIVIRUS_HIGH_PRIORITY_RESERVED_WORKERS must be less than ANTIVIRUS_WORKERS")
	}
	if cfg.PriorityThreshold < 1 || cfg.PriorityWindow <= 0 || cfg.PriorityCooldown <= 0 || cfg.QueueAckWait <= 0 || cfg.QueueReplicas < 1 {
		return nil, errors.New("antivirus priority and queue configuration must be positive")
	}
	if cfg.QueueAckWait >= 2*time.Minute {
		return nil, errors.New("ANTIVIRUS_QUEUE_ACK_WAIT must be less than 2m")
	}
	options := []nats.Option{
		nats.Name(generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus) + ":priority-queue"),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
	}
	if cfg.Events.AuthUsername != "" && cfg.Events.AuthPassword != "" {
		options = append(options, nats.UserInfo(cfg.Events.AuthUsername, cfg.Events.AuthPassword))
	}
	if cfg.Events.EnableTLS {
		tlsConfig := &tls.Config{ //nolint:gosec // TLSInsecure is an explicit administrator option.
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: cfg.Events.TLSInsecure,
		}
		if cfg.Events.TLSRootCACertificate != "" {
			pem, err := os.ReadFile(cfg.Events.TLSRootCACertificate)
			if err != nil {
				return nil, fmt.Errorf("read NATS root CA: %w", err)
			}
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(pem) {
				return nil, errors.New("NATS root CA contains no certificates")
			}
			tlsConfig.RootCAs = pool
			tlsConfig.InsecureSkipVerify = false
		}
		options = append(options, nats.Secure(tlsConfig))
	}
	queueCtx, cancel := context.WithCancel(ctx)

	conn, err := nats.Connect(cfg.Events.Endpoint, options...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("connect antivirus priority queue to NATS: %w", err)
	}
	closeOnError := true
	defer func() {
		if closeOnError {
			cancel()
			conn.Close()
		}
	}()

	legacyJS, err := conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("create NATS JetStream context: %w", err)
	}
	js, err := jetstream.New(conn)
	if err != nil {
		return nil, fmt.Errorf("create NATS JetStream client: %w", err)
	}
	if _, err := js.Stream(queueCtx, revaevents.MainQueueName); errors.Is(err, jetstream.ErrStreamNotFound) {
		if _, err := js.CreateStream(queueCtx, jetstream.StreamConfig{
			Name:     revaevents.MainQueueName,
			Subjects: []string{revaevents.MainQueueName},
			MaxAge:   7 * 24 * time.Hour,
		}); err != nil {
			if !errors.Is(err, jetstream.ErrStreamNameAlreadyInUse) {
				return nil, fmt.Errorf("create OpenCloud main event stream: %w", err)
			}
			if _, err := js.Stream(queueCtx, revaevents.MainQueueName); err != nil {
				return nil, fmt.Errorf("inspect concurrently created OpenCloud main event stream: %w", err)
			}
		}
	} else if err != nil {
		return nil, fmt.Errorf("inspect OpenCloud main event stream: %w", err)
	}

	jobs, err := js.CreateOrUpdateStream(queueCtx, jetstream.StreamConfig{
		Name:       jobStream,
		Subjects:   []string{highSubject, lowSubject},
		Retention:  jetstream.WorkQueuePolicy,
		Storage:    jetstream.FileStorage,
		MaxAge:     7 * 24 * time.Hour,
		Duplicates: 2 * time.Minute,
		Replicas:   cfg.QueueReplicas,
	})
	if err != nil {
		return nil, fmt.Errorf("create antivirus job stream: %w", err)
	}

	rateStore, err := js.CreateOrUpdateKeyValue(queueCtx, jetstream.KeyValueConfig{
		Bucket:      rateBucket,
		Description: "Shared per-user antivirus priority rate state",
		History:     1,
		TTL:         24 * time.Hour,
		Storage:     jetstream.FileStorage,
		Replicas:    cfg.QueueReplicas,
	})
	if err != nil {
		return nil, fmt.Errorf("create antivirus priority rate bucket: %w", err)
	}

	high, err := jobs.CreateOrUpdateConsumer(queueCtx, jetstream.ConsumerConfig{
		Durable:       highConsumer,
		FilterSubject: highSubject,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       cfg.QueueAckWait,
	})
	if err != nil {
		return nil, fmt.Errorf("create high-priority antivirus consumer: %w", err)
	}
	low, err := jobs.CreateOrUpdateConsumer(queueCtx, jetstream.ConsumerConfig{
		Durable:       lowConsumer,
		FilterSubject: lowSubject,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       cfg.QueueAckWait,
	})
	if err != nil {
		return nil, fmt.Errorf("create low-priority antivirus consumer: %w", err)
	}

	q := &Queue{
		ctx:            queueCtx,
		cancel:         cancel,
		conn:           conn,
		legacyJS:       legacyJS,
		js:             js,
		jobs:           jobs,
		classifier:     newClassifier(rateStore, cfg.PriorityThreshold, cfg.PriorityWindow, cfg.PriorityCooldown),
		priorityWindow: cfg.PriorityWindow,
		high:           high,
		low:            low,
		ackWait:        cfg.QueueAckWait,
		inputMsgs:      make(chan *nats.Msg, 1000),
		onError:        onError,
	}
	if err := q.subscribeInput(); err != nil {
		return nil, err
	}
	for range cfg.Workers {
		q.inputWG.Add(1)
		go q.intake()
	}
	q.monitorWG.Add(1)
	go q.monitor()
	closeOnError = false
	return q, nil
}

func (q *Queue) subscribeInput() error {
	var err error
	q.input, err = q.legacyJS.QueueSubscribe(
		revaevents.MainQueueName,
		inputConsumer,
		func(msg *nats.Msg) {
			select {
			case q.inputMsgs <- msg:
			case <-q.ctx.Done():
				_ = msg.NakWithDelay(time.Second)
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

func (q *Queue) monitor() {
	defer q.monitorWG.Done()
	update := func(priority string, consumer jetstream.Consumer) {
		info, err := consumer.Info(q.ctx)
		if err != nil {
			q.report(fmt.Errorf("read %s antivirus queue metrics: %w", priority, err))
			return
		}
		jobsPending.WithLabelValues(priority).Set(float64(info.NumPending))
		jobsInFlight.WithLabelValues(priority).Set(float64(info.NumAckPending))
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		update(string(PriorityHigh), q.high)
		update(string(PriorityLow), q.low)
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
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

	if envelope.Metadata[revaevents.MetadatakeyEventType] != reflect.TypeOf(revaevents.StartPostprocessingStep{}).String() {
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

	jobID := envelope.Metadata[revaevents.MetadatakeyEventID]
	if jobID == "" {
		jobID = envelope.ID
	}
	if jobID == "" {
		metadata, err := msg.Metadata()
		if err != nil {
			q.report(fmt.Errorf("read antivirus input stream sequence: %w", err))
			_ = msg.NakWithDelay(time.Second)
			return
		}
		jobID = fmt.Sprintf("main-%d", metadata.Sequence.Stream)
	}

	now := time.Now()
	eventTime := envelope.Timestamp
	priority, err := q.classifier.Classify(q.ctx, userIdentity(ev.ExecutingUser), eventTime, now)
	if err != nil {
		q.report(err)
		_ = msg.NakWithDelay(time.Second)
		return
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
		_ = msg.NakWithDelay(time.Second)
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
	if _, err := q.js.Publish(ctx, subject, data, jetstream.WithMsgID(messageID)); err != nil {
		return fmt.Errorf("persist antivirus job %q: %w", job.ID, err)
	}
	jobsEnqueued.WithLabelValues(string(job.Priority)).Inc()
	return nil
}

// Next returns a high-priority job whenever one is available. Low-priority
// work is fetched only after checking the high-priority lane.
func (q *Queue) Next(ctx context.Context) (*Lease, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if lease, err := q.claimHigh(ctx, 0); err == nil {
			return lease, nil
		} else if errors.Is(err, errHighJobDemoted) {
			continue
		} else if !isEmpty(err) {
			return nil, err
		}
		lease, err := q.fetch(ctx, q.low, 100*time.Millisecond)
		if err == nil {
			return lease, nil
		}
		if !isEmpty(err) {
			return nil, err
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
		lease, err := q.claimHigh(ctx, 100*time.Millisecond)
		if err == nil {
			return lease, nil
		}
		if errors.Is(err, errHighJobDemoted) {
			continue
		}
		if !isEmpty(err) {
			return nil, err
		}
	}
}

func (q *Queue) claimHigh(ctx context.Context, wait time.Duration) (*Lease, error) {
	lease, err := q.fetch(ctx, q.high, wait)
	if err != nil {
		return nil, err
	}
	if !lease.Job.EnqueuedAt.IsZero() {
		if delay := time.Until(lease.Job.EnqueuedAt.Add(q.priorityWindow)); delay > 0 {
			if err := lease.NakWithDelay(delay); err != nil {
				return nil, fmt.Errorf("defer antivirus job until its rate window closes: %w", err)
			}
			return nil, jetstream.ErrNoMessages
		}
	}

	priority, err := q.classifier.CurrentPriority(ctx, userIdentity(lease.Job.Event.ExecutingUser), time.Now())
	if err != nil {
		_ = lease.NakWithDelay(time.Second)
		return nil, err
	}
	if priority == PriorityLow {
		lowJob := lease.Job
		lowJob.Priority = PriorityLow
		if err := q.enqueueWithMessageID(ctx, lowJob, lowJob.ID+"-low"); err != nil {
			_ = lease.NakWithDelay(time.Second)
			return nil, err
		}
		if err := lease.Ack(); err != nil {
			q.report(fmt.Errorf("acknowledge demoted antivirus job: %w", err))
		}
		return nil, errHighJobDemoted
	}
	return lease, nil
}

func (q *Queue) fetch(ctx context.Context, consumer jetstream.Consumer, wait time.Duration) (*Lease, error) {
	var (
		batch jetstream.MessageBatch
		err   error
	)
	if wait == 0 {
		batch, err = consumer.FetchNoWait(1)
	} else {
		batch, err = consumer.Fetch(1, jetstream.FetchMaxWait(wait))
	}
	if err != nil {
		return nil, err
	}
	for msg := range batch.Messages() {
		var job Job
		if err := json.Unmarshal(msg.Data(), &job); err != nil {
			_ = msg.Term()
			q.report(fmt.Errorf("discard malformed antivirus job: %w", err))
			return nil, jetstream.ErrNoMessages
		}
		queueWait.WithLabelValues(string(job.Priority)).Observe(time.Since(job.EnqueuedAt).Seconds())
		return &Lease{Job: job, msg: msg}, nil
	}
	if err := batch.Error(); err != nil {
		return nil, err
	}
	return nil, jetstream.ErrNoMessages
}

func isEmpty(err error) bool {
	return errors.Is(err, nats.ErrTimeout) || errors.Is(err, jetstream.ErrNoMessages)
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

func (q *Queue) AckWait() time.Duration { return q.ackWait }

// Close releases the input subscription and NATS connection.
func (q *Queue) Close() {
	q.closeOnce.Do(func() {
		q.cancel()
		if q.input != nil {
			_ = q.input.Unsubscribe()
		}
		q.inputWG.Wait()
		q.monitorWG.Wait()
		if q.conn != nil {
			q.conn.Close()
		}
	})
}
