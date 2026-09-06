package workflow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	grpcmetadata "google.golang.org/grpc/metadata"

	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/templates"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/dav/requests"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/generator"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/preprocessor"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail/cache"
)

// Stater stats a file via the CS3 gateway.
type Stater interface {
	Stat(ctx context.Context, ref *providerv1beta1.Reference, auth string) (*providerv1beta1.StatResponse, error)
}

// FileDownloader opens a read stream on a file in storage. The returned reader
// must be consumed (and closed by the caller) exactly once; it is the raw
// response body and is never buffered or decoded here.
type FileDownloader interface {
	DownloadStream(ctx context.Context, ref *providerv1beta1.Reference, auth string) (io.ReadCloser, error)
}

// SpaceLookup resolves a path-only reference (ResourceId == nil) into a full
// reference by looking up the storage space that owns the absolute path via the
// CS3 gateway's ListStorageSpaces. This mirrors reva's ocdav spacelookup and is
// used for path-only requests (user home paths, /webdav/..., public links),
// which carry an absolute path but no space root.
type SpaceLookup interface {
	Resolve(ctx context.Context, ref *providerv1beta1.Reference, auth string) (*providerv1beta1.Reference, error)
}

// UserResolver resolves users for path-only thumbnail requests. WhoAmI maps an
// auth token to its user (used for /webdav/... where the URL carries no
// username); GetUserByClaim maps a username from /dav/files/{user}/... to its
// user, mirroring main's behavior.
type UserResolver interface {
	WhoAmI(ctx context.Context, auth string) (*userv1beta1.User, error)
	GetUserByClaim(ctx context.Context, claim, value string) (*userv1beta1.User, error)
}

// ErrFileProcessing is returned when the file is still being processed by the
// storage backend (e.g. a virus scan or conversion in flight). Callers should
// surface this as HTTP 425 Too Early with a Retry-After header.
var ErrFileProcessing = fmt.Errorf("file is processing")

// ErrNoThumbnail is returned when the file has no thumbnail; cached.
var ErrNoThumbnail = errors.New("thumbnails: the file has no thumbnail")

// ErrImageTooLarge is returned when the input image exceeds the configured
// maximum width/height. Callers should surface this as HTTP 403 Forbidden,
// matching the legacy thumbnail service error message.
var ErrImageTooLarge = errors.New("thumbnails: image is too large")

// ErrNotAFile is returned when the requested resource is not a regular file
// (e.g. a folder). Callers should surface this as HTTP 400 Bad Request with an
// "Unsupported file type" message, matching the legacy thumbnail service.
var ErrNotAFile = errors.New("thumbnails: resource is not a file")

// ErrPermissionDenied is returned when the requesting user lacks the download
// permission on the resource (e.g. a Secure Viewer share). Callers should
// surface this as HTTP 403 Forbidden, matching the legacy thumbnail service.
var ErrPermissionDenied = errors.New("thumbnails: no download permission")

// ErrNotFound is returned when the requested file could not be located or has no
// usable checksum. Callers should surface this as HTTP 404 Not Found with a
// "File could not be located" message, matching the legacy thumbnail service.
var ErrNotFound = errors.New("thumbnails: file could not be located")

// fileIsProcessing reports whether the resource carries the "processing" status
// in its opaque map, as set by the storage backend while a file is being handled.
func fileIsProcessing(info *providerv1beta1.ResourceInfo) bool {
	return utils.ReadPlainFromOpaque(info.GetOpaque(), "status") == "processing"
}

// processorID normalizes the requested processor to the identifier used in cache
// keys, matching the legacy behavior where an empty processor resolved to "thumbnail".
func processorID(processor string) string {
	if processor == "" {
		return "thumbnail"
	}
	return strings.ToLower(processor)
}

// thumbnailStorageKey builds the cache key: checksum[:2]/checksum[2:4]/checksum[4:]/wxh-<operation>.ext.
// The <operation> segment (fit-in, fill, stretch, ...) distinguishes requests that
// share a size but produce different images, so they never share a cache entry.
// The requested resolution is used (not the matched one) so the key can be
// computed before downloading the source. Caches regenerate when the configured
// resolutions or operations change.
func thumbnailStorageKey(checksum string, w, h int, ext, processor string) string {
	name := fmt.Sprintf("%dx%d-%s.%s", w, h, processor, ext)
	if len(checksum) >= 6 {
		return checksum[:2] + "/" + checksum[2:4] + "/" + checksum[4:] + "/" + name
	}
	return checksum + "/" + name
}

