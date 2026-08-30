package svc

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/riandyrn/otelchi"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/config"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/constants"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/dav/requests"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/generator"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail/cache"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail/workflow"
)

var (
	codesEnum = map[int]string{
		http.StatusBadRequest:       "Sabre\\DAV\\Exception\\BadRequest",
		http.StatusUnauthorized:     "Sabre\\DAV\\Exception\\NotAuthenticated",
		http.StatusNotFound:         "Sabre\\DAV\\Exception\\NotFound",
		http.StatusMethodNotAllowed: "Sabre\\DAV\\Exception\\MethodNotAllowed",
	}
)

// register the REPORT method at init so it cannot race with concurrent route setup in other services.
func init() {
	chi.RegisterMethod("REPORT")
}

// Service defines the extension handlers.
type Service interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	Thumbnail(w http.ResponseWriter, r *http.Request)
}

// NewService returns a service implementation for Service.
func NewService(opts ...Option) (Service, error) {
	options := newOptions(opts...)
	conf := options.Config

	m := chi.NewMux()
	m.Use(
		otelchi.Middleware(
			conf.Service.Name,
			otelchi.WithChiRoutes(m),
			otelchi.WithTracerProvider(options.TraceProvider),
			otelchi.WithPropagators(tracing.GetPropagator()),
		),
	)

	tm, err := pool.StringToTLSMode(conf.GRPCClientTLS.Mode)
	if err != nil {
		return nil, err
	}
	gatewaySelector, err := pool.GatewaySelector(conf.RevaGateway,
		pool.WithTLSCACert(conf.GRPCClientTLS.CACert),
		pool.WithTLSMode(tm),
		pool.WithRegistry(registry.GetRegistry()),
		pool.WithTracerProvider(options.TraceProvider),
	)
	if err != nil {
		return nil, err
	}

	genTimeout, _ := time.ParseDuration(conf.ThumbnailGeneratorTimeout)
	if genTimeout == 0 {
		genTimeout = 30 * time.Second
	}

	resolutions, err := thumbnail.ParseResolutions(conf.ThumbnailResolutions)
	if err != nil {
		return nil, fmt.Errorf("parse thumbnail resolutions: %w", err)
	}

	maxInputSize, err := generator.ParseMaxInputFileSize(conf.MaxInputFileSize)
	if err != nil {
		return nil, fmt.Errorf("parse max input file size: %w", err)
	}

	s3cfg := cache.BuildS3CacheConfig(
		conf.ThumbnailCacheS3Bucket,
		conf.ThumbnailCacheS3Region,
		conf.ThumbnailCacheS3Endpoint,
		conf.ThumbnailCacheS3AccessKey,
		conf.ThumbnailCacheS3SecretKey,
	)

	c := cache.NewThumbnailCache(conf.ThumbnailCacheBackend, conf.ThumbnailCacheDir, s3cfg)

	httpClient := &http.Client{Timeout: genTimeout}

	wf, err := workflow.NewWorkflow(
		workflow.WithGeneratorURL(conf.ThumbnailGeneratorURL),
		workflow.WithCache(c),
		workflow.WithHTTPClient(httpClient),
		workflow.WithMaxInputSize(maxInputSize),
		workflow.WithResolutions(resolutions),
		workflow.WithWebdavNamespace(conf.WebdavNamespace),
		workflow.WithFontMapFile(conf.FontMapFile),
		workflow.WithTika(thumbnail.NewTika(conf.TikaURL, conf.TikaThumbnailMimeTypes)),
		workflow.WithLogger(options.Logger),
		workflow.WithStater(workflow.NewGatewayStater(gatewaySelector)),
		workflow.WithFileDownloader(workflow.NewGatewayFileDownloader(gatewaySelector, httpClient)),
		workflow.WithSpaceLookup(workflow.NewGatewaySpaceLookup(gatewaySelector)),
		workflow.WithUserResolver(workflow.NewGatewayUserResolver(gatewaySelector)),
	)
	if err != nil {
		return nil, fmt.Errorf("create thumbnail workflow: %w", err)
	}

	svc := Webdav{
		config:          conf,
		log:             options.Logger,
		mux:             m,
		searchClient:    searchsvc.NewSearchProviderService("eu.opencloud.api.search", conf.GrpcClient),
		workflow:        wf,
		gatewaySelector: gatewaySelector,
	}

	m.Route(options.Config.HTTP.Root, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(svc.DavUserContext())

			r.Get("/remote.php/dav/spaces/{id}", svc.SpacesThumbnail)
			r.Get("/remote.php/dav/spaces/{id}/*", svc.SpacesThumbnail)
			r.Get("/dav/spaces/{id}", svc.SpacesThumbnail)
			r.Get("/dav/spaces/{id}/*", svc.SpacesThumbnail)
			r.MethodFunc("REPORT", "/remote.php/dav/spaces*", svc.Search)
			r.MethodFunc("REPORT", "/dav/spaces*", svc.Search)

			r.Get("/remote.php/dav/files/{id}", svc.Thumbnail)
			r.Get("/remote.php/dav/files/{id}/*", svc.Thumbnail)
			r.Get("/dav/files/{id}", svc.Thumbnail)
			r.Get("/dav/files/{id}/*", svc.Thumbnail)

			r.MethodFunc("REPORT", "/remote.php/dav/files*", svc.Search)
			r.MethodFunc("REPORT", "/dav/files*", svc.Search)
		})

		r.Group(func(r chi.Router) {
			r.Use(svc.DavPublicContext())

			r.Head("/remote.php/dav/public-files/{token}/*", svc.PublicThumbnailHead)
			r.Head("/dav/public-files/{token}/*", svc.PublicThumbnailHead)

			r.Get("/remote.php/dav/public-files/{token}/*", svc.PublicThumbnail)
			r.Get("/dav/public-files/{token}/*", svc.PublicThumbnail)
		})

		r.Group(func(r chi.Router) {
			r.Use(svc.WebDAVContext())
			r.Get("/remote.php/webdav/*", svc.Thumbnail)
			r.Get("/webdav/*", svc.Thumbnail)

			r.MethodFunc("REPORT", "/remote.php/webdav*", svc.Search)
			r.MethodFunc("REPORT", "/webdav*", svc.Search)
		})
	})

	_ = chi.Walk(m, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		options.Logger.Debug().Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})

	return svc, nil
}

