package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/bytesize"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/config"
	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/queue"
	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/scanners"
)

// Scanner is an abstraction for the actual virus scan
type Scanner interface {
	Scan(body scanners.Input) (scanners.Result, error)
}

// NewAntivirus returns a service implementation for Service.
func NewAntivirus(cfg *config.Config, logger log.Logger, tracerProvider trace.TracerProvider) (Antivirus, error) {
	var scanner Scanner
	var err error
	switch cfg.Scanner.Type {
	default:
		return Antivirus{}, fmt.Errorf("unknown av scanner: '%s'", cfg.Scanner.Type)
	case config.ScannerTypeClamAV:
		scanner, err = scanners.NewClamAV(cfg.Scanner.ClamAV.Socket, cfg.Scanner.ClamAV.Timeout)
	case config.ScannerTypeICap:
		scanner, err = scanners.NewICAP(cfg.Scanner.ICAP.URL, cfg.Scanner.ICAP.Service, cfg.Scanner.ICAP.Timeout)
	}
	if err != nil {
		return Antivirus{}, err
	}

	av := Antivirus{
		config:         cfg,
		log:            logger,
		tracerProvider: tracerProvider,
		scanner:        scanner,
		client:         rhttp.GetHTTPClient(rhttp.Insecure(true)),
		stopCh:         make(chan struct{}, 1),
		stopped:        new(atomic.Bool),
	}

	switch mode := cfg.MaxScanSizeMode; mode {
	case config.MaxScanSizeModeSkip, config.MaxScanSizeModePartial:
		break
	default:
		return av, fmt.Errorf("unknown max scan size mode '%s'", cfg.MaxScanSizeMode)
	}

	switch outcome := events.PostprocessingOutcome(cfg.InfectedFileHandling); outcome {
	case events.PPOutcomeContinue, events.PPOutcomeAbort, events.PPOutcomeDelete:
		av.outcome = outcome
	default:
		return av, fmt.Errorf("unknown infected file handling '%s'", outcome)
	}

	if cfg.MaxScanSize != "" {
		b, err := bytesize.Parse(cfg.MaxScanSize)
		if err != nil {
			return av, err
		}

		av.maxScanSize = b.Bytes()
	}

	return av, nil
}

// Antivirus defines implements the business logic for Service.
type Antivirus struct {
	config         *config.Config
	log            log.Logger
	scanner        Scanner
	outcome        events.PostprocessingOutcome
	maxScanSize    uint64
	tracerProvider trace.TracerProvider

	client  *http.Client
	stopCh  chan struct{}
	stopped *atomic.Bool
}

// Run runs the service
func (av Antivirus) Run() error {
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-av.stopCh:
			cancel()
		case <-runCtx.Done():
		}
	}()

	priorityQueue, err := queue.Open(runCtx, av.config, func(err error) {
		av.log.Error().Err(err).Msg("antivirus priority queue error")
	})
	if err != nil {
		return err
	}
	defer priorityQueue.Close()

	wg := sync.WaitGroup{}
	for worker := range av.config.Workers {
		highPriorityOnly := worker < av.config.HighPriorityReservedWorkers
		wg.Go(func() {
			for {
				var lease *queue.Lease
				var err error
				if highPriorityOnly {
					lease, err = priorityQueue.NextHighPriority(runCtx)
				} else {
					lease, err = priorityQueue.Next(runCtx)
				}
				if err != nil {
					if runCtx.Err() != nil {
						return
					}
					av.log.Error().Err(err).Msg("failed to claim antivirus job")
					timer := time.NewTimer(time.Second)
					select {
					case <-runCtx.Done():
						timer.Stop()
						return
					case <-timer.C:
					}
					continue
				}

				if err := av.processLease(runCtx, priorityQueue, lease); err != nil {
					av.log.Error().Err(err).Str("jobID", lease.Job.ID).Msg("antivirus job failed; it will be retried")
					if nakErr := lease.NakWithDelay(time.Second); nakErr != nil {
						av.log.Error().Err(nakErr).Str("jobID", lease.Job.ID).Msg("failed to release antivirus job for retry")
					}
					continue
				}
				if err := lease.Ack(); err != nil {
					av.log.Error().Err(err).Str("jobID", lease.Job.ID).Msg("failed to acknowledge antivirus job")
				}
			}
		})
	}

	wg.Wait()

	return nil
}