// ThumbnailWorkflow owns the complete thumbnail generation pipeline:
// stat → validate → cache check → download → preprocess → generate → cache → respond.
type ThumbnailWorkflow struct {
	generatorURL   string
	cache          cache.ThumbnailCache
	httpClient     *http.Client
	maxInputSize   uint64
	resolutions    *thumbnail.Resolutions
	webdavNS       string
	tika           thumbnail.Tika
	fontMapFile    string
	log            log.Logger
	stater         Stater
	fileDownloader FileDownloader
	spaceLookup    SpaceLookup
	userResolver   UserResolver
}

// NewWorkflow creates a ThumbnailWorkflow. generatorURL must be non-empty.
func NewWorkflow(opts ...Option) (*ThumbnailWorkflow, error) {
	w := &ThumbnailWorkflow{
		httpClient: &http.Client{},
		log:        log.NewLogger(log.Name("webdav")),
	}
	for _, opt := range opts {
		opt(w)
	}

	if w.generatorURL == "" {
		return nil, fmt.Errorf("generator URL is required")
	}

	if w.stater == nil {
		return nil, fmt.Errorf("stater is required")
	}

	if w.fileDownloader == nil {
		return nil, fmt.Errorf("file downloader is required")
	}

	return w, nil
}

// Option configures a ThumbnailWorkflow.
type Option func(*ThumbnailWorkflow)

// WithGeneratorURL sets the base URL of the thumbnail generator service.
func WithGeneratorURL(url string) Option {
	return func(w *ThumbnailWorkflow) { w.generatorURL = url }
}

// WithCache sets the thumbnail cache.
func WithCache(c cache.ThumbnailCache) Option {
	return func(w *ThumbnailWorkflow) { w.cache = c }
}

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(w *ThumbnailWorkflow) { w.httpClient = client }
}

// WithMaxInputSize sets the maximum input file size in bytes (0 = no limit).
func WithMaxInputSize(size uint64) Option {
	return func(w *ThumbnailWorkflow) { w.maxInputSize = size }
}

// WithResolutions sets the supported target resolutions used to snap a requested
// size onto a generator box. The generator is a dumb executor, so webdav owns the
// configured list and chooses the box; it only needs the list to pick the box.
func WithResolutions(rs *thumbnail.Resolutions) Option {
	return func(w *ThumbnailWorkflow) { w.resolutions = rs }
}

// WithWebdavNamespace sets the CS3 path layout for resolving user paths.
func WithWebdavNamespace(ns string) Option {
	return func(w *ThumbnailWorkflow) { w.webdavNS = ns }
}

// WithFontMapFile sets the font map file used for text thumbnail rendering.
// WithTika sets the Tika server the preprocessing falls back to.
func WithTika(t thumbnail.Tika) Option {
	return func(w *ThumbnailWorkflow) { w.tika = t }
}

func WithFontMapFile(file string) Option {
	return func(w *ThumbnailWorkflow) { w.fontMapFile = file }
}

// WithLogger sets the logger.
func WithLogger(l log.Logger) Option {
	return func(w *ThumbnailWorkflow) { w.log = l }
}

// WithStater sets the file stater.
func WithStater(s Stater) Option {
	return func(w *ThumbnailWorkflow) { w.stater = s }
}

// WithFileDownloader sets the file downloader.
func WithFileDownloader(d FileDownloader) Option {
	return func(w *ThumbnailWorkflow) { w.fileDownloader = d }
}

// WithSpaceLookup sets the resolver used to look up the storage space that owns
// a path-only reference. When unset, path-only references are passed through as-is.
func WithSpaceLookup(l SpaceLookup) Option {
	return func(w *ThumbnailWorkflow) { w.spaceLookup = l }
}

// WithUserResolver sets the resolver used to map an auth token to its user for
// path-only requests (/dav/files/{user}/..., /webdav/...). When unset, such
// requests cannot be absolutized and fall back to the raw path.
func WithUserResolver(r UserResolver) Option {
	return func(w *ThumbnailWorkflow) { w.userResolver = r }
}

// Execute handles a user thumbnail request (GET).
func (w *ThumbnailWorkflow) Execute(ctx context.Context, tr *requests.ThumbnailRequest, auth string, logger log.Logger) (data []byte, ext string, aIgnored bool, err error) {
	ref, err := w.resolveReference(ctx, tr, auth)
	if err != nil {
		return nil, "", false, err
	}

	data, ext, aIgnored, err = w.generate(ctx, ref, auth, tr, logger)
	return data, ext, aIgnored, err
}

