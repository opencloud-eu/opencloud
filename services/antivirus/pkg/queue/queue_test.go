package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	microevents "go-micro.dev/v4/events"

	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/config"
	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/config/defaults"
	revaevents "github.com/opencloud-eu/reva/v2/pkg/events"
)

func startTestNATS(t *testing.T) (*server.Server, *nats.Conn) {
	t.Helper()
	natsServer, err := server.NewServer(&server.Options{
		Host:      "127.0.0.1",
		Port:      -1,
		JetStream: true,
		StoreDir:  t.TempDir(),
		NoSigs:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	go natsServer.Start()
	t.Cleanup(natsServer.Shutdown)
	if !natsServer.ReadyForConnections(10 * time.Second) {
		t.Fatal("embedded NATS server did not become ready")
	}
	conn, err := nats.Connect(natsServer.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(conn.Close)
	return natsServer, conn
}

func testQueueConfig(endpoint string) *config.Config {
	return &config.Config{
		Service:                     config.Service{Name: "antivirus-test"},
		Events:                      config.Events{Endpoint: endpoint},
		Workers:                     10,
		QueueIntakeWorkers:          10,
		HighPriorityReservedWorkers: 1,
		PriorityEnabled:             true,
		PriorityThreshold:           2,
		PriorityWindow:              100 * time.Millisecond,
		PriorityCooldown:            time.Minute,
		PriorityMaxOutstandingJobs:  1000,
		PriorityMaxOutstandingBytes: "1GB",
		QueueAckWait:                5 * time.Second,
		QueueReplicas:               1,
	}
}

func publishStartEvent(t *testing.T, js nats.JetStreamContext, eventID, uploadID, userName string, filesize uint64) {
	t.Helper()
	start := revaevents.StartPostprocessingStep{
		UploadID:      uploadID,
		ExecutingUser: &user.User{Id: &user.UserId{Idp: "test", OpaqueId: userName}},
		ResourceID:    &provider.ResourceId{StorageId: "test-storage", SpaceId: "test-space", OpaqueId: uploadID},
		Filename:      uploadID + ".xlsx",
		Filesize:      filesize,
		StepToStart:   revaevents.PPStepAntivirus,
	}
	payload, err := json.Marshal(start)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := json.Marshal(microevents.Event{
		ID:        eventID,
		Topic:     revaevents.MainQueueName,
		Timestamp: time.Now().UTC(),
		Metadata: map[string]string{
			revaevents.MetadatakeyEventType: reflect.TypeFor[revaevents.StartPostprocessingStep]().String(),
			revaevents.MetadatakeyEventID:   eventID,
		},
		Payload: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := js.Publish(revaevents.MainQueueName, envelope); err != nil {
		t.Fatal(err)
	}
}

func waitForOutstanding(t *testing.T, q *Queue, want uint64) {
	t.Helper()
	ctx := context.Background()
	// Use dedicated consumer handles: nats.go Consumer.Info mutates per-handle
	// state and the monitor goroutine uses the queue's own handles.
	high, err := q.js.Consumer(ctx, jobStream, highConsumer)
	if err != nil {
		t.Fatal(err)
	}
	low, err := q.js.Consumer(ctx, jobStream, lowConsumer)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		highInfo, err := high.Info(ctx)
		if err != nil {
			t.Fatal(err)
		}
		lowInfo, err := low.Info(ctx)
		if err != nil {
			t.Fatal(err)
		}
		outstanding := highInfo.NumPending + uint64(highInfo.NumAckPending) + lowInfo.NumPending + uint64(lowInfo.NumAckPending)
		if outstanding == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("priority consumers have %d outstanding jobs, want %d", outstanding, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestOpenWaitsForNATSConnectionUntilContextCancellation(t *testing.T) {
	cfg := defaults.FullDefaultConfig()
	cfg.Events.Endpoint = "nats://127.0.0.1:1"
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := Open(ctx, cfg, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Open error = %v, want context deadline while waiting for NATS", err)
	}
}

func TestQueueClassifiesHeavyUserAndAlwaysSelectsHighFirst(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	callbackErrors := make(chan error, 16)
	reportError := func(err error) {
		select {
		case callbackErrors <- err:
		default:
		}
	}

	natsServer, publisher := startTestNATS(t)
	_ = natsServer
	cfg := testQueueConfig(publisher.ConnectedUrl())
	q, err := Open(ctx, cfg, reportError)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(q.Close)

	// The classifier state is shared through JetStream KV, so a second replica
	// must see the same per-user budget.
	replicaConnection, err := nats.Connect(publisher.ConnectedUrl())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(replicaConnection.Close)
	replicaJS, err := jetstream.New(replicaConnection)
	if err != nil {
		t.Fatal(err)
	}
	sharedRateStore, err := replicaJS.KeyValue(ctx, rateBucket)
	if err != nil {
		t.Fatal(err)
	}
	secondReplicaClassifier := newClassifier(sharedRateStore, cfg.PriorityThreshold, cfg.PriorityWindow, cfg.PriorityCooldown, int64(cfg.PriorityMaxOutstandingJobs), 1<<30, true)
	classifyAt := time.Now().UTC()
	if priority, err := q.classifier.Classify(ctx, "shared-classifier-user", "classify-1", classifyAt, classifyAt, 1); err != nil || priority != PriorityHigh {
		t.Fatalf("first replica classification = %q, err=%v; want high", priority, err)
	}
	if priority, err := secondReplicaClassifier.Classify(ctx, "shared-classifier-user", "classify-2", classifyAt.Add(10*time.Millisecond), classifyAt.Add(10*time.Millisecond), 1); err != nil || priority != PriorityHigh {
		t.Fatalf("second replica classification = %q, err=%v; want high", priority, err)
	}
	if priority, err := q.classifier.Classify(ctx, "shared-classifier-user", "classify-3", classifyAt.Add(20*time.Millisecond), classifyAt.Add(20*time.Millisecond), 1); err != nil || priority != PriorityLow {
		t.Fatalf("shared rate classification = %q, err=%v; want low", priority, err)
	}
	if priority, err := q.classifier.Classify(ctx, "shared-classifier-user", "classify-1", classifyAt, classifyAt.Add(30*time.Millisecond), 1); err != nil || priority != PriorityHigh {
		t.Fatalf("redelivered event classification = %q, err=%v; want its original high classification", priority, err)
	}

	js, err := publisher.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	for _, upload := range []struct {
		id   string
		user string
	}{
		{id: "alice-1", user: "alice"},
		{id: "alice-2", user: "alice"},
		{id: "alice-3", user: "alice"},
		{id: "bob-1", user: "bob"},
		{id: "bob-2", user: "bob"},
	} {
		publishStartEvent(t, js, "event-"+upload.id, upload.id, upload.user, 1024)
	}
	waitForOutstanding(t, q, 5)
	time.Sleep(150 * time.Millisecond) // Let the rate window close before workers dispatch jobs.

	gotHighByUser := map[string]int{}
	gotLowByUser := map[string]int{}
	for i := 0; i < 5; i++ {
		lease, err := q.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		userID := lease.Job.Event.ExecutingUser.GetId().GetOpaqueId()
		if lease.Job.Priority == PriorityHigh {
			gotHighByUser[userID]++
		} else {
			gotLowByUser[userID]++
		}
		if err := lease.Ack(); err != nil {
			t.Fatal(fmt.Errorf("ack %s: %w", lease.Job.Event.UploadID, err))
		}
	}
	if gotHighByUser["alice"] != 0 || gotHighByUser["bob"] != 2 || gotLowByUser["alice"] != 3 {
		t.Fatalf("priority jobs by user: high=%v low=%v, want bob high and all alice jobs low", gotHighByUser, gotLowByUser)
	}

	finished := revaevents.PostprocessingStepFinished{
		UploadID:     "alice-1",
		FinishedStep: revaevents.PPStepAntivirus,
		Outcome:      revaevents.PPOutcomeContinue,
	}
	completionJob := Job{ID: "stable-completion-id"}
	if err := q.PublishFinished(ctx, completionJob, finished); err != nil {
		t.Fatal(err)
	}
	if err := q.PublishFinished(ctx, completionJob, finished); err != nil {
		t.Fatal(err)
	}
	mainInfo, err := q.legacyJS.StreamInfo(revaevents.MainQueueName)
	if err != nil {
		t.Fatal(err)
	}
	if mainInfo.Config.Duplicates < duplicateWindow {
		t.Fatalf("main event stream duplicate window = %s, want at least %s", mainInfo.Config.Duplicates, duplicateWindow)
	}
	if got, want := mainInfo.State.Msgs, uint64(6); got != want {
		t.Fatalf("main stream contains %d messages after completions, want %d", got, want)
	}

	lowJob := Job{
		ID:         "low-only-job",
		Priority:   PriorityLow,
		EnqueuedAt: time.Now().UTC(),
	}
	if err := q.enqueue(ctx, lowJob); err != nil {
		t.Fatal(err)
	}
	highOnlyCtx, cancelHighOnly := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancelHighOnly()
	if lease, err := q.NextHighPriority(highOnlyCtx); lease != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("reserved worker received low-priority job: lease=%v err=%v", lease, err)
	}
	lease, err := q.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if lease.Job.ID != lowJob.ID {
		t.Fatalf("regular worker selected job %q, want %q", lease.Job.ID, lowJob.ID)
	}
	if err := lease.Ack(); err != nil {
		t.Fatal(err)
	}

	canceledCtx, cancelContext := context.WithCancel(ctx)
	cancelContext()
	if lease, err := q.Next(canceledCtx); lease != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("Next with canceled context = (%v, %v), want context.Canceled", lease, err)
	}

	q.Close()
	select {
	case err := <-callbackErrors:
		t.Errorf("queue callback reported an error: %v", err)
	default:
	}
}

func TestQueueDemotesSlowBulkUploadByOutstandingBacklog(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	natsServer, publisher := startTestNATS(t)
	_ = natsServer
	cfg := testQueueConfig(publisher.ConnectedUrl())
	cfg.PriorityThreshold = 1000
	cfg.PriorityMaxOutstandingJobs = 2
	cfg.PriorityMaxOutstandingBytes = "1GB"
	q, err := Open(ctx, cfg, func(err error) { t.Logf("queue error: %v", err) })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(q.Close)

	js, err := publisher.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		publishStartEvent(t, js, fmt.Sprintf("slow-%d", i), fmt.Sprintf("slow-upload-%d", i), "slow-user", 1024)
	}
	waitForOutstanding(t, q, 4)

	high, low := 0, 0
	for i := 0; i < 4; i++ {
		lease, err := q.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if lease.Job.Priority == PriorityHigh {
			high++
		} else {
			low++
		}
		if err := lease.Ack(); err != nil {
			t.Fatal(err)
		}
	}
	if high != 0 || low != 4 {
		t.Fatalf("slow bulk upload priorities: high=%d low=%d, want all jobs demoted once the backlog budget is exceeded", high, low)
	}

	// Once the backlog has drained the user is eligible for the high lane again.
	publishStartEvent(t, js, "after-drain", "after-drain-upload", "slow-user", 1024)
	waitForOutstanding(t, q, 1)
	lease, err := q.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if lease.Job.Priority != PriorityHigh {
		t.Fatalf("job after backlog drain priority = %q, want high", lease.Job.Priority)
	}
	if err := lease.Ack(); err != nil {
		t.Fatal(err)
	}
}

func TestQueueDemotesLargeUploadByOutstandingBytes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	natsServer, publisher := startTestNATS(t)
	_ = natsServer
	cfg := testQueueConfig(publisher.ConnectedUrl())
	cfg.PriorityThreshold = 1000
	cfg.PriorityMaxOutstandingJobs = 1000
	cfg.PriorityMaxOutstandingBytes = "1MB"
	q, err := Open(ctx, cfg, func(err error) { t.Logf("queue error: %v", err) })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(q.Close)

	js, err := publisher.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	publishStartEvent(t, js, "small", "small-upload", "size-user", 1024)
	waitForOutstanding(t, q, 1)
	smallLease, err := q.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if smallLease.Job.Event.UploadID != "small-upload" || smallLease.Job.Priority != PriorityHigh {
		t.Fatalf("small upload = (%q, %q), want (small-upload, high)", smallLease.Job.Event.UploadID, smallLease.Job.Priority)
	}
	if err := smallLease.Ack(); err != nil {
		t.Fatal(err)
	}

	publishStartEvent(t, js, "large", "large-upload", "size-user", 2<<20)
	waitForOutstanding(t, q, 1)
	largeLease, err := q.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if largeLease.Job.Event.UploadID != "large-upload" || largeLease.Job.Priority != PriorityLow {
		t.Fatalf("large upload = (%q, %q), want (large-upload, low)", largeLease.Job.Event.UploadID, largeLease.Job.Priority)
	}
	if err := largeLease.Ack(); err != nil {
		t.Fatal(err)
	}
}

func TestQueuePriorityDisabledUsesHighLane(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	natsServer, publisher := startTestNATS(t)
	_ = natsServer
	cfg := testQueueConfig(publisher.ConnectedUrl())
	cfg.PriorityEnabled = false
	cfg.PriorityThreshold = 1
	cfg.PriorityMaxOutstandingJobs = 1
	q, err := Open(ctx, cfg, func(err error) { t.Logf("queue error: %v", err) })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(q.Close)

	js, err := publisher.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		publishStartEvent(t, js, fmt.Sprintf("disabled-%d", i), fmt.Sprintf("disabled-upload-%d", i), "disabled-user", 2<<30)
	}
	waitForOutstanding(t, q, 3)

	for i := 0; i < 3; i++ {
		lease, err := q.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if lease.Job.Priority != PriorityHigh {
			t.Fatalf("disabled priority mode returned %q, want high", lease.Job.Priority)
		}
		if err := lease.Ack(); err != nil {
			t.Fatal(err)
		}
	}
}