func (av Antivirus) Close() {
	if av.stopped.CompareAndSwap(false, true) {
		close(av.stopCh)
	}
}

func (av Antivirus) processLease(ctx context.Context, q *queue.Queue, lease *queue.Lease) error {
	job := lease.Job
	e := events.Event{TraceParent: job.TraceParent}
	ctx = e.GetTraceContext(ctx)
	ctx = ctxpkg.ContextSetInitiator(ctx, job.InitiatorID)
	ctx, span := av.tracerProvider.Tracer("antivirus").Start(ctx, "processEvent")
	defer span.End()
	av.log.Info().Str("traceID", span.SpanContext().TraceID().String()).Msg("TraceID")

	// Keep the JetStream lease alive for scans that take longer than AckWait.
	heartbeatDone := make(chan struct{})
	heartbeatStopped := make(chan struct{})
	go func() {
		defer close(heartbeatStopped)
		interval := q.AckWait() / 3
		if interval <= 0 || interval > 10*time.Second {
			interval = 10 * time.Second
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatDone:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := lease.InProgress(); err != nil {
					av.log.Warn().Err(err).Str("jobID", job.ID).Msg("failed to extend antivirus job lease")
				}
			}
		}
	}()
	defer func() {
		close(heartbeatDone)
		<-heartbeatStopped
	}()

	ev := job.Event
	av.log.Debug().Str("uploadid", ev.UploadID).Str("priority", string(job.Priority)).Dur("queue_wait", time.Since(job.EnqueuedAt)).Msg("Starting virus scan")

	if av.config.DebugScanOutcome != "" {
		av.log.Warn().Str("antivir, clamav", ">>>>>>> ANTIVIRUS_DEBUG_SCAN_OUTCOME IS SET NO ACTUAL VIRUS SCAN IS PERFORMED!").Send()
		if err := q.PublishFinished(ctx, job, events.PostprocessingStepFinished{
			FinishedStep:  events.PPStepAntivirus,
			Outcome:       events.PostprocessingOutcome(av.config.DebugScanOutcome),
			UploadID:      ev.UploadID,
			ExecutingUser: ev.ExecutingUser,
			Filename:      ev.Filename,
			Result: events.VirusscanResult{
				Infected:    true,
				Description: "DEBUG: forced outcome",
				Scandate:    time.Now(),
				ResourceID:  ev.ResourceID,
			},
		}); err != nil {
			av.log.Error().Err(err).Str("uploadid", ev.UploadID).Interface("resourceID", ev.ResourceID).Msg("cannot publish antivirus result")
			return fmt.Errorf("cannot publish antivirus result: %w", err)
		}
		return nil
	}

	var errmsg string
	start := time.Now()
	res, err := av.process(ev)
	if err != nil {
		errmsg = err.Error()
	}
	duration := time.Since(start)

	var outcome events.PostprocessingOutcome
	switch {
	case res.Infected:
		outcome = av.outcome
	case !res.Infected && err == nil:
		outcome = events.PPOutcomeContinue
	case err != nil:
		outcome = events.PPOutcomeRetry
	default:
		// Not sure what this is about. Abort.
		outcome = events.PPOutcomeAbort
	}

	av.log.Info().Str("uploadid", ev.UploadID).Interface("resourceID", ev.ResourceID).Str("virus", res.Description).Str("outcome", string(outcome)).Str("filename", ev.Filename).Str("user", ev.ExecutingUser.GetId().GetOpaqueId()).Bool("infected", res.Infected).Dur("duration", duration).Msg("File scanned")
	if err := q.PublishFinished(ctx, job, events.PostprocessingStepFinished{
		FinishedStep:  events.PPStepAntivirus,
		Outcome:       outcome,
		UploadID:      ev.UploadID,
		ExecutingUser: ev.ExecutingUser,
		Filename:      ev.Filename,
		Result: events.VirusscanResult{
			Infected:    res.Infected,
			Description: res.Description,
			Scandate:    time.Now(),
			ResourceID:  ev.ResourceID,
			ErrorMsg:    errmsg,
		},
	}); err != nil {
		av.log.Error().Err(err).Str("uploadid", ev.UploadID).Interface("resourceID", ev.ResourceID).Msg("cannot publish antivirus result")
		return fmt.Errorf("cannot publish antivirus result: %w", err)
	}
	return nil
}