// ExecutePublic handles a public link thumbnail request (GET).
func (w *ThumbnailWorkflow) ExecutePublic(ctx context.Context, tr *requests.ThumbnailRequest, auth string, logger log.Logger) (data []byte, ext string, aIgnored bool, err error) {
	ref, err := w.resolveReference(ctx, tr, auth)
	if err != nil {
		return nil, "", false, err
	}

	data, ext, aIgnored, err = w.generate(ctx, ref, auth, tr, logger)
	return data, ext, aIgnored, err
}

// Head performs a cache-only check for a public link thumbnail (HEAD).
// Returns nil error if the file exists and is supported, without triggering generation.
func (w *ThumbnailWorkflow) Head(ctx context.Context, tr *requests.ThumbnailRequest, auth string, logger log.Logger) error {
	ref, err := w.resolveReference(ctx, tr, auth)
	if err != nil {
		return err
	}

	statRsp, err := w.stater.Stat(ctx, ref, auth)
	if err != nil {
		logger.Debug().Err(err).Msg("could not stat file")
		return fmt.Errorf("stat: %w", err)
	}

	info := statRsp.GetInfo()

	if fileIsProcessing(info) {
		return ErrFileProcessing
	}

	if !info.GetPermissionSet().GetInitiateFileDownload() {
		return ErrPermissionDenied
	}

	if !w.supportsMimeType(info.GetMimeType()) {
		return fmt.Errorf("unsupported mime type: %s", info.GetMimeType())
	}

	if w.maxInputSize > 0 && info.GetSize() > w.maxInputSize {
		logger.Debug().Uint64("size", info.GetSize()).Uint64("max", w.maxInputSize).Msg("file too large for thumbnail")
		return ErrImageTooLarge
	}

	return nil
}

// resolveReference turns the request's reference into a fully-anchored CS3
// reference, mirroring reva's ocdav split of space-ID-based requests from
// path-only requests:
//   - If Ref carries a ResourceId (dav/spaces/{id}/...), it is used directly —
//     the URL already identifies the space root.
//   - Otherwise (path-only request) the absolute CS3 path is resolved to the
//     user's home-relative path and, when a SpaceLookup is configured, the
//     owning storage space is looked up so the reference is anchored at the
//     space root. This also covers public links (/public/<token>/...).
func (w *ThumbnailWorkflow) resolveReference(ctx context.Context, tr *requests.ThumbnailRequest, auth string) (*providerv1beta1.Reference, error) {
	ref := tr.Ref
	if ref == nil {
		return nil, fmt.Errorf("request carries no reference")
	}

	if ref.GetResourceId() != nil {
		// Space-ID based request: the URL carries the space root.
		return ref, nil
	}

	// Path-only request: absolutize the path relative to a user's home (the
	// legacy webdav behavior) before resolving the owning space. The user is the
	// one named in the URL (/dav/files/{user}/...) when present, otherwise the
	// token owner (/webdav/...).
	absPath := w.absolutizeUserPath(ctx, ref.GetPath(), tr.Identifier, auth)

	if w.spaceLookup == nil {
		return &providerv1beta1.Reference{Path: absPath}, nil
	}

	resolved, err := w.spaceLookup.Resolve(ctx, &providerv1beta1.Reference{Path: absPath}, auth)
	if err != nil {
		return nil, fmt.Errorf("resolve space for path %q: %w", absPath, err)
	}
	return resolved, nil
}

// absolutizeUserPath maps a path-only reference to an absolute CS3 path under a
// user's home, matching the legacy webdav handler. The user is resolved from the
// username in the URL (identifier, via GetUserByClaim) when present, otherwise
// from the auth token (via WhoAmI). Public link paths (already absolute under
// /public) and any path that is not relative to the caller are returned
// unchanged.
func (w *ThumbnailWorkflow) absolutizeUserPath(ctx context.Context, p, identifier, auth string) string {
	if strings.HasPrefix(p, "/") {
		return p
	}

	if w.userResolver == nil {
		// No resolver configured: we cannot absolutize the path.
		w.log.Warn().Msg("no user resolver configured, using raw path for thumbnail")
		return p
	}

	var (
		user *userv1beta1.User
		err  error
	)
	if identifier != "" {
		// /dav/files/{user}/...: honor the username from the URL.
		user, err = w.userResolver.GetUserByClaim(ctx, "username", identifier)
	} else {
		// /webdav/...: resolve the token owner.
		user, err = w.userResolver.WhoAmI(ctx, auth)
	}
	if err != nil {
		// Without a resolved user we cannot absolutize; fall back to the raw path.
		w.log.Warn().Err(err).Msg("could not resolve user for thumbnail path")
		return p
	}
	return filepath.Join(templates.WithUser(user, w.webdavNS), p)
}

