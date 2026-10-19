package workflow

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strconv"
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
// user, looked up as the requester, whose tenant the user provider searches.
type UserResolver interface {
	WhoAmI(ctx context.Context, auth string) (*userv1beta1.User, error)
	GetUserByClaim(ctx context.Context, claim, value, auth string) (*userv1beta1.User, error)
}

// ErrFileProcessing is returned when the file is still being processed by the
// storage backend (e.g. a virus scan or conversion in flight). Callers should
// surface this as HTTP 425 Too Early with a Retry-After header.
var ErrFileProcessing = fmt.Errorf("file is processing")

// ErrNoThumbnail is the answer for a file that carries none at all. It is
// cached as an empty entry, so the next request does not ask Tika again.
var ErrNoThumbnail = errors.New("thumbnails: the file has no thumbnail")

// ErrImageTooLarge is returned when the input image exceeds the configured
// maximum width/height. Callers should surface this as HTTP 403 Forbidden.
var ErrImageTooLarge = errors.New("thumbnails: image is too large")

// ErrNotAFile is returned when the requested resource is not a regular file
// (e.g. a folder). Callers should surface this as HTTP 400 Bad Request with an
// "Unsupported file type" message.
var ErrNotAFile = errors.New("thumbnails: resource is not a file")

// ErrPermissionDenied is returned when the requesting user lacks the download
// permission on the resource (e.g. a Secure Viewer share). Callers should
// surface this as HTTP 403 Forbidden.
var ErrPermissionDenied = errors.New("thumbnails: no download permission")

// ErrNotFound is returned when the requested file could not be located or has no
// usable checksum. Callers should surface this as HTTP 404 Not Found with a
// "File could not be located" message.
var ErrNotFound = errors.New("thumbnails: file could not be located")

// ErrNoStorageSpace is returned when no storage space owns the requested path,
// e.g. a path in another user's home or a non-existent path. Callers should
// surface this as HTTP 404 Not Found.
var ErrNoStorageSpace = errors.New("thumbnails: no storage space found for path")

// ErrUnsupportedFileType is returned when the file's mime type is not in the
// supported set (e.g. PDF, ODT, ZIP). Callers should surface this as HTTP 404
// Not Found with an "Unsupported file type" message, matching the legacy
// thumbnail service.
var ErrUnsupportedFileType = errors.New("unsupported file type")

// GeneratorBusyError is returned when the generator refuses a request because
// its concurrency limit is reached. RetryAfterSeconds carries its own hint.
type GeneratorBusyError struct {
	RetryAfterSeconds int
}

func (e *GeneratorBusyError) Error() string {
	return fmt.Sprintf("thumbnails: generator is busy, retry after %ds", e.RetryAfterSeconds)
}

// fileIsProcessing reports whether the resource carries the "processing" status
// in its opaque map, as set by the storage backend while a file is being handled.
func fileIsProcessing(info *providerv1beta1.ResourceInfo) bool {
	return utils.ReadPlainFromOpaque(info.GetOpaque(), "status") == "processing"
}

// thumbnailStorageKey builds the cache key: hash[:2]/hash[2:4]/hash[4:]/wxh-<operation>.format.
// The <operation> segment (fit-in, fill, stretch, ...) distinguishes requests that
// share a size but produce different images, so they never share a cache entry.
// The key carries the snapped box, not the requested size, so every request that
// rounds to the same box shares one entry. Caches regenerate when the configured
// resolutions or operations change.
//
// The checksum is hashed rather than used as it comes: it is not this service's
// to trust, an OCM share takes it from a remote PROPFIND, and it ends up as a
// path on disk and as an object key. The source type is hashed with it, because
// the same bytes under two types are not the same thumbnail: the type picks the
// extractor, and what it finds depends on it.
func thumbnailStorageKey(checksum, mimeType string, w, h int, format, operation string) string {
	sum := sha256.Sum256([]byte(checksum + "\x00" + mimeType))
	hash := hex.EncodeToString(sum[:])

	return hash[:2] + "/" + hash[2:4] + "/" + hash[4:] + "/" + fmt.Sprintf("%dx%d-%s.%s", w, h, operation, format)
}