// process the scan
func (av Antivirus) process(ev events.StartPostprocessingStep) (scanners.Result, error) {
	if ev.Filesize == 0 {
		av.log.Info().Str("uploadid", ev.UploadID).Msg("Skipping file to be virus scanned, file size is 0.")
		return scanners.Result{ScanTime: time.Now()}, nil
	}

	filesize := ev.Filesize
	headers := make(map[string]string)
	switch {
	case av.maxScanSize == 0:
		// there is no size limit
		break
	case av.config.MaxScanSizeMode == config.MaxScanSizeModeSkip && filesize > av.maxScanSize:
		// skip the file if it is bigger than the max scan size
		av.log.Info().Str("uploadid", ev.UploadID).Uint64("filesize", filesize).
			Msg("Skipping file to be virus scanned, file size is bigger than max scan size.")
		return scanners.Result{ScanTime: time.Now()}, nil
	case av.config.MaxScanSizeMode == config.MaxScanSizeModePartial && filesize > av.maxScanSize:
		// set the range header to only download the first maxScanSize bytes
		headers["Range"] = fmt.Sprintf("bytes=0-%d", av.maxScanSize-1)
		filesize = av.maxScanSize // inform the scanner that we are only scanning part of the file
	}

	var err error
	var rrc io.ReadCloser

	switch ev.UploadID {
	default:
		rrc, err = av.downloadViaToken(ev.URL, headers)
	case "":
		rrc, err = av.downloadViaReva(ev.URL, ev.Token, ev.RevaToken, headers)
	}
	if err != nil {
		av.log.Error().Err(err).Str("uploadid", ev.UploadID).Msg("error downloading file")
		return scanners.Result{}, err
	}
	defer func() {
		_ = rrc.Close()
	}()

	av.log.Debug().Str("uploadid", ev.UploadID).Msg("Downloaded file successfully, starting virusscan")

	res, err := av.scanner.Scan(scanners.Input{Body: rrc, Size: int64(filesize), Url: ev.URL, Name: ev.Filename})
	if err != nil {
		av.log.Error().Err(err).Str("uploadid", ev.UploadID).Msg("error scanning file")
	}

	return res, err
}

// download will download the file
func (av Antivirus) downloadViaToken(url string, headers map[string]string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return av.doDownload(req, headers)
}

// download will download the file
func (av Antivirus) downloadViaReva(url string, dltoken string, revatoken string, headers map[string]string) (io.ReadCloser, error) {
	req, err := rhttp.NewRequest(ctxpkg.ContextSetToken(context.Background(), revatoken), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Reva-Transfer", dltoken)

	return av.doDownload(req, headers)
}

func (av Antivirus) doDownload(req *http.Request, headers map[string]string) (io.ReadCloser, error) {
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	res, err := av.client.Do(req)
	if err != nil {
		return nil, err
	}

	if !slices.Contains([]int{http.StatusOK, http.StatusPartialContent}, res.StatusCode) {
		_ = res.Body.Close()
		return nil, fmt.Errorf("unexpected status code from Download %v", res.StatusCode)
	}

	return res.Body, nil
}
