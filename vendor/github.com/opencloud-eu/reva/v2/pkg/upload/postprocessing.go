// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

// This file holds the postprocessing result collector: the second half of an
// async upload. finishUpload stages the bytes and publishes BytesReceived; the
// postprocessing service scans them and reports back here, and only then are the
// bytes committed through the driver seam.
//
// It is driver-agnostic on purpose: this logic used to live inside decomposedfs,
// which meant only that driver could offer async uploads.

package upload

import (
	"context"
	"errors"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/datatx/metrics"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// RegisteredEvents are the postprocessing events the coordinator consumes.
var RegisteredEvents = []events.Unmarshaller{
	events.PostprocessingFinished{},
	events.PostprocessingStepFinished{},
	events.RestartPostprocessing{},
	events.StartPostprocessingStep{},
	events.CleanUpload{},
}

// StartPostprocessing subscribes to postprocessing results and switches the
// coordinator over to async uploads: from here on finished uploads stage their
// bytes and wait for a scan verdict instead of committing inline.
//
// The two go together on purpose. Deferring a commit is only safe if something
// will arrive to finish it, so there is no way to enable async without a running
// consumer, and none to run a consumer that never receives work.
//
// Call once, before serving requests. Fails without a publisher: nothing would
// hand uploads to postprocessing, so every one of them would wait for a verdict
// that never comes.
func (c *coordinator) StartPostprocessing(stream events.Consumer, ac AsyncConf) error {
	if c.pub == nil {
		return errors.New("coordinator: async uploads need an event publisher")
	}
	ch, err := events.Consume(stream, ac.ConsumerGroup, RegisteredEvents...)
	if err != nil {
		return err
	}
	numConsumers := ac.NumConsumers
	if numConsumers <= 0 {
		numConsumers = 1
	}
	c.mountID = ac.MountID
	c.commitMaxRetries = ac.CommitMaxRetries
	c.commitRetryBackoff = ac.CommitRetryBackoff
	c.async = true
	for i := 0; i < numConsumers; i++ {
		go c.Postprocessing(ch)
	}
	return nil
}

// Postprocessing consumes postprocessing results until ch is closed. Run it in
// its own goroutine, one per configured consumer.
func (c *coordinator) Postprocessing(ch <-chan events.Event) {
	for event := range ch {
		// derive a per-event context from the event's W3C trace context so the
		// postprocessing work is linked to the originating request
		ctx := event.GetTraceContext(context.Background())
		ctx = ctxpkg.ContextSetInitiator(ctx, event.InitiatorID)
		c.processEvent(ctx, event)
	}
}

// servesStorage reports whether an event is for the storage this coordinator
// serves. Postprocessing runs as a separate service and broadcasts its results to
// every storage provider, so each has to recognise its own. Events that name no
// storage predate this and are accepted.
func (c *coordinator) servesStorage(id *provider.ResourceId) bool {
	if c.mountID == "" || id.GetStorageId() == "" {
		return true
	}
	return id.GetStorageId() == c.mountID
}

func (c *coordinator) processEvent(ctx context.Context, event events.Event) {
	log := appctx.GetLogger(ctx)

	switch ev := event.Event.(type) {
	case events.PostprocessingFinished:
		if !c.servesStorage(ev.ResourceID) {
			return
		}
		c.onPostprocessingFinished(ctx, ev, log)
	case events.RestartPostprocessing:
		c.onRestartPostprocessing(ctx, ev, log)
	case events.StartPostprocessingStep:
		if ev.UploadID == "" {
			return
		}
		session, err := c.store.Get(ctx, ev.UploadID)
		if err != nil {
			// postprocessing broadcasts to every storage, the session may live elsewhere
			log.Debug().Err(err).Str("uploadid", ev.UploadID).Msg("StartPostprocessingStep: could not load session")
			return
		}
		session.SetStatus(SessionStatusProcessing, "started postprocessing step: "+string(ev.StepToStart))
		if err := session.Persist(ctx); err != nil {
			log.Error().Err(err).Str("uploadid", ev.UploadID).Msg("could not persist the session status")
		}
	case events.CleanUpload:
		session, err := c.store.Get(ctx, ev.UploadID)
		if err != nil {
			log.Error().Err(err).Str("uploadid", ev.UploadID).Msg("CleanUpload: could not load session")
			return
		}
		if !ev.KeepUpload {
			c.rollbackPrepared(ctx, session, session.SizeDiff())
		} else {
			c.rollbackNode(ctx, session, "cleaned up, keeping the upload")
			metrics.UploadProcessing.Dec()
		}
	case events.PostprocessingStepFinished:
		if !c.servesStorage(ev.ResourceID) {
			return
		}
		if ev.FinishedStep != events.PPStepAntivirus {
			// only the antivirus result is recorded on the session
			return
		}
		res, ok := ev.Result.(events.VirusscanResult)
		if !ok || res.ErrorMsg != "" {
			// the scan itself failed; PostprocessingFinished decides the outcome
			return
		}
		session, err := c.store.Get(ctx, ev.UploadID)
		if err != nil {
			// an empty upload id means an on-demand scan, which has no session
			if ev.UploadID != "" {
				log.Error().Err(err).Str("uploadid", ev.UploadID).Msg("PostprocessingStepFinished: could not load session")
			}
			return
		}
		session.SetScanData(res.Description, res.Scandate)
		switch ev.Outcome {
		case events.PPOutcomeAbort, events.PPOutcomeDelete:
			session.SetStatus(SessionStatusFailed, "finished postprocessing step: "+string(ev.FinishedStep))
		default:
			session.SetStatus(SessionStatusProcessing, "finished postprocessing step: "+string(ev.FinishedStep))
		}
		metrics.UploadSessionsScanned.Inc()
		if err := session.Persist(ctx); err != nil {
			log.Error().Err(err).Str("uploadid", ev.UploadID).Msg("could not persist scan result")
		}
	}
}

// onPostprocessingFinished completes or discards an upload according to the
// outcome postprocessing reported.
func (c *coordinator) onPostprocessingFinished(ctx context.Context, ev events.PostprocessingFinished, log *zerolog.Logger) {
	session, err := c.store.Get(ctx, ev.UploadID)
	if err != nil {
		// Without the session we cannot reach the staged bytes, so they are leaked
		// here. Housekeeping cleans them up later.
		log.Error().Err(err).Str("uploadid", ev.UploadID).Msg("PostprocessingFinished: could not load session")
		return
	}
	ctx = session.Context(ctx)
	log = appctx.GetLogger(ctx)

	switch ev.Outcome {
	case events.PPOutcomeContinue:
		if err := c.finishAsync(ctx, session); err != nil {
			log.Error().Err(err).Str("uploadid", ev.UploadID).Msg("blob commit failed after all retries, reverting node and keeping bytes")
			metrics.UploadSessionsCommitFailed.Inc()
			// Keep the staged bytes: the blobstore outage may be transient, so an
			// admin can retry with RestartPostprocessing rather than lose the upload.
			c.rollbackNode(ctx, session, "commit failed: "+err.Error())
			metrics.UploadProcessing.Dec()
			c.publishUploadFailed(ctx, session, ev)
		}
		return
	case events.PPOutcomeAbort:
		metrics.UploadSessionsAborted.Inc()
		// Keep the staged bytes: an abort is a transient failure and the upload can
		// be restarted with RestartPostprocessing.
		c.rollbackNode(ctx, session, "postprocessing aborted")
	case events.PPOutcomeDelete:
		metrics.UploadSessionsDeleted.Inc()
		c.rollbackPrepared(ctx, session, session.SizeDiff())
	default:
		log.Error().Str("outcome", string(ev.Outcome)).Str("uploadid", ev.UploadID).Msg("unknown postprocessing outcome, aborting")
		metrics.UploadSessionsAborted.Inc()
		c.rollbackNode(ctx, session, "unknown postprocessing outcome: "+string(ev.Outcome))
	}

	c.publishUploadFailed(ctx, session, ev)
}

// onRestartPostprocessing re-publishes BytesReceived so a previously aborted
// upload gets another postprocessing run.
func (c *coordinator) onRestartPostprocessing(ctx context.Context, ev events.RestartPostprocessing, log *zerolog.Logger) {
	session, err := c.store.Get(ctx, ev.UploadID)
	if err != nil {
		log.Error().Err(err).Str("uploadid", ev.UploadID).Msg("RestartPostprocessing: could not load session")
		return
	}
	metrics.UploadSessionsRestarted.Inc()
	if err := c.publishBytesReceived(session.Context(ctx), session); err != nil {
		log.Error().Err(err).Str("uploadid", ev.UploadID).Msg("could not restart postprocessing")
	}
}

// rollbackNode reverts the node to its pre-upload state but keeps the staged
// bytes and the session, so postprocessing can be restarted.
func (c *coordinator) rollbackNode(ctx context.Context, session Session, reason string) {
	ref := session.Reference()
	if err := c.fs.RollbackUpload(ctx, &ref, session.ID(), rollbackInfo(session, session.SizeDiff())); err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Str("uploadid", session.ID()).Msg("could not roll back upload, keeping session")
		return
	}
	c.unmarkProcessing(ctx, session, &ref)

	// the size was given back, so a later cleanup of the kept session must not
	// release it again
	session.SetSizeDiff(0)
	session.SetStatus(SessionStatusFailed, reason)
	if err := session.Persist(ctx); err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Str("uploadid", session.ID()).Msg("could not persist the rolled back session")
	}
}

// publishUploadFailed tells consumers the upload will not become available.
// Clients wait on UploadReady, so staying silent would leave them hanging.
func (c *coordinator) publishUploadFailed(ctx context.Context, session Session, ev events.PostprocessingFinished) {
	if c.pub == nil {
		return
	}
	if err := events.Publish(ctx, c.pub, events.UploadReady{
		UploadID:      session.ID(),
		Failed:        true,
		Filename:      session.Filename(),
		SpaceOwner:    session.SpaceOwner(),
		ExecutingUser: ev.ExecutingUser,
		FileRef:       c.uploadRef(session),
		ResourceID: &provider.ResourceId{
			StorageId: session.ProviderID(),
			SpaceId:   session.SpaceID(),
			OpaqueId:  session.NodeID(),
		},
		Timestamp:         utils.TSNow(),
		IsVersion:         session.VersionCreated(),
		ImpersonatingUser: ev.ImpersonatingUser,
	}); err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Str("uploadid", session.ID()).Msg("failed to publish UploadReady event")
	}
}