// ThumbnailWorkflow owns the complete thumbnail generation pipeline:
// stat → validate → cache check → download → preprocess → generate → cache → respond.
type ThumbnailWorkflow struct {
	generatorURL   string
	cache          cache.ThumbnailCache
	httpClient     *http.Client
	maxInputSize   uint64
	sizes          *thumbnail.Sizes
	typeMapping    thumbnail.TypeMapping
	sourceTypes    thumbnail.SourceTypes
	tika           thumbnail.Tika
	formats        generator.Formats
	webdavNS       string
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

// WithSizes sets the ladder of edge lengths a generated thumbnail may have. The
// generator is a dumb executor, so webdav owns the configured ladder and decides
// the box.
func WithSizes(s *thumbnail.Sizes) Option {
	return func(w *ThumbnailWorkflow) { w.sizes = s }
}

// WithWebdavNamespace sets the CS3 path layout for resolving user paths.
func WithWebdavNamespace(ns string) Option {
	return func(w *ThumbnailWorkflow) { w.webdavNS = ns }
}

// WithSourceTypes sets the types a preview is attempted for.
func WithSourceTypes(t thumbnail.SourceTypes) Option {
	return func(w *ThumbnailWorkflow) { w.sourceTypes = t }
}

// WithFormats sets the formats the generator writes, by type.
func WithFormats(f generator.Formats) Option {
	return func(w *ThumbnailWorkflow) { w.formats = f }
}

// WithTypeMapping sets the type a thumbnail of a given image type is returned in.
func WithTypeMapping(m thumbnail.TypeMapping) Option {
	return func(w *ThumbnailWorkflow) { w.typeMapping = m }
}

// WithTika sets the Tika server that thumbnails what webdav cannot read itself.
func WithTika(t thumbnail.Tika) Option {
	return func(w *ThumbnailWorkflow) { w.tika = t }
}

// WithFontMapFile sets the font map file used for text thumbnail rendering.
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

// Thumbnail is what a request produces: the image and the type it is returned in.
type Thumbnail struct {
	Data        []byte
	ContentType string
}

// Execute handles a user thumbnail request (GET).
func (w *ThumbnailWorkflow) Execute(ctx context.Context, tr *requests.ThumbnailRequest, auth string, logger log.Logger) (Thumbnail, error) {
	ref, err := w.resolveReference(ctx, tr, auth)
	if err != nil {
		return Thumbnail{}, err
	}

	return w.generate(ctx, ref, auth, tr, logger)
}

// Head performs a cache-only check for a public link thumbnail (HEAD).
// Returns nil error if the file exists and is supported, without triggering generation.
func (w *ThumbnailWorkflow) Head(ctx context.Context, tr *requests.ThumbnailRequest, auth string, logger log.Logger) error {
	ref, err := w.resolveReference(ctx, tr, auth)
	if err != nil {
		return err
	}

	_, err = w.statSource(ctx, ref, auth, logger)

	return err
}

// statSource stats the source and reports whether a thumbnail may be made of it
// at all, which is the same question for a HEAD and for a generation.
func (w *ThumbnailWorkflow) statSource(ctx context.Context, ref *providerv1beta1.Reference, auth string, logger log.Logger) (*providerv1beta1.ResourceInfo, error) {
	statRsp, err := w.stater.Stat(ctx, ref, auth)
	if err != nil {
		logger.Debug().Err(err).Msg("could not stat file")

		return nil, fmt.Errorf("stat: %w", err)
	}

	info := statRsp.GetInfo()

	if fileIsProcessing(info) {
		return nil, ErrFileProcessing
	}

	if !info.GetPermissionSet().GetInitiateFileDownload() {
		return nil, ErrPermissionDenied
	}

	if !w.supportsMimeType(info.GetMimeType()) {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFileType, info.GetMimeType())
	}

	if w.maxInputSize > 0 && info.GetSize() > w.maxInputSize {
		logger.Debug().Uint64("size", info.GetSize()).Uint64("max", w.maxInputSize).Msg("file too large for thumbnail")

		return nil, ErrImageTooLarge
	}

	return info, nil
}

// resolveReference turns the request's reference into a fully-anchored CS3
// reference, mirroring reva's ocdav split of space-ID-based requests from
// path-only requests:
//   - If Ref carries a ResourceId (dav/spaces/{id}/...), it is used directly,
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
		if errors.Is(err, ErrNoStorageSpace) {
			// No storage space owns the path (e.g. a file in another user's home
			// or a non-existent path). Surface it as 404, matching the legacy
			// thumbnail service.
			return nil, fmt.Errorf("%w: %v", ErrNotFound, err)
		}
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
		user, err = w.userResolver.GetUserByClaim(ctx, "username", identifier, auth)
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

