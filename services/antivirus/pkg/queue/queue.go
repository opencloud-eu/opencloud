// Package queue implements the durable, per-user priority scheduling for
// antivirus scan requests.
package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/opencloud-eu/reva/v2/pkg/bytesize"
	revaevents "github.com/opencloud-eu/reva/v2/pkg/events"

	"github.com/opencloud-eu/opencloud/pkg/generators"
	ocnats "github.com/opencloud-eu/opencloud/pkg/nats"
	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/config"
)

const (
	inputConsumer       = "antivirus"
	inputBufferSize     = 1000
	inputRetryDelay     = time.Second
	jobStream           = "OPENCLOUD_ANTIVIRUS_JOBS"
	jobStreamMaxAge     = 7 * 24 * time.Hour
	rateBucket          = "ANTIVIRUS_USER_RATES"
	rateStateTTL        = 24 * time.Hour
	duplicateWindow     = config.MainEventStreamDuplicateWindow
	monitorInterval     = 5 * time.Second
	connectPollInterval = 100 * time.Millisecond
	readerGoroutines    = 2
	highSubject         = "opencloud.antivirus.jobs.high"
	lowSubject          = "opencloud.antivirus.jobs.low"
	highConsumer        = "antivirus-jobs-high"
	lowConsumer         = "antivirus-jobs-low"
)

// Job is a durable copy of an antivirus postprocessing request.
type Job struct {
	ID          string                             `json:"id"`
	Event       revaevents.StartPostprocessingStep `json:"event"`
	TraceParent string                             `json:"trace_parent,omitempty"`
	InitiatorID string                             `json:"initiator_id,omitempty"`
	Priority    Priority                           `json:"priority"`
	EnqueuedAt  time.Time                          `json:"enqueued_at"`
}

// Lease is a claimed scan job. The message remains unacknowledged until the
// scan result has been durably published to the main event stream.
type Lease struct {
	Job   Job
	msg   jetstream.Msg
	queue *Queue
}

// Ack acknowledges the job and removes it from the per-user outstanding
// backlog. It must only be called for jobs that are finally done.
func (l *Lease) Ack() error {
	if l.queue != nil {
		l.queue.finishOutstanding(l.Job)
		l.queue.metrics.JobsCompleted.WithLabelValues(string(l.Job.Priority)).Inc()
	}
	return l.msg.Ack()
}

// InProgress extends the JetStream acknowledgement deadline of the job.
func (l *Lease) InProgress() error { return l.msg.InProgress() }

// NakWithDelay releases the job for a later redelivery.
func (l *Lease) NakWithDelay(delay time.Duration) error { return l.msg.NakWithDelay(delay) }

// ackRaw acknowledges without touching the outstanding backlog accounting.
// It is used when a high-priority job is moved to the low lane: the low-lane
// copy is still outstanding until it has been scanned.
func (l *Lease) ackRaw() error { return l.msg.Ack() }

// Queue moves antivirus requests out of the shared event stream into durable,
// strict-priority JetStream lanes.
type Queue struct {
	ctx             context.Context
	cancel          context.CancelFunc
	conn            *nats.Conn
	legacyJS        nats.JetStreamContext
	js              jetstream.JetStream
	classifier      *Classifier
	priorityEnabled bool
	high            jetstream.Consumer
	low             jetstream.Consumer
	ackWait         time.Duration
	input           *nats.Subscription
	inputMsgs       chan *nats.Msg
	inputWG         sync.WaitGroup
	highMessages    jetstream.MessagesContext
	lowMessages     jetstream.MessagesContext
	highReady       chan *Lease
	lowReady        chan *Lease
	readerWG        sync.WaitGroup
	monitorWG       sync.WaitGroup
	lastMonitorErr  atomic.Int64
	metrics         *Metrics
	onError         func(error)
	closeOnce       sync.Once
}