// Webdav implements the business logic for Service.
type Webdav struct {
	config          *config.Config
	log             log.Logger
	mux             *chi.Mux
	searchClient    searchsvc.SearchProviderService
	workflow        *workflow.ThumbnailWorkflow
	gatewaySelector pool.Selectable[gatewayv1beta1.GatewayAPIClient]
}

// ServeHTTP implements the Service interface.
func (g Webdav) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

func (g Webdav) DavUserContext() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			filePath := r.URL.Path

			id := chi.URLParam(r, "id")
			id, err := url.QueryUnescape(id)
			if err == nil && id != "" {
				ctx = context.WithValue(ctx, constants.ContextKeyID, id)
			}

			if id != "" {
				filePath = strings.TrimPrefix(filePath, path.Join("/remote.php/dav/spaces", id))
				filePath = strings.TrimPrefix(filePath, path.Join("/dav/spaces", id))

				filePath = strings.TrimPrefix(filePath, path.Join("/remote.php/dav/files", id))
				filePath = strings.TrimPrefix(filePath, path.Join("/dav/files", id))
				filePath = strings.TrimPrefix(filePath, "/")
			}

			ctx = context.WithValue(ctx, constants.ContextKeyPath, filePath)

			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}

func (g Webdav) DavPublicContext() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			filePath := r.URL.Path

			if token := chi.URLParam(r, "token"); token != "" {
				filePath = strings.TrimPrefix(filePath, path.Join("/remote.php/dav/public-files", token)+"/")
				filePath = strings.TrimPrefix(filePath, path.Join("/dav/public-files", token)+"/")
			}
			ctx = context.WithValue(ctx, constants.ContextKeyPath, filePath)

			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}
func (g Webdav) WebDAVContext() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			filePath := r.URL.Path
			filePath = strings.TrimPrefix(filePath, "/remote.php")
			filePath = strings.TrimPrefix(filePath, "/webdav/")

			ctx := context.WithValue(r.Context(), constants.ContextKeyPath, filePath)

			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}

// SpacesThumbnail is the endpoint for retrieving thumbnails inside of spaces.
func (g Webdav) SpacesThumbnail(w http.ResponseWriter, r *http.Request) {
	logger := g.log.SubloggerWithRequestID(r.Context())
	tr, err := requests.ParseThumbnailRequest(r)
	if err != nil {
		logger.Debug().Err(err).Msg("could not create Request")
		renderError(w, r, errBadRequest(err.Error()))
		return
	}

	auth := r.Header.Get(revactx.TokenHeader)
	data, ext, aIgnored, err := g.workflow.Execute(r.Context(), tr, auth, logger)
	if err != nil {
		g.handleWorkflowError(w, r, err, tr, logger)
		return
	}
	setAspectIgnoredHeader(w, aIgnored)
	generator.WriteThumbnailResponse(w, data, ext)
}

