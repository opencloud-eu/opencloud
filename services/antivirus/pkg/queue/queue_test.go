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
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	microevents "go-micro.dev/v4/events"

	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/config"
	revaevents "github.com/opencloud-eu/reva/v2/pkg/events"
)

func TestQueueClassifiesHeavyUserAndAlwaysSelectsHighFirst(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
		HighPriorityReservedWorkers: 1,
		PriorityThreshold:           2,
		PriorityWindow:              time.Second,
		PriorityCooldown:            time.Minute,
		QueueAckWait:                5 * time.Second,
		QueueReplicas:               1,
	}
	q, err := Open(ctx, cfg, func(err error) { t.Errorf("queue callback: %v", err) })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(q.Close)
	secondReplica, err := Open(ctx, cfg, func(err error) { t.Errorf("second replica callback: %v", err) })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(secondReplica.Close)

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
		id   string
		user string
	}{
		{id: "alice-1", user: "alice"},
		{id: "alice-2", user: "alice"},
		{id: "alice-3", user: "alice"},
		{id: "bob-1", user: "bob"},
	} {
		start := revaevents.StartPostprocessingStep{
			UploadID:      upload.id,
			ExecutingUser: &user.User{Id: &user.UserId{Idp: "test", OpaqueId: upload.user}},
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
		info, err := q.jobs.Info(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if info.State.Msgs == 4 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("priority queue contains %d jobs, want 4", info.State.Msgs)
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(1100 * time.Millisecond) // Allow the rate window to close before workers dispatch jobs.

	gotHighByUser := map[string]int{}
	gotLowByUser := map[string]int{}
	for i := 0; i < 4; i++ {
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
	if gotHighByUser["alice"] != 0 || gotHighByUser["bob"] != 1 || gotLowByUser["alice"] != 3 {
		t.Fatalf("priority jobs by user: high=%v low=%v, want only bob high and all alice jobs low", gotHighByUser, gotLowByUser)
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
	if got, want := mainInfo.State.Msgs, uint64(5); got != want {
		t.Fatalf("main stream contains %d messages after duplicate completion, want %d", got, want)
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
}
