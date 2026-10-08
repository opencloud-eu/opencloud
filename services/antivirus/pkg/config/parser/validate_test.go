package parser

import (
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/config/defaults"
)

func TestValidateDefaultConfiguration(t *testing.T) {
	cfg := defaults.FullDefaultConfig()
	if err := Validate(cfg); err != nil {
		t.Fatalf("default config is invalid: %v", err)
	}
}

func TestSingleWorkerDefaultsToNoReservedWorker(t *testing.T) {
	cfg := defaults.DefaultConfig()
	cfg.Workers = 1
	defaults.Sanitize(cfg)
	if cfg.HighPriorityReservedWorkers != 0 {
		t.Fatalf("reserved workers = %d, want 0 for a single scan worker", cfg.HighPriorityReservedWorkers)
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("single-worker config is invalid: %v", err)
	}
}

func TestValidateRejectsQueueAckWaitBeyondDedupWindow(t *testing.T) {
	cfg := defaults.FullDefaultConfig()
	cfg.QueueAckWait = 2 * time.Minute
	if err := Validate(cfg); err == nil {
		t.Fatal("expected queue ack wait at or above the deduplication window to be rejected")
	}
}