// Open creates or binds the durable streams and consumers used by the
// antivirus priority scheduler.
func Open(ctx context.Context, cfg *config.Config, onError func(error)) (*Queue, error) {
	if cfg == nil {
		return nil, errors.New("antivirus config is nil")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	maxOutstandingBytes, err := bytesize.Parse(cfg.PriorityMaxOutstandingBytes)
	if err != nil {
		return nil, fmt.Errorf("parse ANTIVIRUS_PRIORITY_MAX_OUTSTANDING_BYTES: %w", err)
	}
	options := []nats.Option{
		nats.Name(generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus) + ":priority-queue"),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
	}
	if cfg.Events.AuthUsername != "" && cfg.Events.AuthPassword != "" {
		options = append(options, nats.UserInfo(cfg.Events.AuthUsername, cfg.Events.AuthPassword))
	}
	if secure := ocnats.Secure(cfg.Events.EnableTLS, cfg.Events.TLSInsecure, cfg.Events.TLSRootCACertificate); secure != nil {
		options = append(options, secure)
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
	if err := waitForNATSConnection(queueCtx, conn); err != nil {
		return nil, fmt.Errorf("wait for antivirus priority queue NATS connection: %w", err)
	}

	legacyJS, err := conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("create NATS JetStream context: %w", err)
	}
	js, err := jetstream.New(conn)
	if err != nil {
		return nil, fmt.Errorf("create NATS JetStream client: %w", err)
	}
	if err := ensureMainEventStream(queueCtx, js, onError); err != nil {
		return nil, err
	}

	jobs, err := js.CreateOrUpdateStream(queueCtx, jetstream.StreamConfig{
		Name:       jobStream,
		Subjects:   []string{highSubject, lowSubject},
		Retention:  jetstream.WorkQueuePolicy,
		Storage:    jetstream.FileStorage,
		MaxAge:     jobStreamMaxAge,
		Duplicates: duplicateWindow,
		Replicas:   cfg.QueueReplicas,
	})
	if err != nil {
		return nil, fmt.Errorf("create antivirus job stream: %w", err)
	}

	rateStore, err := js.CreateOrUpdateKeyValue(queueCtx, jetstream.KeyValueConfig{
		Bucket:      rateBucket,
		Description: "Shared per-user antivirus priority rate and backlog state",
		History:     1,
		TTL:         rateStateTTL,
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
	highMessages, err := high.Messages(jetstream.PullMaxMessages(1))
	if err != nil {
		return nil, fmt.Errorf("create high-priority antivirus message iterator: %w", err)
	}
	lowMessages, err := low.Messages(jetstream.PullMaxMessages(1))
	if err != nil {
		highMessages.Stop()
		return nil, fmt.Errorf("create low-priority antivirus message iterator: %w", err)
	}

	q := &Queue{
		ctx:             queueCtx,
		cancel:          cancel,
		conn:            conn,
		legacyJS:        legacyJS,
		js:              js,
		classifier:      newClassifier(rateStore, cfg.PriorityThreshold, cfg.PriorityWindow, cfg.PriorityCooldown, int64(cfg.PriorityMaxOutstandingJobs), maxOutstandingBytes.Bytes(), cfg.PriorityEnabled),
		priorityEnabled: cfg.PriorityEnabled,
		high:            high,
		low:             low,
		ackWait:         cfg.QueueAckWait,
		inputMsgs:       make(chan *nats.Msg, inputBufferSize),
		highMessages:    highMessages,
		lowMessages:     lowMessages,
		highReady:       make(chan *Lease, 1),
		lowReady:        make(chan *Lease, 1),
		metrics:         newMetrics(),
		onError:         onError,
	}
	if err := q.subscribeInput(); err != nil {
		return nil, err
	}
	for range cfg.QueueIntakeWorkers {
		q.inputWG.Add(1)
		go q.intake()
	}
	q.readerWG.Add(readerGoroutines)
	go q.readJobs(q.highMessages, q.highReady)
	go q.readJobs(q.lowMessages, q.lowReady)
	q.monitorWG.Add(1)
	go q.monitor()
	closeOnError = false
	return q, nil
}

func waitForNATSConnection(ctx context.Context, conn *nats.Conn) error {
	ticker := time.NewTicker(connectPollInterval)
	defer ticker.Stop()
	for {
		if conn.IsConnected() {
			return nil
		}
		if conn.IsClosed() {
			if err := conn.LastError(); err != nil {
				return err
			}
			return errors.New("NATS connection closed before connecting")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// ensureMainEventStream makes sure the shared main stream exists and is
// configured with a duplicate window large enough for stable completion event
// IDs. Failing to update an existing stream is not fatal: scans still work,
// only the duplicate protection is weaker, so the error is reported instead.
func ensureMainEventStream(ctx context.Context, js jetstream.JetStream, onError func(error)) error {
	main, err := js.Stream(ctx, revaevents.MainQueueName)
	if errors.Is(err, jetstream.ErrStreamNotFound) {
		main, err = js.CreateStream(ctx, jetstream.StreamConfig{
			Name:       revaevents.MainQueueName,
			Subjects:   []string{revaevents.MainQueueName},
			MaxAge:     jobStreamMaxAge,
			Duplicates: duplicateWindow,
		})
		if errors.Is(err, jetstream.ErrStreamNameAlreadyInUse) {
			main, err = js.Stream(ctx, revaevents.MainQueueName)
		}
	}
	if err != nil {
		return fmt.Errorf("ensure OpenCloud main event stream: %w", err)
	}
	info, err := main.Info(ctx)
	if err != nil {
		return fmt.Errorf("inspect OpenCloud main event stream: %w", err)
	}
	if info.Config.Duplicates >= duplicateWindow {
		return nil
	}
	streamConfig := info.Config
	streamConfig.Duplicates = duplicateWindow
	if _, err := js.UpdateStream(ctx, streamConfig); err != nil {
		if onError != nil {
			onError(fmt.Errorf("configure OpenCloud main event deduplication: %w", err))
		}
		return nil
	}
	return nil
}
