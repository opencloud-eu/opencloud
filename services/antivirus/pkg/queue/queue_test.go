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

	cfg := &config.Config{
		Service:                     config.Service{Name: "antivirus-test"},
		Events:                      config.Events{Endpoint: natsServer.ClientURL()},
		Workers:                     10,
		QueueIntakeWorkers:          10,
		HighPriorityReservedWorkers: 1,
		PriorityThreshold:           2,
		PriorityWindow:              100 * time.Millisecond,
		PriorityCooldown:            time.Minute,
		QueueAckWait:                5 * time.Second,
		QueueReplicas:               1,
	}
	q, err := Open(ctx, cfg, reportError)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(q.Close)
	replicaConnection, err := nats.Connect(natsServer.ClientURL())
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
	secondReplicaClassifier := newClassifier(sharedRateStore, cfg.PriorityThreshold, cfg.PriorityWindow, cfg.PriorityCooldown)
	classifyAt := time.Now().UTC()
	if priority, err := q.classifier.Classify(ctx, "shared-classifier-user", "classify-1", classifyAt, classifyAt); err != nil || priority != PriorityHigh {
		t.Fatalf("first replica classification = %q, err=%v; want high", priority, err)
	}
	if priority, err := secondReplicaClassifier.Classify(ctx, "shared-classifier-user", "classify-2", classifyAt.Add(10*time.Millisecond), classifyAt.Add(10*time.Millisecond)); err != nil || priority != PriorityHigh {
		t.Fatalf("second replica classification = %q, err=%v; want high", priority, err)
	}
	if priority, err := q.classifier.Classify(ctx, "shared-classifier-user", "classify-3", classifyAt.Add(20*time.Millisecond), classifyAt.Add(20*time.Millisecond)); err != nil || priority != PriorityLow {
		t.Fatalf("shared rate classification = %q, err=%v; want low", priority, err)
	}
	if priority, err := q.classifier.Classify(ctx, "shared-classifier-user", "classify-1", classifyAt, classifyAt.Add(30*time.Millisecond)); err != nil || priority != PriorityHigh {
		t.Fatalf("redelivered event classification = %q, err=%v; want its original high classification", priority, err)
	}

	publisher, err := nats.Connect(natsServer.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(publisher.Close)
	js, err := publisher.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	startedAt := time.Now().UTC()
	for _, upload := range []struct {
		id         string
		user       string
		resourceID string
	}{
		{id: "alice-1", user: "alice", resourceID: "shared-sheet"},
		{id: "alice-2", user: "alice", resourceID: "shared-sheet"},
		{id: "alice-3", user: "alice", resourceID: "shared-sheet"},
		{id: "bob-1", user: "bob", resourceID: "bob-file"},
		{id: "bob-2", user: "bob", resourceID: "shared-sheet"},
	} {
		start := revaevents.StartPostprocessingStep{
			UploadID:      upload.id,
			ExecutingUser: &user.User{Id: &user.UserId{Idp: "test", OpaqueId: upload.user}},
			ResourceID:    &provider.ResourceId{StorageId: "test-storage", SpaceId: "test-space", OpaqueId: upload.resourceID},
			Filename:      upload.id + ".xlsx",
			StepToStart:   revaevents.PPStepAntivirus,
		}
		payload, err := json.Marshal(start)
		if err != nil {
			t.Fatal(err)
		}
		eventID := "event-" + upload.id
		envelope, err := json.Marshal(microevents.Event{
			ID:        eventID,
			Topic:     revaevents.MainQueueName,
			Timestamp: startedAt,
			Metadata: map[string]string{
				revaevents.MetadatakeyEventType: reflect.TypeOf(revaevents.StartPostprocessingStep{}).String(),
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

	deadline := time.Now().Add(5 * time.Second)
	for {
		highInfo, err := q.high.Info(ctx)
		if err != nil {
			t.Fatal(err)
		}
		lowInfo, err := q.low.Info(ctx)
		if err != nil {
			t.Fatal(err)
		}
		outstanding := highInfo.NumPending + uint64(highInfo.NumAckPending) + lowInfo.NumPending + uint64(lowInfo.NumAckPending)
		if outstanding == 5 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("priority consumers have %d outstanding jobs, want 5", outstanding)
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(150 * time.Millisecond) // Allow the rate window to close before workers dispatch jobs.

	gotHighByUser := map[string]int{}
	gotLowByUser := map[string]int{}
	var selectedUploads []string
	publishUploadReady := func(uploadID string) {
		readyID := "ready-" + uploadID
		payload, err := json.Marshal(revaevents.UploadReady{UploadID: uploadID})
		if err != nil {
			t.Fatal(err)
		}
		envelope, err := json.Marshal(microevents.Event{
			ID:        readyID,
			Topic:     revaevents.MainQueueName,
			Timestamp: time.Now().UTC(),
			Metadata: map[string]string{
				revaevents.MetadatakeyEventType: reflect.TypeOf(revaevents.UploadReady{}).String(),
				revaevents.MetadatakeyEventID:   readyID,
			},
			Payload: payload,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := js.Publish(revaevents.MainQueueName, envelope); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(5 * time.Second)
		for {
			_, err := q.resourceOrder.store.Get(ctx, uploadOrderKey(uploadID))
			if errors.Is(err, jetstream.ErrKeyNotFound) {
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if time.Now().After(deadline) {
				t.Fatalf("UploadReady did not release the resource queue for %q", uploadID)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	for i := 0; i < 5; i++ {
		lease, err := q.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		userID := lease.Job.Event.ExecutingUser.GetId().GetOpaqueId()
		selectedUploads = append(selectedUploads, lease.Job.Event.UploadID)
		if lease.Job.Priority == PriorityHigh {
			gotHighByUser[userID]++
		} else {
			gotLowByUser[userID]++
		}
		if err := lease.Ack(); err != nil {
			t.Fatal(fmt.Errorf("ack %s: %w", lease.Job.Event.UploadID, err))
		}
		publishUploadReady(lease.Job.Event.UploadID)
	}
	if gotHighByUser["alice"] != 0 || gotHighByUser["bob"] != 2 || gotLowByUser["alice"] != 3 {
		t.Fatalf("priority jobs by user: high=%v low=%v, want bob high and all alice jobs low", gotHighByUser, gotLowByUser)
	}
	wantUploads := []string{"bob-1", "alice-1", "alice-2", "alice-3", "bob-2"}
	for index := range wantUploads {
		if selectedUploads[index] != wantUploads[index] {
			t.Fatalf("scan order = %v, want %v (preserve same-resource FIFO)", selectedUploads, wantUploads)
		}
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
	if got, want := mainInfo.State.Msgs, uint64(11); got != want {
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
	retryResource := classifierKey("resource-order-retry-test")
	if err := q.resourceOrder.Register(ctx, retryResource, "attempt-1", "retry-upload", 100); err != nil {
		t.Fatal(err)
	}
	if err := q.resourceOrder.Register(ctx, retryResource, "later-upload", "later-upload", 101); err != nil {
		t.Fatal(err)
	}
	if err := q.resourceOrder.Register(ctx, retryResource, "attempt-2", "retry-upload", 200); err != nil {
		t.Fatal(err)
	}
	retryJob := Job{ID: "attempt-2", ResourceKey: retryResource, SourceSequence: 200, Event: revaevents.StartPostprocessingStep{UploadID: "retry-upload"}}
	claim, token, err := q.resourceOrder.Claim(ctx, retryJob)
	if err != nil || claim != resourceReady || token == "" {
		t.Fatalf("retry claim = (%v, %q, %v), want ready with a lease", claim, token, err)
	}
	laterJob := Job{ID: "later-upload", ResourceKey: retryResource, SourceSequence: 101, Event: revaevents.StartPostprocessingStep{UploadID: "later-upload"}}
	claim, _, err = q.resourceOrder.Claim(ctx, laterJob)
	if err != nil || claim != resourceBlocked {
		t.Fatalf("later resource claim = (%v, %v), want blocked behind retry", claim, err)
	}
	if err := q.resourceOrder.Release(ctx, retryJob, token); err != nil {
		t.Fatal(err)
	}
	finalRetry := Job{ID: "attempt-3", ResourceKey: retryResource, SourceSequence: 300, Event: revaevents.StartPostprocessingStep{UploadID: "retry-upload"}}
	if err := q.resourceOrder.Register(ctx, retryResource, finalRetry.ID, finalRetry.Event.UploadID, finalRetry.SourceSequence); err != nil {
		t.Fatal(err)
	}
	claim, token, err = q.resourceOrder.Claim(ctx, finalRetry)
	if err != nil || claim != resourceReady || token == "" {
		t.Fatalf("final retry claim = (%v, %q, %v), want ready", claim, token, err)
	}
	if err := q.resourceOrder.FinishUpload(ctx, finalRetry.Event.UploadID); err != nil {
		t.Fatal(err)
	}
	claim, token, err = q.resourceOrder.Claim(ctx, laterJob)
	if err != nil || claim != resourceReady || token == "" {
		t.Fatalf("later resource claim after retry = (%v, %q, %v), want ready", claim, token, err)
	}
	if err := q.resourceOrder.Complete(ctx, laterJob, token); err != nil {
		t.Fatal(err)
	}

	contextJob := Job{
		ID:          "context-cancel-job",
		ResourceKey: classifierKey("context-cancel-resource"),
		Priority:    PriorityLow,
		EnqueuedAt:  time.Now().UTC(),
	}
	if err := q.resourceOrder.Register(ctx, contextJob.ResourceKey, contextJob.ID, "", 0); err != nil {
		t.Fatal(err)
	}
	if err := q.enqueue(ctx, contextJob); err != nil {
		t.Fatal(err)
	}
	var contextLease *Lease
	deadline = time.Now().Add(5 * time.Second)
	for contextLease == nil {
		select {
		case contextLease = <-q.lowReady:
		case <-time.After(10 * time.Millisecond):
			if time.Now().After(deadline) {
				t.Fatal("timed out waiting for context-check job")
			}
		}
	}
	canceledCtx, cancelContext := context.WithCancel(ctx)
	cancelContext()
	if _, _, err := q.claimResource(canceledCtx, contextLease); !errors.Is(err, context.Canceled) {
		t.Fatalf("resource claim error = %v, want canceled caller context", err)
	}

	q.Close()
	select {
	case err := <-callbackErrors:
		t.Errorf("queue callback reported an error: %v", err)
	default:
	}
}