func whoami(gatewayClient gatewayv1beta1.GatewayAPIClient, ctx context.Context, token string) (*userv1beta1.User, error) {
	userRes, err := gatewayClient.WhoAmI(ctx, &gatewayv1beta1.WhoAmIRequest{
		Token: token,
	})
	if err != nil {
		return nil, err
	}
	if userRes.Status.Code != rpcv1beta1.Code_CODE_OK {
		return nil, fmt.Errorf("could not get user: %s", userRes.GetStatus().GetMessage())
	}
	return userRes.GetUser(), nil
}

// Thumbnail implements the Service interface.
func (g Webdav) Thumbnail(w http.ResponseWriter, r *http.Request) {
	logger := g.log.SubloggerWithRequestID(r.Context())
	tr, err := requests.ParseThumbnailRequest(r)
	if err != nil {
		logger.Debug().Err(err).Msg("could not create Request")
		renderError(w, r, errBadRequest(err.Error()))
		return
	}

	auth := r.Header.Get(revactx.TokenHeader)
	data, ext, aIgnored, err := g.workflow.Execute(r.Context(), tr, auth, logger)
	if err != nil {
		g.handleWorkflowError(w, r, err, tr, logger)
		return
	}
	setAspectIgnoredHeader(w, aIgnored)
	generator.WriteThumbnailResponse(w, data, ext)
}

func (g Webdav) PublicThumbnail(w http.ResponseWriter, r *http.Request) {
	logger := g.log.SubloggerWithRequestID(r.Context())
	tr, err := requests.ParseThumbnailRequest(r)
	if err != nil {
		logger.Debug().Err(err).Msg("could not create Request")
		renderError(w, r, errBadRequest(err.Error()))
		return
	}

	auth, err := workflow.ResolvePublicLinkAuth(r.Context(), r, chi.URLParam(r, "token"), g.gatewaySelector)
	if err != nil {
		g.handlePublicLinkAuthError(w, r, err, tr.Filename, logger)
		return
	}

	data, ext, aIgnored, err := g.workflow.ExecutePublic(r.Context(), tr, auth, logger)
	if err != nil {
		g.handleWorkflowError(w, r, err, tr, logger)
		return
	}
	setAspectIgnoredHeader(w, aIgnored)
	generator.WriteThumbnailResponse(w, data, ext)
}

