package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/event"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/revaconfig"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/registry"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/revisioncleanup"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
)

// The exit codes of the revisions cleanup command
const (
	revisionsCleanupExitFailed  = 1 // the cleanup or some of its files failed
	revisionsCleanupExitStopped = 2 // stopped by --timeout before all files were cleaned up
)

// Revisions wraps the commands for the revisions (versions) of files.
func Revisions(cfg *config.Config) *cobra.Command {
	revisionsCmd := &cobra.Command{
		Use:   "revisions",
		Short: "manage the revisions (versions) of files",
	}
	revisionsCmd.AddCommand(RevisionsCleanup(cfg))
	return revisionsCmd
}

// RevisionsCleanup deletes old revisions of the files of whole spaces, folders or files.
func RevisionsCleanup(cfg *config.Config) *cobra.Command {
	cmd, _ := revisionsCleanupCommand(cfg)
	return cmd
}

// revisionsCleanupCommand returns the revisions cleanup command, and the values of its flags.
func revisionsCleanupCommand(cfg *config.Config) (*cobra.Command, *revisionsCleanupFlags) {
	f := &revisionsCleanupFlags{}
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Delete old revisions according to a cleanup strategy",
		Long: `Delete old revisions (versions) of files according to a cleanup strategy.

The storage-users service cleans up the revisions of a file whenever it gets a new one,
with the strategy configured in OC_REVISIONS_CLEANUP_STRATEGY. This command cleans up
the revisions of files that are not changed anymore, or applies a different strategy
once, e.g. after the strategy was changed or to free storage.

Exactly one of these selects the files to clean up:
  --space <id>      all files of a space, repeatable; with --path, only that file or
                    folder within the space
  --all             all files of all spaces
  --fs-path <path>  a file or folder by its path on disk ('posix' driver only)

The strategy and the settings of the background workers default to the configured
ones (OC_REVISIONS_CLEANUP_*) and can be overridden for this run. See the description
of OC_REVISIONS_CLEANUP_STRATEGY for the strategies, e.g. '30d', '10' or '7d, auto, 90d'.
Only --queue-full defaults to 'block' without a timeout, whatever is configured: the
command finds files faster than the workers clean them up, so dropping the files that
find the queue full would skip most of them.

The command uses the storage directly, with the configuration of the storage-users
service, and can run while the service is running. It needs access to the storage and,
for the 'posix' driver, to NATS.

By default, it prints a line for every file it deleted revisions of, could not clean
up or skipped, and a summary. --verbose prints every file with every revision, what
was decided about it and why. --json prints the same as JSON objects, one per line.

--dry-run deletes nothing and prints what would be deleted. It doesn't change any
file, but it still takes the locks of the files, which can create lock files.

--timeout stops the cleanup gracefully: the files being cleaned up are finished, the
files still waiting are skipped. It can't interrupt a file that waits for the lock of
another process, though, so the command can run longer than the timeout. As every run
starts over from the beginning, combine --timeout with --shuffle for regular runs, e.g.
by a cron job, so that every run starts in a different place.

Exit codes: 0 when done, 1 if the cleanup or any file failed, 2 if --timeout stopped
the cleanup before all files were cleaned up (and nothing failed).`,
		Example: `  opencloud storage-users revisions cleanup --all --strategy "auto, 90d" --dry-run
  opencloud storage-users revisions cleanup --space <space id> --path /Documents --strategy 10
  opencloud storage-users revisions cleanup --all --timeout 1h --shuffle`,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			code, err := runRevisionsCleanup(cmd, cfg, f)
			if err != nil {
				return err
			}
			if code != 0 {
				os.Exit(code)
			}
			return nil
		},
	}
	flags := cmd.Flags()
	flags.StringSliceVar(&f.spaces, "space", nil, "the id of a space to clean up, repeatable")
	flags.StringVar(&f.o.request.Path, "path", "", "a file or folder within the space given by --space, to clean up only that")
	flags.StringVarP(&f.o.request.FSPath, "fs-path", "p", "", "a file or folder to clean up, by its path on disk ('posix' driver only)")
	flags.BoolVarP(&f.o.request.All, "all", "a", false, "clean up all spaces")
	flags.StringVarP(&f.strategy, "strategy", "s", "", "the cleanup strategy, defaults to OC_REVISIONS_CLEANUP_STRATEGY")
	flags.BoolVarP(&f.o.request.DryRun, "dry-run", "n", false, "delete nothing, only print what would be deleted")
	flags.BoolVarP(&f.o.verbose, "verbose", "v", false, "print every file and every revision, with the decision and its reason")
	flags.BoolVarP(&f.o.json, "json", "j", false, "print JSON objects, one per line, instead of text")
	flags.DurationVarP(&f.o.timeout, "timeout", "t", 0, "stop the cleanup gracefully after this time, e.g. 30m or 2h; 0 never stops it")
	flags.BoolVar(&f.o.request.Shuffle, "shuffle", false, "clean up the spaces and the files within each folder in a random order")
	flags.BoolVar(&f.async, "async", false, "clean up files in background workers, while looking for more, defaults to OC_REVISIONS_CLEANUP_ASYNC")
	flags.IntVar(&f.workers, "workers", 0, "the number of background workers, 0 for automatic, defaults to OC_REVISIONS_CLEANUP_ASYNC_WORKERS")
	flags.IntVar(&f.queueSize, "queue-size", 0, "the number of files that can wait for a worker, 0 for automatic, defaults to OC_REVISIONS_CLEANUP_ASYNC_QUEUE_SIZE")
	flags.StringVar(&f.queueFull, "queue-full", string(revisioncleanup.QueueFullBlock), "'block' (wait for room) or 'drop' (skip) files when the queue is full")
	flags.DurationVar(&f.queueFullTimeout, "queue-full-timeout", 0, "with --queue-full block, skip a file after waiting this long for room, e.g. 10s; 0 waits as long as it takes")
	return cmd, f
}