func (w *ThumbnailWorkflow) generate(ctx context.Context, ref *providerv1beta1.Reference, auth string, tr *requests.ThumbnailRequest, logger log.Logger) (Thumbnail, error) {
	info, err := w.statSource(ctx, ref, auth, logger)
	if err != nil {
		return Thumbnail{}, err
	}

	checksum := info.GetChecksum().GetSum()
	if checksum == "" {
		// Without a checksum the cache key would be shared across all files, so
		// the resource is treated as not found.
		logger.Debug().Msg("resource info is missing a checksum")
		return Thumbnail{}, ErrNotFound
	}
	mimeType := info.GetMimeType()

	// The thumbnail is returned in the type the source type maps to, asked for
	// by the name the generator's filter uses. What an extractor hands on is
	// re-encoded to that type, because the request has to decide it: a cache
	// hit knows the key, not the bytes. Keeping a png cover a png needs the
	// embedded type before the key is built, which is what the preview presence
	// work puts into the file's metadata.
	outputType, outputFormat := w.outputFor(mimeType)

	// The operation is resolved up front so it can be part of the cache key: a
	// fill and a fit-in request with the same dimensions produce different images
	// and must not share a cache entry. noUpscale reports whether the generator
	// must not enlarge the source (fit-in only).
	operation, noUpscale := w.matchOperation(tr, mimeType)

	// webdav is the sizing brain: it snaps each axis of the requested size onto
	// the configured ladder. It never inspects the image bytes; the generator is
	// a dumb executor that just fits within the given box.
	box := image.Rect(0, 0, int(tr.Width), int(tr.Height))
	if w.sizes != nil {
		box = w.sizes.Snap(box)
	}

	// The cache key uses the snapped box, not what the client asked for, so
	// requests that produce the same image share one entry and the number of
	// entries per file stays bounded by the ladder. It needs nothing but the
	// request, so a hit still returns without touching storage.
	cacheKey := thumbnailStorageKey(checksum, mimeType, box.Dx(), box.Dy(), outputFormat, operation)

	if w.cache != nil {
		cached, err := w.cache.Get(cacheKey)
		switch {
		case err == nil:
			// an empty entry is a file we know carries no thumbnail
			if len(cached) == 0 {
				return Thumbnail{}, ErrNoThumbnail
			}

			return Thumbnail{Data: cached, ContentType: outputType}, nil
		case !errors.Is(err, cache.ErrCacheMiss):
			// a miss is the normal answer; anything else is the cache itself
			// failing, and the thumbnail is made again without a word otherwise
			logger.Error().Err(err).Str("key", cacheKey).Msg("could not read from the thumbnail cache")
		}
	}

	// Produce the source image to send to the generator, never decoding an image
	// in webdav: real images (incl. gif) are streamed straight through undecoded;
	// non-image sources are converted to image bytes here so the generator always
	// receives an image.
	imgStream, srcType, cleanup, err := w.sourceImage(ctx, ref, auth, mimeType, tr.Filename, logger)
	if err != nil {
		if errors.Is(err, ErrNoThumbnail) {
			// a file that carries no thumbnail is an answer, not a failure, and
			// the empty entry keeps the next request from asking again
			logger.Debug().Err(err).Msg("no source image for thumbnail")
			if w.cache != nil {
				_ = w.cache.Put(cacheKey, nil)
			}

			return Thumbnail{}, err
		}

		logger.Error().Err(err).Msg("could not obtain source image for thumbnail")

		return Thumbnail{}, err
	}
	defer cleanup()

	// webdav snaps the requested size onto a configured resolution and lets the
	// generator fill that box. Only fit-in suppresses upscaling (noUpscale); for
	// fill/thumbnail/resize the generator may enlarge a small source (e.g. a text
	// file rendered at a fixed canvas) to fill the snapped box, which is what
	// imagor's default resize does.
	genURL := generator.BuildURL(w.generatorURL, int32(box.Dx()), int32(box.Dy()), operation, outputFormat, noUpscale)
	thumbBytes, thumbType, err := w.postToGenerator(ctx, genURL, imgStream, tr.Filename, srcType)
	if err != nil {
		logger.Error().Err(err).Msg("could not generate thumbnail")
		return Thumbnail{}, fmt.Errorf("generate: %w", err)
	}
	if thumbType != "" && thumbType != outputType {
		// the answer is always the type we cache under, so the same request
		// cannot report one type now and another one after a cache hit. A
		// generator that writes something else than it was asked for means the
		// configured formats do not match it, which an operator has to fix.
		logger.Warn().Str("asked", outputType).Str("answered", thumbType).
			Msg("generator wrote a type we did not ask for")
	}

	if w.cache != nil {
		if err := w.cache.Put(cacheKey, thumbBytes); err != nil {
			logger.Debug().Err(err).Msg("could not cache thumbnail")
		}
	}

	return Thumbnail{Data: thumbBytes, ContentType: outputType}, nil
}

