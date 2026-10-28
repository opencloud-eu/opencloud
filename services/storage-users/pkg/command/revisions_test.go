package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/revisioncleanup"
	"github.com/test-go/testify/require"

	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
)

// revisionsCleanupOptionsFor parses the flags of the revisions cleanup command
func revisionsCleanupOptionsFor(t *testing.T, cfg *config.Config, args ...string) (revisionsCleanupOptions, error) {
	t.Helper()
	cmd, f := revisionsCleanupCommand(cfg)
	require.NoError(t, cmd.ParseFlags(args))
	return f.options(cfg, cmd.Flags().Changed)
}

func revisionsCleanupConfig() *config.Config {
	return &config.Config{
		Driver: "decomposed",
		RevisionsCleanup: config.RevisionsCleanup{
			Strategy:              "auto, 90d",
			Async:                 true,
			AsyncWorkers:          4,
			AsyncQueueSize:        100,
			AsyncQueueFull:        "drop",
			AsyncQueueFullTimeout: 5 * time.Second,
		},
	}
}

func TestRevisionsCleanupOptions(t *testing.T) {
	t.Run("the configuration is the default", func(t *testing.T) {
		o, err := revisionsCleanupOptionsFor(t, revisionsCleanupConfig(), "--all")
		require.NoError(t, err)
		require.True(t, o.request.All)
		require.Equal(t, "auto, 90d", o.request.Strategy)
		// except for a full queue: the command waits for room as long as it takes
		require.Equal(t, &revisioncleanup.AsyncOptions{Enabled: true, Workers: 4, QueueSize: 100, QueueFull: revisioncleanup.QueueFullBlock}, o.request.Async)
		require.False(t, o.request.DryRun)
		require.False(t, o.request.Shuffle)
		require.Zero(t, o.timeout)
	})

	t.Run("flags override the configuration", func(t *testing.T) {
		o, err := revisionsCleanupOptionsFor(t, revisionsCleanupConfig(),
			"--space", "storage$space1!space1", "--space", "space2", "--strategy", "10",
			"--async=false", "--workers", "0", "--queue-size", "7", "--queue-full", "drop", "--queue-full-timeout", "30s",
			"--dry-run", "--shuffle", "--timeout", "1h", "--verbose", "--json")
		require.NoError(t, err)
		require.Equal(t, []string{"space1", "space2"}, o.request.SpaceIDs, "the space ids as the APIs show them are accepted")
		require.Equal(t, "10", o.request.Strategy)
		require.Equal(t, &revisioncleanup.AsyncOptions{Enabled: false, Workers: 0, QueueSize: 7, QueueFull: revisioncleanup.QueueFullDrop, QueueFullTimeout: 30 * time.Second}, o.request.Async)
		require.True(t, o.request.DryRun)
		require.True(t, o.request.Shuffle)
		require.Equal(t, time.Hour, o.timeout)
		require.True(t, o.verbose)
		require.True(t, o.json)
	})

	t.Run("short flags", func(t *testing.T) {
		o, err := revisionsCleanupOptionsFor(t, revisionsCleanupConfig(), "-anvj", "-s", "10", "-t", "30m")
		require.NoError(t, err)
		require.True(t, o.request.All)
		require.True(t, o.request.DryRun)
		require.True(t, o.verbose)
		require.True(t, o.json)
		require.Equal(t, "10", o.request.Strategy)
		require.Equal(t, 30*time.Minute, o.timeout)

		cfg := revisionsCleanupConfig()
		cfg.Driver = "posix"
		o, err = revisionsCleanupOptionsFor(t, cfg, "-p", "/var/lib/opencloud/users/einstein")
		require.NoError(t, err)
		require.Equal(t, "/var/lib/opencloud/users/einstein", o.request.FSPath)
	})

	t.Run("a path within a space", func(t *testing.T) {
		o, err := revisionsCleanupOptionsFor(t, revisionsCleanupConfig(), "--space", "space1", "--path", "/Documents")
		require.NoError(t, err)
		require.Equal(t, "/Documents", o.request.Path)
	})

	t.Run("a path on disk with posix", func(t *testing.T) {
		cfg := revisionsCleanupConfig()
		cfg.Driver = "posix"
		o, err := revisionsCleanupOptionsFor(t, cfg, "--fs-path", "/var/lib/opencloud/users/einstein")
		require.NoError(t, err)
		require.Equal(t, "/var/lib/opencloud/users/einstein", o.request.FSPath)
	})

	for _, tc := range []struct {
		name     string
		strategy string
		args     []string
		err      string
	}{
		{name: "nothing to clean up", err: "no space"},
		{name: "a path without a space", args: []string{"--path", "/Documents"}, err: "--path requires --space"},
		{name: "all and a space", args: []string{"--all", "--space", "space1"}, err: "not both"},
		{name: "a path in two spaces", args: []string{"--space", "space1", "--space", "space2", "--path", "/Documents"}, err: "exactly one space"},
		{name: "a path on disk and a space", args: []string{"--fs-path", "/data", "--space", "space1"}, err: "can't be combined"},
		{name: "a path on disk without posix", args: []string{"--fs-path", "/data"}, err: "only supported by the 'posix' driver"},
		{name: "the cleanup is disabled", strategy: "disabled", args: []string{"--all"}, err: "--strategy"},
		{name: "an invalid strategy", args: []string{"--all", "--strategy", "whenever"}, err: "invalid strategy"},
		{name: "a negative timeout", args: []string{"--all", "--timeout", "-1s"}, err: "invalid timeout"},
	} {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			cfg := revisionsCleanupConfig()
			if tc.strategy != "" {
				cfg.RevisionsCleanup.Strategy = tc.strategy
			}
			_, err := revisionsCleanupOptionsFor(t, cfg, tc.args...)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.err)
		})
	}
}