// revisionsCleanupFlags holds the values of the flags of the revisions cleanup command: in o
// the ones that are used as they are, the others are turned into options by options.
type revisionsCleanupFlags struct {
	o         revisionsCleanupOptions
	spaces    []string
	strategy  string
	async     bool
	workers   int
	queueSize int
	queueFull string
	// the settings of a full queue, which, unlike the others, don't default to the configured
	// ones, see the help of the command
	queueFullTimeout time.Duration
}

// revisionsCleanupOptions are the settings of a run of the revisions cleanup command
type revisionsCleanupOptions struct {
	request revisioncleanup.Request
	timeout time.Duration
	verbose bool
	json    bool
}

// options builds the options of a run from the flags, with the configured strategy and async
// settings unless the flags override them; changed tells whether a flag was given.
func (f *revisionsCleanupFlags) options(cfg *config.Config, changed func(name string) bool) (revisionsCleanupOptions, error) {
	o := f.o
	r := &o.request

	for _, space := range f.spaces {
		// accept the ids as the APIs show them, e.g. <storage id>$<space id>
		id, err := storagespace.ParseID(space)
		if err != nil {
			return o, fmt.Errorf("invalid space id %q: %w", space, err)
		}
		r.SpaceIDs = append(r.SpaceIDs, id.GetSpaceId())
	}
	if r.Path != "" && len(r.SpaceIDs) == 0 {
		return o, errors.New("--path requires --space")
	}
	err := r.Validate()
	if err != nil {
		return o, err
	}
	if r.FSPath != "" && cfg.Driver != "posix" {
		return o, fmt.Errorf("--fs-path is only supported by the 'posix' driver, not by %q", cfg.Driver)
	}

	r.Strategy = cfg.RevisionsCleanup.Strategy
	if changed("strategy") {
		r.Strategy = f.strategy
	}
	// check the strategy before the storage is opened
	switch cleaner, err := revisioncleanup.New(r.Strategy); {
	case err != nil:
		return o, fmt.Errorf("invalid strategy %q: %w", r.Strategy, err)
	case !cleaner.Enabled():
		return o, errors.New("no strategy: the revisions cleanup is disabled, give one with --strategy or configure OC_REVISIONS_CLEANUP_STRATEGY")
	}

	async := revisioncleanup.AsyncOptions{
		Enabled:   cfg.RevisionsCleanup.Async,
		Workers:   cfg.RevisionsCleanup.AsyncWorkers,
		QueueSize: cfg.RevisionsCleanup.AsyncQueueSize,
		// not the configured ones, see revisionsCleanupFlags.queueFullTimeout
		QueueFull:        revisioncleanup.QueueFullPolicy(f.queueFull),
		QueueFullTimeout: f.queueFullTimeout,
	}
	if changed("async") {
		async.Enabled = f.async
	}
	if changed("workers") {
		async.Workers = f.workers
	}
	if changed("queue-size") {
		async.QueueSize = f.queueSize
	}
	r.Async = &async

	if o.timeout < 0 {
		return o, fmt.Errorf("invalid timeout %s", o.timeout)
	}
	return o, nil
}