func (g Webdav) PublicThumbnailHead(w http.ResponseWriter, r *http.Request) {
	logger := g.log.SubloggerWithRequestID(r.Context())
	tr, err := requests.ParseThumbnailRequest(r)
	if err != nil {
		logger.Debug().Err(err).Msg("could not create Request")
		renderError(w, r, errBadRequest(err.Error()))
		return
	}

	auth, err := workflow.ResolvePublicLinkAuth(r.Context(), r, chi.URLParam(r, "token"), g.gatewaySelector)
	if err != nil {
		g.handlePublicLinkAuthError(w, r, err, tr.Filename, logger)
		return
	}

	if err := g.workflow.Head(r.Context(), tr, auth, logger); err != nil {
		g.handleHeadError(w, r, err, tr, logger)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// aspectIgnoredHeader is set on thumbnail responses when an explicit processor
// overrode the legacy "a" flag with a different behavior, so developers can tell
// their client to send a consistent request (see the thumbnail docs).
const aspectIgnoredHeader = "X-OpenCloud-Thumbnail-Aspect-Ignored"

func setAspectIgnoredHeader(w http.ResponseWriter, ignored bool) {
	if !ignored {
		return
	}
	w.Header().Set(aspectIgnoredHeader, "1; an explicit processor overrode the legacy 'a' flag, see the thumbnail docs")
}

func (g Webdav) handleWorkflowError(w http.ResponseWriter, r *http.Request, err error, tr *requests.ThumbnailRequest, logger log.Logger) {
	if errors.Is(err, workflow.ErrFileProcessing) {
		addRetryAfterHeader(w)
		renderError(w, r, errTooEarly("file is being processed"))
		return
	}
	if errors.Is(err, workflow.ErrImageTooLarge) {
		logger.Debug().Err(err).Msg("thumbnail input image too large")
		renderError(w, r, errPermissionDenied(workflow.ErrImageTooLarge.Error()))
		return
	}
	if errors.Is(err, workflow.ErrPermissionDenied) {
		logger.Debug().Err(err).Msg("user lacks download permission for thumbnail")
		renderError(w, r, errPermissionDenied(workflow.ErrPermissionDenied.Error()))
		return
	}
	if errors.Is(err, workflow.ErrNotAFile) {
		logger.Debug().Err(err).Msg("thumbnail requested for a non-file resource")
		renderError(w, r, errBadRequest("Unsupported file type"))
		return
	}
	if errors.Is(err, workflow.ErrNotFound) {
		logger.Debug().Err(err).Msg("thumbnail source could not be located")
		renderError(w, r, errNotFound(notFoundMsg(tr.Filename)))
		return
	}
	// Anything else (generator down, download failure, timeout, ...) is a server
	// error: clients must not cache it as "no preview" the way they do 404s.
	logger.Error().Err(err).Msg("thumbnail workflow failed")
	renderError(w, r, errInternalError("could not generate thumbnail"))
}

func (g Webdav) handleHeadError(w http.ResponseWriter, r *http.Request, err error, tr *requests.ThumbnailRequest, logger log.Logger) {
	if errors.Is(err, workflow.ErrFileProcessing) {
		addRetryAfterHeader(w)
		renderError(w, r, errTooEarly("file is being processed"))
		return
	}
	if errors.Is(err, workflow.ErrImageTooLarge) {
		logger.Debug().Err(err).Msg("thumbnail input image too large")
		renderError(w, r, errPermissionDenied(workflow.ErrImageTooLarge.Error()))
		return
	}
	if errors.Is(err, workflow.ErrPermissionDenied) {
		logger.Debug().Err(err).Msg("user lacks download permission for thumbnail")
		renderError(w, r, errPermissionDenied(workflow.ErrPermissionDenied.Error()))
		return
	}
	if errors.Is(err, workflow.ErrNotAFile) {
		logger.Debug().Err(err).Msg("thumbnail head check requested for a non-file resource")
		renderError(w, r, errBadRequest("Unsupported file type"))
		return
	}
	if errors.Is(err, workflow.ErrNotFound) {
		logger.Debug().Err(err).Msg("thumbnail source could not be located")
		renderError(w, r, errNotFound(notFoundMsg(tr.Filename)))
		return
	}
	logger.Error().Err(err).Msg("thumbnail head check failed")
	renderError(w, r, errInternalError("could not check thumbnail"))
}

func (g Webdav) handlePublicLinkAuthError(w http.ResponseWriter, r *http.Request, err error, filename string, logger log.Logger) {
	if errors.Is(err, workflow.ErrPublicLinkPasswordRequired) {
		// A password-protected public link accessed without (or with the wrong)
		// password must not reveal that the resource exists, so it is hidden
		// behind a 404 rather than a 403.
		logger.Debug().Err(err).Msg("public link requires a password")
		renderError(w, r, errNotFound(notFoundMsg(filename)))
		return
	}
	if errors.Is(err, workflow.ErrPublicLinkExpired) {
		logger.Debug().Err(err).Msg("public link has expired")
		renderError(w, r, newErrResponse(http.StatusGone, "public link has expired"))
		return
	}
	logger.Error().Err(err).Msg("could not authenticate public link")
	renderError(w, r, errInternalError("could not authenticate public link"))
}

// http://www.webdav.org/specs/rfc4918.html#ELEMENT_error
type errResponse struct {
	HTTPStatusCode int      `json:"-" xml:"-"`
	XMLName        xml.Name `xml:"d:error"`
	Xmlnsd         string   `xml:"xmlns:d,attr"`
	Xmlnss         string   `xml:"xmlns:s,attr"`
	Exception      string   `xml:"s:exception"`
	Message        string   `xml:"s:message"`
	InnerXML       []byte   `xml:",innerxml"`
}

func newErrResponse(statusCode int, msg string) *errResponse {
	rsp := &errResponse{
		HTTPStatusCode: statusCode,
		Xmlnsd:         "DAV",
		Xmlnss:         "http://sabredav.org/ns",
		Exception:      codesEnum[statusCode],
	}
	if msg != "" {
		rsp.Message = msg
	}
	return rsp
}

func errInternalError(msg string) *errResponse {
	return newErrResponse(http.StatusInternalServerError, msg)
}

func errBadRequest(msg string) *errResponse {
	return newErrResponse(http.StatusBadRequest, msg)
}

func errPermissionDenied(msg string) *errResponse {
	return newErrResponse(http.StatusForbidden, msg)
}

func errNotFound(msg string) *errResponse {
	return newErrResponse(http.StatusNotFound, msg)
}

func errTooEarly(msg string) *errResponse {
	return newErrResponse(http.StatusTooEarly, msg)
}

func addRetryAfterHeader(w http.ResponseWriter) {
	after := rand.IntN(14) + 1
	w.Header().Set("Retry-After", strconv.Itoa(after))
}

func renderError(w http.ResponseWriter, r *http.Request, err *errResponse) {
	render.Status(r, err.HTTPStatusCode)
	render.XML(w, r, err)
}

func notFoundMsg(name string) string {
	return "File with name " + name + " could not be located"
}
