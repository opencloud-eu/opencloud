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
	revaevents "github.com/opencloud-eu/reva/v2/pkg/events"

	"github.com/opencloud-eu/opencloud/pkg/generators"
	ocnats "github.com/opencloud-eu/opencloud/pkg/nats"
	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/config"
)

const (
	inputConsumer        = "antivirus"
	inputBufferSize      = 1000
	inputRetryDelay      = time.Second
	jobStream            = "OPENCLOUD_ANTIVIRUS_JOBS"
	jobStreamMaxAge      = 7 * 24 * time.Hour
	rateBucket           = "ANTIVIRUS_USER_RATES"
	rateStateTTL         = 24 * time.Hour
	resourceBucket       = "ANTIVIRUS_RESOURCE_ORDER"
	resourceStateTTL     = 7 * 24 * time.Hour
	resourceOpTimeout    = 5 * time.Second
	duplicateWindow      = config.MainEventStreamDuplicateWindow
	consumerPollWait     = 100 * time.Millisecond
	resourceBlockedWait  = 100 * time.Millisecond
	priorityPollInterval = 100 * time.Millisecond
	monitorInterval      = 5 * time.Second
	highSubject          = "opencloud.antivirus.jobs.high"
	lowSubject           = "opencloud.antivirus.jobs.low"
	highConsumer         = "antivirus-jobs-high"
	lowConsumer          = "antivirus-jobs-low"
)

// Job is a durable copy of an antivirus postprocessing request.
type Job struct {
	ID             string
	SourceSequence uint64
	ResourceKey    string
	Event          revaevents.StartPostprocessingStep
	TraceParent    string
	InitiatorID    string
	Priority       Priority
	EnqueuedAt     time.Time
}

// Lease is a claimed scan job. The message remains unacknowledged until the
// scan result has been durably published to the main event stream.
type Lease struct {
	Job           Job
	msg           jetstream.Msg
	resourceOrder *resourceOrder
	resourceToken string
}

func (l *Lease) Ack() error {
	if l.resourceOrder != nil {
		ctx, cancel := context.WithTimeout(context.Background(), resourceOpTimeout)
		defer cancel()
		var err error
		if l.Job.Event.UploadID != "" {
			err = l.resourceOrder.Release(ctx, l.Job, l.resourceToken)
		} else {
			err = l.resourceOrder.Complete(ctx, l.Job, l.resourceToken)
		}
		if err != nil {
			return err
		}
	}
	return l.msg.Ack()
}

func (l *Lease) InProgress() error {
	if l.resourceOrder != nil {
		ctx, cancel := context.WithTimeout(context.Background(), resourceOpTimeout)
		defer cancel()
		if err := l.resourceOrder.Renew(ctx, l.Job, l.resourceToken); err != nil {
			return err
		}
	}
	return l.msg.InProgress()
}

func (l *Lease) Validate(ctx context.Context) error {
	if l.resourceOrder == nil {
		return nil
	}
	return l.resourceOrder.Validate(ctx, l.Job, l.resourceToken)
}

func (l *Lease) NakWithDelay(delay time.Duration) error {
	var releaseErr error
	if l.resourceOrder != nil {
		ctx, cancel := context.WithTimeout(context.Background(), resourceOpTimeout)
		releaseErr = l.resourceOrder.Release(ctx, l.Job, l.resourceToken)
		cancel()
	}
	return errors.Join(releaseErr, l.msg.NakWithDelay(delay))
}

// Queue moves antivirus requests out of the shared event stream into durable,
// strict-priority JetStream lanes.
type Queue struct {
	ctx            context.Context
	cancel         context.CancelFunc
	conn           *nats.Conn
	legacyJS       nats.JetStreamContext
	js             jetstream.JetStream
	classifier     *Classifier
	resourceOrder  *resourceOrder
	priorityWindow time.Duration
	high           jetstream.Consumer
	low            jetstream.Consumer
	ackWait        time.Duration
	input          *nats.Subscription
	inputMsgs      chan *nats.Msg
	inputWG        sync.WaitGroup
	highMessages   jetstream.MessagesContext
	lowMessages    jetstream.MessagesContext
	highReady      chan *Lease
	lowReady       chan *Lease
	readerWG       sync.WaitGroup
	monitorWG      sync.WaitGroup
	highPending    atomic.Bool
	inputAckFloor  atomic.Uint64
	lastMonitorErr atomic.Int64
	metrics        *Metrics
	onError        func(error)
	closeOnce      sync.Once
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
	if err := ensureMainEventStream(queueCtx, js); err != nil {
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
		Description: "Shared per-user antivirus priority rate state",
		History:     1,
		TTL:         rateStateTTL,
		Storage:     jetstream.FileStorage,
		Replicas:    cfg.QueueReplicas,
	})
	if err != nil {
		return nil, fmt.Errorf("create antivirus priority rate bucket: %w", err)
	}
	resourceStore, err := js.CreateOrUpdateKeyValue(queueCtx, jetstream.KeyValueConfig{
		Bucket:      resourceBucket,
		Description: "Per-resource FIFO state for antivirus scans",
		History:     1,
		TTL:         resourceStateTTL,
		Storage:     jetstream.FileStorage,
		Replicas:    cfg.QueueReplicas,
	})
	if err != nil {
		return nil, fmt.Errorf("create antivirus resource order bucket: %w", err)
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
		ctx:            queueCtx,
		cancel:         cancel,
		conn:           conn,
		legacyJS:       legacyJS,
		js:             js,
		classifier:     newClassifier(rateStore, cfg.PriorityThreshold, cfg.PriorityWindow, cfg.PriorityCooldown),
		resourceOrder:  newResourceOrder(resourceStore, cfg.QueueAckWait),
		priorityWindow: cfg.PriorityWindow,
		high:           high,
		low:            low,
		ackWait:        cfg.QueueAckWait,
		inputMsgs:      make(chan *nats.Msg, inputBufferSize),
		highMessages:   highMessages,
		lowMessages:    lowMessages,
		highReady:      make(chan *Lease, 1),
		lowReady:       make(chan *Lease, 1),
		metrics:        newMetrics(),
		onError:        onError,
	}
	if err := q.subscribeInput(); err != nil {
		return nil, err
	}
	for range cfg.QueueIntakeWorkers {
		q.inputWG.Add(1)
		go q.intake()
	}
	q.readerWG.Add(2)
	go q.readJobs(q.highMessages, q.highReady)
	go q.readJobs(q.lowMessages, q.lowReady)
	q.monitorWG.Add(1)
	go q.monitor()
	closeOnError = false
	return q, nil
}

func waitForNATSConnection(ctx context.Context, conn *nats.Conn) error {
	ticker := time.NewTicker(100 * time.Millisecond)
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

func ensureMainEventStream(ctx context.Context, js jetstream.JetStream) error {
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
		return fmt.Errorf("configure OpenCloud main event deduplication: %w", err)
	}
	return nil
}