// runRevisionsCleanup runs the revisions cleanup command and returns its exit code.
func runRevisionsCleanup(cmd *cobra.Command, cfg *config.Config, f *revisionsCleanupFlags) (int, error) {
	o, err := f.options(cfg, cmd.Flags().Changed)
	if err != nil {
		return 0, err
	}

	onDemand, err := revisionsCleanupDriver(cfg)
	if err != nil {
		return 0, err
	}

	// stop gracefully on ctrl-c as well
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// warnings of the cleanup, e.g. about revisions dated in the future; errors are reported
	logger := zerolog.New(zerolog.ConsoleWriter{Out: cmd.ErrOrStderr(), TimeFormat: time.RFC3339, NoColor: true}).
		With().Timestamp().Logger().Level(zerolog.WarnLevel)
	ctx = logger.WithContext(ctx)

	return cleanupRevisions(ctx, onDemand, o, cmd.OutOrStdout())
}

// cleanupRevisions runs the cleanup with the given options, prints its results to w and
// returns the exit code of the command.
func cleanupRevisions(ctx context.Context, onDemand revisioncleanup.OnDemand, o revisionsCleanupOptions, w io.Writer) (int, error) {
	if o.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.timeout)
		defer cancel()
	}

	out := newRevisionsCleanupOutput(w, o)
	err := onDemand.CleanupRevisionsUnchecked(ctx, o.request, out.report)
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		out.summary(fmt.Sprintf("stopped after the timeout of %s", o.timeout))
		if out.failed() {
			return revisionsCleanupExitFailed, nil
		}
		return revisionsCleanupExitStopped, nil
	case errors.Is(err, context.Canceled):
		out.summary("interrupted")
		return revisionsCleanupExitFailed, nil
	case err != nil:
		out.summary("")
		return 0, err
	}
	out.summary("")
	if out.err != nil {
		return 0, fmt.Errorf("could not write the output: %w", out.err)
	}
	if out.failed() {
		return revisionsCleanupExitFailed, nil
	}
	return 0, nil
}

// revisionsCleanupDriver creates the storage driver, without the parts of the
// storage-users service that would interfere with the running service.
func revisionsCleanupDriver(cfg *config.Config) (revisioncleanup.OnDemand, error) {
	newFS, ok := registry.NewFuncs[cfg.Driver]
	if !ok {
		return nil, fmt.Errorf("unknown storage driver %q", cfg.Driver)
	}
	drivers := revaconfig.StorageProviderDrivers(cfg)
	var stream events.Stream
	if cfg.Driver == "posix" {
		// neither scan nor watch the file system, that is the service's job: e.g. a second
		// inotifywait would stop the one of the service
		drivers["posix"] = revaconfig.Posix(cfg, false, false)
		// posix refuses to start without an event stream
		var err error
		stream, err = event.NewStream(cfg)
		if err != nil {
			return nil, fmt.Errorf("could not create the event stream for the posix driver: %w", err)
		}
	}
	driverConfig, ok := drivers[cfg.Driver].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unknown storage driver %q", cfg.Driver)
	}
	// Without async uploads, the driver doesn't join the postprocessing consumer group of the
	// service: the events delivered to this short-lived process would be lost.
	driverConfig["asyncfileuploads"] = false

	fs, err := newFS(driverConfig, stream, nil)
	if err != nil {
		return nil, fmt.Errorf("could not initialize the storage driver %q: %w", cfg.Driver, err)
	}
	onDemand, ok := fs.(revisioncleanup.OnDemand)
	if !ok {
		return nil, fmt.Errorf("the %q driver can't clean up revisions", cfg.Driver)
	}
	return onDemand, nil
}