func (w *ThumbnailWorkflow) generate(ctx context.Context, ref *providerv1beta1.Reference, auth string, tr *requests.ThumbnailRequest, logger log.Logger) ([]byte, string, bool, error) {
	statRsp, err := w.stater.Stat(ctx, ref, auth)
	if err != nil {
		logger.Debug().Err(err).Msg("could not stat file")
		return nil, "", false, fmt.Errorf("stat: %w", err)
	}

	info := statRsp.GetInfo()

	if fileIsProcessing(info) {
		return nil, "", false, ErrFileProcessing
	}

	if !info.GetPermissionSet().GetInitiateFileDownload() {
		return nil, "", false, ErrPermissionDenied
	}

	if !w.supportsMimeType(info.GetMimeType()) {
		return nil, "", false, fmt.Errorf("unsupported mime type: %s", info.GetMimeType())
	}

	if w.maxInputSize > 0 && info.GetSize() > w.maxInputSize {
		logger.Debug().Uint64("size", info.GetSize()).Uint64("max", w.maxInputSize).Msg("file too large for thumbnail")
		return nil, "", false, ErrImageTooLarge
	}

	checksum := info.GetChecksum().GetSum()
	if checksum == "" {
		// Without a checksum the cache key would be shared across all files, so
		// the resource is treated as not found (matching main's grpc service).
		logger.Debug().Msg("resource info is missing a checksum")
		return nil, "", false, ErrNotFound
	}
	mimeType := info.GetMimeType()

	// The output type follows the source mime (like main's GetExtForMime); when
	// the source has no dedicated image type (txt, audio) it falls back to the
	// requested extension.
	outputExt := generator.OutputFormat(tr.Extension)
	if ext := generator.ExtForMime(mimeType); ext != "" {
		outputExt = ext
	}

	// The operation is resolved up front so it can be part of the cache key: a
	// fill and a fit-in request with the same dimensions produce different images
	// and must not share a cache entry. aIgnored reports whether the legacy "a"
	// flag was overridden by an explicit processor, so the client can be told.
	operation, aIgnored := w.matchOperation(tr, mimeType)

	// The cache key uses the requested resolution (like main's PrepareRequest),
	// so it can be computed before downloading. Checking the cache first means a
	// hit returns without touching storage, matching main's fast path.
	cacheKey := thumbnailStorageKey(checksum, int(tr.Width), int(tr.Height), outputExt, processorID(operation))

	if w.cache != nil {
		if cached, err := w.cache.Get(cacheKey); err == nil {
			// empty entry: known to have no thumbnail
			if len(cached) == 0 {
				return nil, "", false, ErrNoThumbnail
			}
			return cached, outputExt, aIgnored, nil
		}
	}

	// webdav is the sizing brain: it snaps the requested size onto a configured
	// resolution (orientation-aware). It never inspects the image bytes; the
	// generator is a dumb executor that just fits within the given box.
	reqBox := image.Rect(0, 0, int(tr.Width), int(tr.Height))
	box := reqBox
	if w.resolutions != nil {
		box = w.resolutions.Match(reqBox)
	}

	// webdav never asks the generator to upscale: it snaps onto a resolution at
	// least as large as requested, so the box is always >= the source intent.
	// The no_upscale() filter makes the generator behave like real imagor for
	// any other client that posts smaller boxes.
	genURL := generator.BuildURL(w.generatorURL, int32(box.Dx()), int32(box.Dy()), operation, outputExt, true)

	// Produce the source image to send to the generator, never decoding an image
	// in webdav: real images (incl. gif) are streamed straight through undecoded;
	// non-image sources are converted to image bytes here so the generator always
	// receives an image.
	imgStream, cleanup, err := w.sourceImage(ctx, ref, auth, mimeType, tr.Filename, logger)
	if err != nil {
		logger.Error().Err(err).Msg("could not obtain source image for thumbnail")
		return nil, "", false, err
	}
	defer cleanup()

	thumbBytes, err := w.postToGenerator(ctx, genURL, imgStream, tr.Filename)
	if err != nil {
		logger.Error().Err(err).Msg("could not generate thumbnail")
		return nil, "", false, fmt.Errorf("generate: %w", err)
	}

	if w.cache != nil {
		if err := w.cache.Put(cacheKey, thumbBytes); err != nil {
			logger.Debug().Err(err).Msg("could not cache thumbnail")
		}
	}

	return thumbBytes, outputExt, aIgnored, nil
}

