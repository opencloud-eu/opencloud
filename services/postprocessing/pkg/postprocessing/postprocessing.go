package postprocessing

import (
	"math"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/config"
	"github.com/opencloud-eu/reva/v2/pkg/events"
)

// Postprocessing handles postprocessing of a file
type Postprocessing struct {
	ID                string
	URL               string
	User              *user.User
	ImpersonatingUser *user.User
	Filename          string
	Filesize          uint64
	ResourceID        *provider.ResourceId
	Steps             []events.Postprocessingstep
	Status            Status
	Failures          int
	InitiatorID       string
	Finished          bool
	StartTime         time.Time

	config config.Postprocessing
}

// Status is helper struct to show current postprocessing status
type Status struct {
	CurrentStep events.Postprocessingstep
	Outcome     events.PostprocessingOutcome
}

// New returns a new postprocessing instance
func New(config config.Postprocessing) *Postprocessing {
	return &Postprocessing{
		config: config,
	}
}

// Init is the first step of the postprocessing
func (pp *Postprocessing) Init(_ events.BytesReceived) interface{} {
	if len(pp.Steps) == 0 {
		return pp.finished(events.PPOutcomeContinue)
	}

	return pp.step(pp.Steps[0])
}

// NextStep returns the next postprocessing step
func (pp *Postprocessing) NextStep(ev events.PostprocessingStepFinished) interface{} {
	switch ev.Outcome {
	case events.PPOutcomeContinue:
		return pp.next(ev.FinishedStep)
	case events.PPOutcomeRetry:
		pp.Failures++
		if pp.Failures > pp.config.MaxRetries {
			return pp.finished(events.PPOutcomeAbort)
		}
		return pp.retry()
	default:
		return pp.finished(ev.Outcome)
	}
}

// CurrentStep returns the current postprocessing step
func (pp *Postprocessing) CurrentStep() interface{} {
	if pp.Status.CurrentStep == events.PPStepFinished {
		return pp.finished(pp.Status.Outcome)
	}
	return pp.step(pp.Status.CurrentStep)
}

// Delay will sleep the configured time then continue
func (pp *Postprocessing) Delay(f func(next interface{})) {
	next := pp.next(events.PPStepDelay)
	go func() {
		time.Sleep(pp.config.Delayprocessing)
		f(next)
	}()
}

// BackoffDuration calculates the duration for exponential backoff based on the number of failures.
func (pp *Postprocessing) BackoffDuration() time.Duration {
	return pp.config.RetryBackoffDuration * time.Duration(math.Pow(2, float64(pp.Failures-1)))
}

func (pp *Postprocessing) next(current events.Postprocessingstep) interface{} {
	l := len(pp.Steps)
	for i, s := range pp.Steps {
		if s == current && i+1 < l {
			return pp.step(pp.Steps[i+1])
		}
	}
	return pp.finished(events.PPOutcomeContinue)
}

func (pp *Postprocessing) step(next events.Postprocessingstep) events.StartPostprocessingStep {
	pp.Status.CurrentStep = next
	return events.StartPostprocessingStep{
		UploadID:          pp.ID,
		URL:               pp.URL,
		ExecutingUser:     pp.User,
		Filename:          pp.Filename,
		Filesize:          pp.Filesize,
		ResourceID:        pp.ResourceID,
		StepToStart:       next,
		ImpersonatingUser: pp.ImpersonatingUser,
	}
}

func (pp *Postprocessing) finished(outcome events.PostprocessingOutcome) events.PostprocessingFinished {
	pp.Status.CurrentStep = events.PPStepFinished
	pp.Status.Outcome = outcome
	return events.PostprocessingFinished{
		UploadID:          pp.ID,
		ExecutingUser:     pp.User,
		Filename:          pp.Filename,
		Outcome:           outcome,
		ImpersonatingUser: pp.ImpersonatingUser,
	}
}

func (pp *Postprocessing) retry() events.PostprocessingRetry {
	pp.Status.Outcome = events.PPOutcomeRetry
	return events.PostprocessingRetry{
		UploadID:        pp.ID,
		ExecutingUser:   pp.User,
		Filename:        pp.Filename,
		Failures:        pp.Failures,
		BackoffDuration: pp.BackoffDuration(),
	}
}