// revisionsCleanupOutput prints the results of a revisions cleanup as they come in, and
// sums them up.
type revisionsCleanupOutput struct {
	w       io.Writer
	o       revisionsCleanupOptions
	enc     *json.Encoder
	spaces  map[string]struct{}
	files   int
	cleaned int // files with deleted revisions
	deleted int
	bytes   uint64
	skipped int
	errors  int   // files that failed, as a whole or for some of their revisions
	err     error // the first error writing the output
}

func newRevisionsCleanupOutput(w io.Writer, o revisionsCleanupOptions) *revisionsCleanupOutput {
	out := &revisionsCleanupOutput{w: w, o: o, spaces: map[string]struct{}{}}
	if o.json {
		out.enc = json.NewEncoder(w)
	}
	return out
}

// revisionJSON is a revision in the JSON output
type revisionJSON struct {
	Key      string    `json:"key"`
	Time     time.Time `json:"time"`
	Size     uint64    `json:"size"`
	Decision string    `json:"decision"`
	Reason   string    `json:"reason,omitempty"`
	Error    string    `json:"error,omitempty"`
}

// fileJSON is a file in the JSON output
type fileJSON struct {
	Type         string         `json:"type"`
	SpaceID      string         `json:"space_id"`
	NodeID       string         `json:"node_id"`
	Path         string         `json:"path,omitempty"`
	DryRun       bool           `json:"dry_run"`
	Skipped      string         `json:"skipped,omitempty"`
	Error        string         `json:"error,omitempty"`
	Revisions    int            `json:"revisions"`
	Deleted      int            `json:"deleted"`
	DeletedBytes uint64         `json:"deleted_bytes"`
	Failed       int            `json:"failed"`
	Details      []revisionJSON `json:"details,omitempty"`
}

// summaryJSON is the summary in the JSON output
type summaryJSON struct {
	Type         string `json:"type"`
	DryRun       bool   `json:"dry_run"`
	Spaces       int    `json:"spaces"`
	Files        int    `json:"files"`
	Cleaned      int    `json:"cleaned"`
	Deleted      int    `json:"deleted"`
	DeletedBytes uint64 `json:"deleted_bytes"`
	Skipped      int    `json:"skipped"`
	Failed       int    `json:"failed"`
	Stopped      string `json:"stopped,omitempty"`
}

// report prints the Result of a file, it is called one result at a time.
func (out *revisionsCleanupOutput) report(r revisioncleanup.Result) {
	deleted, bytes := r.DeletedCount()
	failedRevisions := 0
	for _, rev := range r.Revisions {
		if rev.Err != nil {
			failedRevisions++
		}
	}
	failed := r.Err != nil || failedRevisions > 0

	out.spaces[r.SpaceID] = struct{}{}
	out.files++
	out.deleted += deleted
	out.bytes += bytes
	switch {
	case failed:
		out.errors++
	case r.Skipped != "":
		out.skipped++
	}
	if deleted > 0 {
		out.cleaned++
	}

	// without --verbose, only what needs attention or changed something
	if !out.o.verbose && deleted == 0 && !failed && r.Skipped == "" {
		return
	}
	if out.enc != nil {
		out.printJSON(r, deleted, bytes, failedRevisions)
		return
	}
	out.printText(r, deleted, bytes, failedRevisions)
}