// sourceImage returns a reader over the source image to send to the generator,
// plus a cleanup func to release any resources (download body, temp buffer). It
// never decodes an image in webdav:
//   - real images (image/*, incl. gif) are streamed straight through from storage
//     undecoded; the generator handles decoding and multi-frame gifs itself.
//   - text/plain is rendered to an image here (the only true conversion).
//   - audio and geogebra sources have their embedded image extracted to bytes.
func (w *ThumbnailWorkflow) sourceImage(ctx context.Context, ref *providerv1beta1.Reference, auth, mimeType, filename string, logger log.Logger) (io.Reader, func(), error) {
	m, _, _ := mime.ParseMediaType(mimeType)

	if strings.HasPrefix(m, "image/") {
		body, err := w.fileDownloader.DownloadStream(ctx, ref, auth)
		if err != nil {
			return nil, nil, fmt.Errorf("download: %w", err)
		}
		return body, func() { _ = body.Close() }, nil
	}

	body, err := w.fileDownloader.DownloadStream(ctx, ref, auth)
	if err != nil {
		return nil, nil, fmt.Errorf("download: %w", err)
	}
	defer body.Close()

	fileBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, nil, fmt.Errorf("read source: %w", err)
	}

	ppOpts := map[string]any{
		"fontFileMap": w.fontMapFile,
		"tika":        w.tika,
		"filename":    filename,
	}
	img, err := preprocessor.ForType(mimeType, ppOpts).Convert(bytes.NewReader(fileBytes))
	if errors.Is(err, preprocessor.ErrNoThumbnail) {
		return nil, nil, ErrNoThumbnail
	}
	if img == nil || err != nil {
		logger.Debug().Err(err).Msg("could not convert file to image")
		return nil, nil, fmt.Errorf("could not get image")
	}

	data, _, err := encodeForUpload(img, mimeType)
	if err != nil {
		return nil, nil, fmt.Errorf("encode converted image: %w", err)
	}
	return bytes.NewReader(data), func() {}, nil
}

// matchOperation resolves the generator resize/crop operation for a request from
// the client's processor and the legacy "a" flag. An explicit processor always
// wins; when no processor is given the default depends on the source type (gifs
// are resized, everything else is filled). webdav has already snapped the box
// onto a configured resolution; this only chooses how to fit within that box:
//   - resize            -> stretch (distort to the exact box)
//   - fill              -> fill (center-crop to the box)
//   - fit / fit-in      -> fit-in (preserve aspect, fit in box, never upscale)
//   - thumbnail         -> fill (legacy alias; crops to the exact square box)
//   - (none), gif       -> stretch (resize for gifs)
//   - (none), non-gif   -> fill (default = thumbnail = fill)
//
// "thumbnail" and "fit" are legacy aliases. They map onto the imagor operations
// above but IGNORE the legacy "a" and "scalingup" query parameters — callers that
// need aspect/upscaling control should use the imagor names directly ("fill",
// "fit-in", "stretch"). The second return value reports whether the legacy "a"
// flag was ignored because an explicit processor overrode it with a different
// behavior, so the caller can tell the client via a response header.
func (w *ThumbnailWorkflow) matchOperation(tr *requests.ThumbnailRequest, mimeType string) (string, bool) {
	proc := strings.ToLower(strings.TrimSpace(tr.Processor))

	var operation string
	switch proc {
	case "resize":
		operation = generator.OpStretch
	case "fill", "thumbnail":
		operation = generator.OpFill
	case "fit", "fit-in":
		operation = generator.OpFitIn
	default: // no processor: default behavior depends on source type
		if strings.HasPrefix(mimeType, "image/gif") {
			operation = generator.OpStretch
		} else {
			operation = generator.OpFill
		}
	}

	// The legacy "a" flag is a hint (a=1/absent -> preserve aspect / fit-in,
	// a=0 -> fill). An explicit processor always wins over it; report when the
	// processor's behavior contradicts what "a" asked so the client can be told
	// via a response header. stretch is neutral (the box is distorted to fit) so
	// "a" has no meaning for it and never counts as ignored.
	aIgnored := false
	if proc != "" {
		switch operation {
		case generator.OpFill:
			aIgnored = tr.Aspect // a asked for fit-in but we filled
		case generator.OpFitIn:
			aIgnored = !tr.Aspect // a asked for fill but we fit-in
		}
	}

	return operation, aIgnored
}