// fakeOnDemand reports the given results, and then waits for the end of ctx if wait is set
type fakeOnDemand struct {
	results []revisioncleanup.Result
	wait    bool
	err     error
	request revisioncleanup.Request
}

func (f *fakeOnDemand) CleanupRevisionsUnchecked(ctx context.Context, req revisioncleanup.Request, report func(revisioncleanup.Result)) error {
	f.request = req
	for _, r := range f.results {
		r.DryRun = req.DryRun
		report(r)
	}
	if f.wait {
		<-ctx.Done()
		return ctx.Err()
	}
	return f.err
}

var (
	revisionsCleanupTime = time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	// a file with a deleted and a kept revision
	cleanedFile = revisioncleanup.Result{SpaceID: "space1", NodeID: "node1", Path: "/Documents/report.odt", Revisions: []revisioncleanup.RevisionResult{
		{Key: "node1.REV.1", Time: revisionsCleanupTime, Size: 1536, Decision: revisioncleanup.Kept, Reason: "kept by \"auto\""},
		{Key: "node1.REV.2", Time: revisionsCleanupTime.Add(-100 * 24 * time.Hour), Size: 2048, Decision: revisioncleanup.Deleted, Reason: "deleted by \"90d\""},
	}}
	// a file with nothing to delete
	untouchedFile = revisioncleanup.Result{SpaceID: "space1", NodeID: "node2", Path: "/notes.txt", Revisions: []revisioncleanup.RevisionResult{
		{Key: "node2.REV.1", Time: revisionsCleanupTime, Size: 10, Decision: revisioncleanup.Kept, Reason: "kept by \"auto\""},
	}}
	skippedFile = revisioncleanup.Result{SpaceID: "space2", NodeID: "node3", Path: "/upload.bin", Skipped: "an upload to the file is still being processed"}
	failedFile  = revisioncleanup.Result{SpaceID: "space2", NodeID: "node4", Path: "/broken.txt", Err: errors.New("could not list revisions")}
)

func runFakeRevisionsCleanup(t *testing.T, f *fakeOnDemand, o revisionsCleanupOptions) (int, string, error) {
	t.Helper()
	if o.request.Async == nil {
		o.request = revisioncleanup.Request{All: true, Strategy: "auto, 90d", DryRun: o.request.DryRun}
	}
	var out bytes.Buffer
	code, err := cleanupRevisions(context.Background(), f, o, &out)
	return code, out.String(), err
}