func (out *revisionsCleanupOutput) printJSON(r revisioncleanup.Result, deleted int, bytes uint64, failedRevisions int) {
	f := fileJSON{
		Type:         "file",
		SpaceID:      r.SpaceID,
		NodeID:       r.NodeID,
		Path:         r.Path,
		DryRun:       r.DryRun,
		Skipped:      r.Skipped,
		Revisions:    len(r.Revisions),
		Deleted:      deleted,
		DeletedBytes: bytes,
		Failed:       failedRevisions,
	}
	if r.Err != nil {
		f.Error = r.Err.Error()
	}
	if out.o.verbose {
		for _, rev := range r.Revisions {
			d := revisionJSON{Key: rev.Key, Time: rev.Time, Size: rev.Size, Decision: string(rev.Decision), Reason: rev.Reason}
			if rev.Err != nil {
				d.Error = rev.Err.Error()
			}
			f.Details = append(f.Details, d)
		}
	}
	out.write(out.enc.Encode(f))
}

func (out *revisionsCleanupOutput) printText(r revisioncleanup.Result, deleted int, bytes uint64, failedRevisions int) {
	file := r.Path
	if file == "" || out.o.verbose {
		file = strings.TrimSpace(file + " (node " + r.NodeID + ")")
	}
	line := fmt.Sprintf("%s %s: ", r.SpaceID, file)
	switch {
	case r.Err != nil:
		line += "failed: " + r.Err.Error()
	case r.Skipped != "":
		line += "skipped: " + r.Skipped
	default:
		verb := "deleted"
		if r.DryRun {
			verb = "would delete"
		}
		line += fmt.Sprintf("%s %d of %s (%s)", verb, deleted, plural(len(r.Revisions), "revision"), formatBytes(bytes))
		if failedRevisions > 0 {
			line += fmt.Sprintf(", %d could not be deleted", failedRevisions)
		}
	}
	_, err := fmt.Fprintln(out.w, line)
	out.write(err)

	if !out.o.verbose {
		for _, rev := range r.Revisions {
			if rev.Err != nil {
				fmt.Fprintf(out.w, "  %s: %s\n", rev.Key, rev.Err)
			}
		}
		return
	}
	for _, rev := range r.Revisions {
		// in the server's time zone, like the times in the reasons
		fmt.Fprintf(out.w, "  %-9s %s %9s  %s\n", rev.Decision, rev.Time.Local().Format(time.RFC3339), formatBytes(rev.Size), rev.Reason)
		if rev.Err != nil {
			fmt.Fprintf(out.w, "  %-9s could not be deleted: %s\n", "", rev.Err)
		}
	}
}

// summary prints the totals, stopped tells why the cleanup stopped early, if it did.
func (out *revisionsCleanupOutput) summary(stopped string) {
	if out.enc != nil {
		out.write(out.enc.Encode(summaryJSON{
			Type:         "summary",
			DryRun:       out.o.request.DryRun,
			Spaces:       len(out.spaces),
			Files:        out.files,
			Cleaned:      out.cleaned,
			Deleted:      out.deleted,
			DeletedBytes: out.bytes,
			Skipped:      out.skipped,
			Failed:       out.errors,
			Stopped:      stopped,
		}))
		return
	}
	verb := "deleted"
	if out.o.request.DryRun {
		verb = "would delete"
	}
	fmt.Fprintf(out.w, "%s in %s: %s %s (%s) of %s, %d skipped, %d failed\n",
		plural(out.files, "file"), plural(len(out.spaces), "space"), verb, plural(out.deleted, "revision"),
		formatBytes(out.bytes), plural(out.cleaned, "file"), out.skipped, out.errors)
	if stopped != "" {
		fmt.Fprintf(out.w, "The cleanup was %s, before all files were cleaned up.\n", stopped)
	}
}

// write records the first error writing the output.
func (out *revisionsCleanupOutput) write(err error) {
	if out.err == nil {
		out.err = err
	}
}

// failed tells whether the cleanup of any file failed.
func (out *revisionsCleanupOutput) failed() bool {
	return out.errors > 0
}

// plural returns the count with the noun, e.g. "1 file" or "2 files".
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// formatBytes formats a size in bytes with a binary unit, e.g. 1.5 MiB.
func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