// sourceImage returns a reader over the source image to send to the generator,
// plus a cleanup func to release any resources (download body, temp buffer). It
// never decodes an image in webdav:
//   - real images (image/*, incl. gif) are streamed straight through from storage
//     undecoded; the generator handles decoding and multi-frame gifs itself.
//   - text/plain is rendered to an image here (the only true conversion).
//   - audio and geogebra sources have their embedded image extracted to bytes.
func (w *ThumbnailWorkflow) sourceImage(ctx context.Context, ref *providerv1beta1.Reference, auth, mimeType, filename string, logger log.Logger) (io.Reader, string, func(), error) {
	m, _, _ := mime.ParseMediaType(mimeType)

	// the preprocessor decides what a type needs. No converter means the file
	// is handed to the generator as it is, which is the case for every type it
	// reads itself.
	converter := preprocessor.ForType(mimeType, preprocessor.Options{
		FontMapFile:  w.fontMapFile,
		MaxImageSize: w.maxInputSize,
		Tika:         w.tika,
		Filename:     filename,
	})

	body, err := w.fileDownloader.DownloadStream(ctx, ref, auth)
	if err != nil {
		return nil, "", nil, fmt.Errorf("download: %w", err)
	}

	if converter == nil {
		return body, m, func() { _ = body.Close() }, nil
	}
	defer body.Close()

	fileBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, "", nil, fmt.Errorf("read source: %w", err)
	}

	img, err := converter.Convert(bytes.NewReader(fileBytes))
	if errors.Is(err, preprocessor.ErrNoThumbnail) {
		return nil, "", nil, ErrNoThumbnail
	}
	if err != nil || img == nil {
		logger.Debug().Err(err).Msg("could not convert file to image")
		return nil, "", nil, fmt.Errorf("%w: could not get image", ErrNotFound)
	}

	return bytes.NewReader(img.Data), img.ContentType, func() {}, nil
}

// supportsMimeType reports whether a preview is attempted for the type: the
// announced source types plus whatever a configured Tika takes.
func (w *ThumbnailWorkflow) supportsMimeType(mimeType string) bool {
	return w.sourceTypes.Supports(mimeType) || w.tika.Supports(mimeType)
}

// outputFor is the type a thumbnail of this source is written in and the name
// the generator's format filter has for it. A type nobody mapped, and one whose
// target the generator cannot write, becomes a JPEG.
func (w *ThumbnailWorkflow) outputFor(mimeType string) (outputType, outputFormat string) {
	outputType, outputFormat = "image/jpeg", "jpg"
	if target := w.typeMapping.TargetFor(mimeType); target != "" {
		if format := w.formats.For(target); format != "" {
			outputType, outputFormat = target, format
		}
	}

	return outputType, outputFormat
}

// matchOperation picks how to fit within the box webdav already snapped. A
// processor the client named wins, "thumbnail" and "fit" being legacy aliases for
// fill and fit-in; without one, an unconstrained axis, then the source type
// decide, and "scalingup" is ignored throughout. Only fit-in never enlarges its
// source, so it is the one operation that suppresses upscaling.
func (w *ThumbnailWorkflow) matchOperation(tr *requests.ThumbnailRequest, mimeType string) (operation string, noUpscale bool) {
	proc := strings.ToLower(strings.TrimSpace(tr.Processor))

	switch proc {
	case "resize":
		operation = generator.OpStretch
	case "fill", "thumbnail":
		operation = generator.OpFill
	case "fit", "fit-in":
		operation = generator.OpFitIn
	default: // no processor: a free axis, then the source type decide
		switch {
		case tr.Width == 0 || tr.Height == 0:
			operation = generator.OpFitIn
		case strings.HasPrefix(mimeType, "image/gif"):
			operation = generator.OpStretch
		default:
			operation = generator.OpFill
		}
	}

	noUpscale = operation == generator.OpFitIn

	return operation, noUpscale
}

// quoteEscaper matches the escaping mime/multipart applies to part filenames.
var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