func TestRevisionsCleanupOutput(t *testing.T) {
	t.Run("only the files that changed or need attention, and a summary", func(t *testing.T) {
		code, out, err := runFakeRevisionsCleanup(t, &fakeOnDemand{results: []revisioncleanup.Result{cleanedFile, untouchedFile, skippedFile}}, revisionsCleanupOptions{})
		require.NoError(t, err)
		require.Equal(t, 0, code)
		require.Equal(t, `space1 /Documents/report.odt: deleted 1 of 2 revisions (2.0 KiB)
space2 /upload.bin: skipped: an upload to the file is still being processed
3 files in 2 spaces: deleted 1 revision (2.0 KiB) of 1 file, 1 skipped, 0 failed
`, out)
	})

	t.Run("a dry run", func(t *testing.T) {
		o := revisionsCleanupOptions{request: revisioncleanup.Request{DryRun: true}}
		code, out, err := runFakeRevisionsCleanup(t, &fakeOnDemand{results: []revisioncleanup.Result{cleanedFile}}, o)
		require.NoError(t, err)
		require.Equal(t, 0, code)
		require.Contains(t, out, "space1 /Documents/report.odt: would delete 1 of 2 revisions")
		require.Contains(t, out, "would delete 1 revision (2.0 KiB) of 1 file")
	})

	t.Run("every file and revision with --verbose", func(t *testing.T) {
		code, out, err := runFakeRevisionsCleanup(t, &fakeOnDemand{results: []revisioncleanup.Result{cleanedFile, untouchedFile}}, revisionsCleanupOptions{verbose: true})
		require.NoError(t, err)
		require.Equal(t, 0, code)
		// the times are in the local time zone, like the ones in the reasons
		local := func(t time.Time) string { return t.Local().Format(time.RFC3339) }
		require.Equal(t, `space1 /Documents/report.odt (node node1): deleted 1 of 2 revisions (2.0 KiB)
  kept      `+local(revisionsCleanupTime)+`   1.5 KiB  kept by "auto"
  deleted   `+local(revisionsCleanupTime.Add(-100*24*time.Hour))+`   2.0 KiB  deleted by "90d"
space1 /notes.txt (node node2): deleted 0 of 1 revision (0 B)
  kept      `+local(revisionsCleanupTime)+`      10 B  kept by "auto"
2 files in 1 space: deleted 1 revision (2.0 KiB) of 1 file, 0 skipped, 0 failed
`, out)
	})

	t.Run("JSON lines", func(t *testing.T) {
		code, out, err := runFakeRevisionsCleanup(t, &fakeOnDemand{results: []revisioncleanup.Result{cleanedFile, untouchedFile, failedFile}}, revisionsCleanupOptions{json: true})
		require.NoError(t, err)
		require.Equal(t, revisionsCleanupExitFailed, code)
		lines := strings.Split(strings.TrimSpace(out), "\n")
		require.Len(t, lines, 3, "the untouched file is left out without --verbose")

		var file fileJSON
		require.NoError(t, json.Unmarshal([]byte(lines[0]), &file))
		require.Equal(t, fileJSON{Type: "file", SpaceID: "space1", NodeID: "node1", Path: "/Documents/report.odt", Revisions: 2, Deleted: 1, DeletedBytes: 2048}, file)
		require.NoError(t, json.Unmarshal([]byte(lines[1]), &file))
		require.Equal(t, "could not list revisions", file.Error)

		var summary summaryJSON
		require.NoError(t, json.Unmarshal([]byte(lines[2]), &summary))
		require.Equal(t, summaryJSON{Type: "summary", Spaces: 2, Files: 3, Cleaned: 1, Deleted: 1, DeletedBytes: 2048, Failed: 1}, summary)
	})

	t.Run("JSON lines with the revisions with --verbose", func(t *testing.T) {
		_, out, err := runFakeRevisionsCleanup(t, &fakeOnDemand{results: []revisioncleanup.Result{cleanedFile}}, revisionsCleanupOptions{json: true, verbose: true})
		require.NoError(t, err)
		var file fileJSON
		require.NoError(t, json.Unmarshal([]byte(strings.Split(out, "\n")[0]), &file))
		require.Len(t, file.Details, 2)
		require.Equal(t, revisionJSON{Key: "node1.REV.2", Time: revisionsCleanupTime.Add(-100 * 24 * time.Hour), Size: 2048, Decision: "deleted", Reason: "deleted by \"90d\""}, file.Details[1])
	})
}