// postToGenerator streams the source image to the generator as a multipart
// POST. The body is produced by an io.Pipe so the (potentially large) image is
// never fully buffered in webdav: the writer goroutine copies the reader into
// the form part on demand. No Content-Length is set, so the request uses chunked
// transfer-encoding; both imagor and the thumbnailer parse the multipart body
// without needing it (imagor enforces its size limit after reading the part).
func (w *ThumbnailWorkflow) postToGenerator(ctx context.Context, url string, img io.Reader, filename string) ([]byte, error) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		part, err := writer.CreateFormFile("image", filename)
		if err != nil {
			errCh <- fmt.Errorf("create form file: %w", err)
			return
		}
		if _, err := io.Copy(part, img); err != nil {
			errCh <- fmt.Errorf("write form data: %w", err)
			return
		}
		if err := writer.Close(); err != nil {
			errCh <- fmt.Errorf("close multipart writer: %w", err)
			return
		}
		errCh <- nil
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
	if err != nil {
		pr.Close()
		return nil, fmt.Errorf("create generator request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// Forward the request ID so the generator's logs can be correlated with this
	// request. No auth token is sent: the generator is an unauthenticated
	// internal service (bound to loopback by default).
	if reqID := chimiddleware.GetReqID(ctx); reqID != "" {
		req.Header.Set("X-Request-ID", reqID)
	}

	httpRsp, err := w.httpClient.Do(req)
	if err != nil {
		pr.Close()
		return nil, fmt.Errorf("generator request: %w", err)
	}
	defer httpRsp.Body.Close()

	if writeErr := <-errCh; writeErr != nil {
		io.Copy(io.Discard, httpRsp.Body)
		return nil, writeErr
	}

	if httpRsp.StatusCode < 200 || httpRsp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(httpRsp.Body)
		// The generator reports ErrMaxResolutionExceeded (422) when the decoded
		// input exceeds its configured max width/height. Translate it to the
		// legacy "image is too large" error so the DAV layer returns 403, matching
		// the pre-refactor behavior where webdav enforced the dimension limit.
		if httpRsp.StatusCode == http.StatusUnprocessableEntity {
			return nil, ErrImageTooLarge
		}
		return nil, fmt.Errorf("generator returned status %d: %s", httpRsp.StatusCode, string(respBody))
	}

	rspData, err := io.ReadAll(httpRsp.Body)
	if err != nil {
		return nil, fmt.Errorf("read generator response: %w", err)
	}

	return rspData, nil
}

// supportsMimeType is the built-in list plus what Tika is asked for.
func (w *ThumbnailWorkflow) supportsMimeType(mimeType string) bool {
	return thumbnail.IsMimeTypeSupported(mimeType) || w.tika.Supports(mimeType)
}

// gatewaySelector is the interface for selecting a gateway client.
type gatewaySelector interface {
	Next(...pool.Option) (gatewayv1beta1.GatewayAPIClient, error)
}

// gatewayUserResolver implements UserResolver using the CS3 gateway's WhoAmI.
type gatewayUserResolver struct {
	selector gatewaySelector
}

func (g *gatewayUserResolver) WhoAmI(ctx context.Context, auth string) (*userv1beta1.User, error) {
	client, err := g.selector.Next()
	if err != nil {
		return nil, fmt.Errorf("get gateway client: %w", err)
	}

	userRes, err := client.WhoAmI(ctx, &gatewayv1beta1.WhoAmIRequest{Token: auth})
	if err != nil {
		return nil, fmt.Errorf("whoami: %w", err)
	}
	if userRes.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		return nil, fmt.Errorf("whoami: %s", userRes.GetStatus().GetMessage())
	}

	return userRes.GetUser(), nil
}

func (g *gatewayUserResolver) GetUserByClaim(ctx context.Context, claim, value string) (*userv1beta1.User, error) {
	client, err := g.selector.Next()
	if err != nil {
		return nil, fmt.Errorf("get gateway client: %w", err)
	}

	userRes, err := client.GetUserByClaim(ctx, &userv1beta1.GetUserByClaimRequest{Claim: claim, Value: value})
	if err != nil {
		return nil, fmt.Errorf("get user by claim: %w", err)
	}
	if userRes.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		return nil, fmt.Errorf("get user by claim: %s", userRes.GetStatus().GetMessage())
	}

	return userRes.GetUser(), nil
}