// postToGenerator streams the source image to the generator as a multipart
// POST. The body is produced by an io.Pipe so the (potentially large) image is
// never fully buffered in webdav: the writer goroutine copies the reader into
// the form part on demand. No Content-Length is set, so the request uses chunked
// transfer-encoding; both imagor and the thumbnailer parse the multipart body
// without needing it (imagor enforces its size limit after reading the part).
func (w *ThumbnailWorkflow) postToGenerator(ctx context.Context, url string, img io.Reader, filename, contentType string) ([]byte, string, error) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		// Not CreateFormFile: it types the part application/octet-stream, which
		// imagor's upload loader rejects (UPLOAD_LOADER_ACCEPT defaults to image/*).
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition",
			`form-data; name="image"; filename="`+quoteEscaper.Replace(filename)+`"`)
		h.Set("Content-Type", contentType)
		part, err := writer.CreatePart(h)
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
		return nil, "", fmt.Errorf("create generator request: %w", err)
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
		return nil, "", fmt.Errorf("generator request: %w", err)
	}
	defer httpRsp.Body.Close()

	// The status is read before the writer's error. A generator that refuses the
	// request without reading it, which is what 429 and 413 do, breaks the write
	// half way, and that broken pipe must not mask the answer the client needs.
	if httpRsp.StatusCode < 200 || httpRsp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(httpRsp.Body)
		// The generator reports ErrMaxResolutionExceeded (422) when the decoded
		// input exceeds its configured max width or height. The DAV layer answers
		// the too-large error with 403.
		if httpRsp.StatusCode == http.StatusUnprocessableEntity {
			return nil, "", ErrImageTooLarge
		}
		if httpRsp.StatusCode == http.StatusTooManyRequests {
			retryAfter, _ := strconv.Atoi(httpRsp.Header.Get("Retry-After"))
			return nil, "", &GeneratorBusyError{RetryAfterSeconds: retryAfter}
		}
		// The generator bounds the whole request body, this service only the file,
		// so the multipart framing can push a file just under the limit over it.
		// Either way the source is too large, which is a 403, not a server fault.
		if httpRsp.StatusCode == http.StatusRequestEntityTooLarge {
			return nil, "", ErrImageTooLarge
		}

		// A source the generator cannot read is a missing preview, not a server
		// fault, so the client gets a 404 and stops asking. A 400 means this
		// service built the request wrong and must not read as one.
		if httpRsp.StatusCode == http.StatusNotAcceptable {
			return nil, "", fmt.Errorf("%w: generator refused the source: %s", ErrUnsupportedFileType, strings.TrimSpace(string(respBody)))
		}
		return nil, "", fmt.Errorf("generator returned status %d: %s", httpRsp.StatusCode, string(respBody))
	}

	if writeErr := <-errCh; writeErr != nil {
		io.Copy(io.Discard, httpRsp.Body)
		return nil, "", writeErr
	}

	rspData, err := io.ReadAll(httpRsp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read generator response: %w", err)
	}

	return rspData, httpRsp.Header.Get("Content-Type"), nil
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

func (g *gatewayUserResolver) GetUserByClaim(ctx context.Context, claim, value, auth string) (*userv1beta1.User, error) {
	client, err := g.selector.Next()
	if err != nil {
		return nil, fmt.Errorf("get gateway client: %w", err)
	}

	ctx = grpcmetadata.AppendToOutgoingContext(ctx, revactx.TokenHeader, auth)

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

	switch rsp.GetStatus().GetCode() {
	case rpcv1beta1.Code_CODE_OK:
		// fallthrough
	case rpcv1beta1.Code_CODE_NOT_FOUND:
		// The resource does not exist or is not visible to the caller (e.g. a
		// file of another user). Surface it as 404, matching the legacy
		// thumbnail service.
		return nil, fmt.Errorf("%w: %s", ErrNotFound, rsp.GetStatus().GetMessage())
	default:
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
		return nil, fmt.Errorf("%w: %q", ErrNoStorageSpace, ref.GetPath())
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
	case rpcv1beta1.Code_CODE_PERMISSION_DENIED, rpcv1beta1.Code_CODE_UNAUTHENTICATED:
		// reva reports a wrong or missing link password as InvalidCredentials,
		// which the gateway passes through as UNAUTHENTICATED.
		return "", fmt.Errorf("%w: %s", ErrPublicLinkPasswordRequired, rsp.GetStatus().GetMessage())
	case rpcv1beta1.Code_CODE_FAILED_PRECONDITION:
		return "", fmt.Errorf("%w: %s", ErrPublicLinkExpired, rsp.GetStatus().GetMessage())
	default:
		return "", fmt.Errorf("public link authentication failed: code=%s message=%s", rsp.GetStatus().GetCode(), rsp.GetStatus().GetMessage())
	}
}