func TestRevisionsCleanupExitCodes(t *testing.T) {
	t.Run("a failed file", func(t *testing.T) {
		code, out, err := runFakeRevisionsCleanup(t, &fakeOnDemand{results: []revisioncleanup.Result{cleanedFile, failedFile}}, revisionsCleanupOptions{})
		require.NoError(t, err)
		require.Equal(t, revisionsCleanupExitFailed, code)
		require.Contains(t, out, "space2 /broken.txt: failed: could not list revisions")
	})

	t.Run("a revision that could not be deleted", func(t *testing.T) {
		file := cleanedFile
		file.Revisions = append([]revisioncleanup.RevisionResult{}, cleanedFile.Revisions...)
		file.Revisions[1].Err = errors.New("permission denied")
		code, out, err := runFakeRevisionsCleanup(t, &fakeOnDemand{results: []revisioncleanup.Result{file}}, revisionsCleanupOptions{})
		require.NoError(t, err)
		require.Equal(t, revisionsCleanupExitFailed, code)
		require.Contains(t, out, "deleted 0 of 2 revisions (0 B), 1 could not be deleted\n  node1.REV.2: permission denied\n")
	})

	t.Run("the cleanup could not run", func(t *testing.T) {
		code, _, err := runFakeRevisionsCleanup(t, &fakeOnDemand{err: errors.New("could not find space space1")}, revisionsCleanupOptions{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "could not find space")
		require.Equal(t, 0, code, "the error itself makes the command fail")
	})

	t.Run("stopped by the timeout", func(t *testing.T) {
		code, out, err := runFakeRevisionsCleanup(t, &fakeOnDemand{results: []revisioncleanup.Result{cleanedFile}, wait: true}, revisionsCleanupOptions{timeout: 10 * time.Millisecond})
		require.NoError(t, err)
		require.Equal(t, revisionsCleanupExitStopped, code)
		require.Contains(t, out, "The cleanup was stopped after the timeout of 10ms, before all files were cleaned up.")
	})

	t.Run("stopped by the timeout, with a failed file", func(t *testing.T) {
		code, _, err := runFakeRevisionsCleanup(t, &fakeOnDemand{results: []revisioncleanup.Result{failedFile}, wait: true}, revisionsCleanupOptions{timeout: 10 * time.Millisecond})
		require.NoError(t, err)
		require.Equal(t, revisionsCleanupExitFailed, code)
	})
}

func TestFormatBytes(t *testing.T) {
	for in, want := range map[uint64]string{0: "0 B", 1023: "1023 B", 1024: "1.0 KiB", 1536: "1.5 KiB", 5 << 20: "5.0 MiB", 3 << 40: "3.0 TiB"} {
		require.Equal(t, want, formatBytes(in))
	}
}

func TestNewEventStream(t *testing.T) {
	cfg := &config.Config{}
	cfg.Events.Addr = "127.0.0.1:9233"

	// blocked stands for a NATS that doesn't answer: newStream returns only when the test ends
	blocked := func(t *testing.T) func(*config.Config) (string, error) {
		release := make(chan struct{})
		t.Cleanup(func() { close(release) })
		return func(*config.Config) (string, error) {
			<-release
			return "", errors.New("released")
		}
	}

	t.Run("returns the stream once connected", func(t *testing.T) {
		stream, err := newEventStream(context.Background(), cfg, time.Second, func(*config.Config) (string, error) {
			return "stream", nil
		})
		require.NoError(t, err)
		require.Equal(t, "stream", stream)
	})

	t.Run("returns the error of the stream", func(t *testing.T) {
		_, err := newEventStream(context.Background(), cfg, time.Second, func(*config.Config) (string, error) {
			return "", errors.New("bad credentials")
		})
		require.EqualError(t, err, "bad credentials")
	})

	t.Run("gives up after the timeout, and says what it needs", func(t *testing.T) {
		start := time.Now()
		_, err := newEventStream(context.Background(), cfg, 50*time.Millisecond, blocked(t))
		require.Error(t, err)
		require.Contains(t, err.Error(), "could not connect to NATS at 127.0.0.1:9233 within 50ms")
		require.Contains(t, err.Error(), "a running OpenCloud server or a separate NATS")
		require.True(t, time.Since(start) < 5*time.Second, "it should give up long before 5s")
	})

	t.Run("stops when interrupted", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		time.AfterFunc(50*time.Millisecond, cancel)
		_, err := newEventStream(ctx, cfg, time.Minute, blocked(t))
		require.True(t, errors.Is(err, context.Canceled), "expected context.Canceled, got %v", err)
		require.Contains(t, err.Error(), "interrupted while connecting to NATS")
	})
}
