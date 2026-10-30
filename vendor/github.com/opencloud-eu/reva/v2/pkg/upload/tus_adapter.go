package upload

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	tusd "github.com/tus/tusd/v2/pkg/handler"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
)

var errNotImplemented = tusd.NewError("ERR_NOT_IMPLEMENTED", "use InitiateUpload on the CS3 API to start a new upload", http.StatusNotImplemented)

// tusAdapter adapts a single upload session to tusd's Upload interfaces.
type tusAdapter struct {
	session Session
	coord   *coordinator
}

func (u *tusAdapter) GetInfo(ctx context.Context) (tusd.FileInfo, error) {
	return u.session.GetInfo(ctx)
}

func (u *tusAdapter) GetReader(ctx context.Context) (io.ReadCloser, error) {
	return u.session.GetReader(ctx)
}

// ServeContent serves the staged bytes, honouring a Range header: postprocessing reads
// only the first bytes of a file larger than it scans.
func (u *tusAdapter) ServeContent(_ context.Context, w http.ResponseWriter, r *http.Request) error {
	f, err := os.Open(u.session.BinPath())
	if err != nil {
		return err
	}
	defer f.Close()
	http.ServeContent(w, r, "", time.Time{}, f)
	return nil
}

func (u *tusAdapter) WriteChunk(ctx context.Context, offset int64, src io.Reader) (int64, error) {
	return u.session.WriteChunk(ctx, offset, src)
}

// FinishUpload runs the coordinator's finish path once all bytes have arrived.
func (u *tusAdapter) FinishUpload(ctx context.Context) error {
	_, err := u.coord.finishUpload(u.session.Context(ctx), u.session)
	if err != nil {
		// tusd only answers the client, so log the cause here
		appctx.GetLogger(ctx).Error().Err(err).Str("uploadid", u.session.ID()).Msg("failed to finish upload")
	}

	// tusd answers an error type it does not recognise with a bare 500.
	switch err.(type) {
	case nil:
		return nil
	case errtypes.AlreadyExists:
		return tusd.NewError("ERR_ALREADY_EXISTS", err.Error(), http.StatusConflict)
	case errtypes.ResourceProcessing, errtypes.TooEarly:
		return tusd.NewError("ERR_TOO_EARLY", err.Error(), http.StatusTooEarly)
	case errtypes.Aborted:
		return tusd.NewError("ERR_PRECONDITION_FAILED", err.Error(), http.StatusPreconditionFailed)
	case errtypes.PreconditionFailed:
		return tusd.NewError("ERR_PRECONDITION_FAILED", err.Error(), http.StatusMethodNotAllowed)
	case errtypes.Locked:
		return tusd.NewError("ERR_LOCKED", err.Error(), http.StatusLocked)
	case errtypes.BadRequest:
		return tusd.NewError("ERR_BAD_REQUEST", err.Error(), http.StatusBadRequest)
	case errtypes.ChecksumMismatch:
		return tusd.NewError("ERR_CHECKSUM_MISMATCH", err.Error(), errtypes.StatusChecksumMismatch)
	case errtypes.IsPermissionDenied:
		return tusd.NewError("ERR_PERMISSION_DENIED", err.Error(), http.StatusForbidden)
	case errtypes.InsufficientStorage:
		return tusd.NewError("ERR_INSUFFICIENT_STORAGE", err.Error(), http.StatusInsufficientStorage)
	default:
		return err
	}
}

// Terminate discards an upload, so a cancelled one leaves nothing behind.
func (u *tusAdapter) Terminate(ctx context.Context) error {
	ref := u.session.Reference()

	// Rollback rather than Delete, which is permission-gated. It is a no-op unless
	// this upload still owns the node.
	if err := u.coord.fs.RollbackUpload(ctx, &ref, u.session.ID(), rollbackInfo(u.session, u.session.SizeDiff())); err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Str("uploadid", u.session.ID()).Msg("could not roll back terminated upload, keeping session")
		return err
	}
	// A node that was never created cannot be unmarked, which is the normal case here.
	if err := u.coord.fs.MarkProcessing(ctx, &ref, false, u.session.ID()); err != nil {
		appctx.GetLogger(ctx).Debug().Err(err).Str("uploadid", u.session.ID()).Msg("could not unmark terminated upload")
	}

	u.session.Cleanup(ctx, true, true)
	return nil
}

// DeclareLength records the total size for uploads initiated without one.
func (u *tusAdapter) DeclareLength(ctx context.Context, length int64) error {
	u.session.SetSize(length)
	u.session.SetSizeIsDeferred(false)
	return u.session.Persist(ctx)
}

// ConcatUploads appends the staged bytes of the partial uploads, in the order given.
func (u *tusAdapter) ConcatUploads(ctx context.Context, partials []tusd.Upload) error {
	file, err := os.OpenFile(u.session.BinPath(), os.O_WRONLY|os.O_APPEND, defaultFilePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, partial := range partials {
		p, ok := partial.(*tusAdapter)
		if !ok {
			return fmt.Errorf("coordinator: unexpected partial upload type %T", partial)
		}
		src, err := p.session.GetReader(ctx)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(file, src)
		src.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

// UseIn registers the coordinator as tusd's data store, in place of the driver's own.
func (c *coordinator) UseIn(composer *tusd.StoreComposer) {
	composer.UseCore(c)
	composer.UseTerminater(c)
	composer.UseConcater(c)
	composer.UseLengthDeferrer(c)
	composer.UseContentServer(c)
}

// NewUpload is unsupported: uploads always start through the CS3 InitiateUpload call.
func (c *coordinator) NewUpload(_ context.Context, _ tusd.FileInfo) (tusd.Upload, error) {
	return nil, errNotImplemented
}

// GetUpload wraps the session with the given id for the TUS data path.
func (c *coordinator) GetUpload(ctx context.Context, id string) (tusd.Upload, error) {
	session, err := c.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &tusAdapter{session: session, coord: c}, nil
}

// The As* methods let tusd reach the extension interfaces on a plain tusd.Upload.

func (c *coordinator) AsTerminatableUpload(up tusd.Upload) tusd.TerminatableUpload {
	return up.(*tusAdapter)
}

func (c *coordinator) AsLengthDeclarableUpload(up tusd.Upload) tusd.LengthDeclarableUpload {
	return up.(*tusAdapter)
}

func (c *coordinator) AsConcatableUpload(up tusd.Upload) tusd.ConcatableUpload {
	return up.(*tusAdapter)
}

func (c *coordinator) AsServableUpload(up tusd.Upload) tusd.ServableUpload {
	return up.(*tusAdapter)
}