// NewGatewayUserResolver creates a UserResolver backed by the CS3 gateway.
func NewGatewayUserResolver(selector gatewaySelector) UserResolver {
	return &gatewayUserResolver{selector: selector}
}

// gatewayStater implements Stater using the CS3 gateway.
type gatewayStater struct {
	selector gatewaySelector
}

func (g *gatewayStater) Stat(ctx context.Context, ref *providerv1beta1.Reference, auth string) (*providerv1beta1.StatResponse, error) {
	client, err := g.selector.Next()
	if err != nil {
		return nil, fmt.Errorf("get gateway client: %w", err)
	}

	ctx = grpcmetadata.AppendToOutgoingContext(ctx, revactx.TokenHeader, auth)

	rsp, err := client.Stat(ctx, &providerv1beta1.StatRequest{Ref: ref})
	if err != nil {
		return nil, fmt.Errorf("stat request: %w", err)
	}

	if rsp.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		return nil, fmt.Errorf("stat failed: %s", rsp.GetStatus().GetMessage())
	}

	if rsp.GetInfo().GetType() != providerv1beta1.ResourceType_RESOURCE_TYPE_FILE {
		return nil, ErrNotAFile
	}

	return rsp, nil
}

// gatewayFileDownloader implements FileDownloader using the CS3 gateway.
type gatewayFileDownloader struct {
	selector   gatewaySelector
	httpClient *http.Client
}

func (g *gatewayFileDownloader) DownloadStream(ctx context.Context, ref *providerv1beta1.Reference, auth string) (io.ReadCloser, error) {
	client, err := g.selector.Next()
	if err != nil {
		return nil, fmt.Errorf("get gateway client: %w", err)
	}

	ctx = grpcmetadata.AppendToOutgoingContext(ctx, revactx.TokenHeader, auth)

	rsp, err := client.InitiateFileDownload(ctx, &providerv1beta1.InitiateFileDownloadRequest{Ref: ref})
	if err != nil {
		return nil, fmt.Errorf("initiate download: %w", err)
	}

	ep, _ := extractProtocol(rsp.GetProtocols())
	if ep == "" {
		return nil, fmt.Errorf("no suitable download protocol")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}
	// Authenticate the data request with the user's token; the dataprovider
	// (exposed or behind the proxy) validates it directly.
	req.Header.Set(revactx.TokenHeader, auth)

	httpRsp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request: %w", err)
	}

	if httpRsp.StatusCode != http.StatusOK {
		httpRsp.Body.Close()
		return nil, fmt.Errorf("download failed with status %d", httpRsp.StatusCode)
	}

	// Return the live response body; the caller owns and closes it. The bytes are
	// never buffered or decoded in webdav.
	return httpRsp.Body, nil
}

func extractProtocol(protocols []*gatewayv1beta1.FileDownloadProtocol) (endpoint, token string) {
	for _, p := range protocols {
		if p.GetProtocol() == "spaces" || p.GetProtocol() == "simple" {
			return p.GetDownloadEndpoint(), p.GetToken()
		}
	}

	for _, p := range protocols {
		if p.GetDownloadEndpoint() != "" && p.GetToken() != "" {
			return p.GetDownloadEndpoint(), p.GetToken()
		}
	}
	return "", ""
}

// NewGatewayStater creates a Stater backed by the CS3 gateway.
func NewGatewayStater(selector gatewaySelector) Stater {
	return &gatewayStater{selector: selector}
}

// NewGatewayFileDownloader creates a FileDownloader backed by the CS3 gateway.
func NewGatewayFileDownloader(selector gatewaySelector, httpClient *http.Client) FileDownloader {
	return &gatewayFileDownloader{selector: selector, httpClient: httpClient}
}

// gatewaySpaceLookup implements SpaceLookup using the CS3 gateway's
// ListStorageSpaces, mirroring reva's ocdav spacelookup.LookUpStorageSpaceForPath.
type gatewaySpaceLookup struct {
	selector gatewaySelector
}

func (g *gatewaySpaceLookup) Resolve(ctx context.Context, ref *providerv1beta1.Reference, auth string) (*providerv1beta1.Reference, error) {
	client, err := g.selector.Next()
	if err != nil {
		return nil, fmt.Errorf("get gateway client: %w", err)
	}

	ctx = grpcmetadata.AppendToOutgoingContext(ctx, revactx.TokenHeader, auth)

	lSSReq := &providerv1beta1.ListStorageSpacesRequest{
		Opaque: &typesv1beta1.Opaque{
			Map: map[string]*typesv1beta1.OpaqueEntry{
				"path":   {Decoder: "plain", Value: []byte(ref.GetPath())},
				"unique": {Decoder: "plain", Value: []byte("true")},
			},
		},
	}

	lSSRes, err := client.ListStorageSpaces(ctx, lSSReq)
	if err != nil {
		return nil, fmt.Errorf("list storage spaces: %w", err)
	}
	if lSSRes.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		return nil, fmt.Errorf("list storage spaces: %s", lSSRes.GetStatus().GetMessage())
	}

	switch len(lSSRes.GetStorageSpaces()) {
	case 0:
		return nil, fmt.Errorf("no storage space found for path %q", ref.GetPath())
	case 1:
		space := lSSRes.GetStorageSpaces()[0]
		if space.GetRoot() == nil {
			return nil, fmt.Errorf("storage space for path %q has no root", ref.GetPath())
		}
		return &providerv1beta1.Reference{
			ResourceId: space.GetRoot(),
			Path:       utils.MakeRelativePath(strings.TrimPrefix(ref.GetPath(), g.spaceMountPath(space))),
		}, nil
	default:
		return nil, fmt.Errorf("too many storage spaces returned for path %q", ref.GetPath())
	}
}

// spaceMountPath extracts the mount path recorded in a storage space's opaque map,
// used to trim the absolute path down to a space-relative one. It returns "" when
// the space is not mounted at a known path (in which case the full path is kept).
func (g *gatewaySpaceLookup) spaceMountPath(space *providerv1beta1.StorageSpace) string {
	if space.GetOpaque() == nil || space.GetOpaque().GetMap()["path"] == nil {
		return ""
	}
	return string(space.GetOpaque().GetMap()["path"].GetValue())
}

// NewGatewaySpaceLookup creates a SpaceLookup backed by the CS3 gateway.
func NewGatewaySpaceLookup(selector gatewaySelector) SpaceLookup {
	return &gatewaySpaceLookup{selector: selector}
}

// ErrPublicLinkPasswordRequired is returned when a password-protected public
// link is accessed without (or with the wrong) password. Callers should surface
// this as HTTP 404 Not Found so the resource's existence is not revealed.
var ErrPublicLinkPasswordRequired = errors.New("public link requires a password")

// ErrPublicLinkExpired is returned when a public link token has expired.
// Callers should surface this as HTTP 410 Gone.
var ErrPublicLinkExpired = errors.New("public link has expired")

// ResolvePublicLinkAuth authenticates a public link token via the gateway. It
// returns ErrPublicLinkPasswordRequired or ErrPublicLinkExpired (wrapped) when
// the gateway reports the corresponding gRPC status code, so callers can branch
// on the error type instead of matching error text.
func ResolvePublicLinkAuth(ctx context.Context, r *http.Request, publicLinkToken string, selector gatewaySelector) (string, error) {
	gatewayClient, err := selector.Next()
	if err != nil {
		return "", fmt.Errorf("could not get gateway client: %w", err)
	}

	q := r.URL.Query()
	var rsp *gatewayv1beta1.AuthenticateResponse

	if q.Get("signature") != "" && q.Get("expiration") != "" {
		sig := q.Get("signature")
		exp := q.Get("expiration")
		rsp, err = gatewayClient.Authenticate(ctx, &gatewayv1beta1.AuthenticateRequest{
			Type:         "publicshares",
			ClientId:     publicLinkToken,
			ClientSecret: strings.Join([]string{"signature", sig, exp}, "|"),
		})
	} else {
		rsp, err = gatewayClient.Authenticate(ctx, &gatewayv1beta1.AuthenticateRequest{
			Type:         "publicshares",
			ClientId:     publicLinkToken,
			ClientSecret: "password|",
		})
	}

	if err != nil {
		return "", fmt.Errorf("could not authenticate public link: %w", err)
	}

	switch rsp.GetStatus().GetCode() {
	case rpcv1beta1.Code_CODE_OK:
		return rsp.GetToken(), nil
	case rpcv1beta1.Code_CODE_PERMISSION_DENIED:
		return "", fmt.Errorf("%w: %s", ErrPublicLinkPasswordRequired, rsp.GetStatus().GetMessage())
	case rpcv1beta1.Code_CODE_FAILED_PRECONDITION:
		return "", fmt.Errorf("%w: %s", ErrPublicLinkExpired, rsp.GetStatus().GetMessage())
	default:
		return "", fmt.Errorf("public link authentication failed: code=%s message=%s", rsp.GetStatus().GetCode(), rsp.GetStatus().GetMessage())
	}
}
