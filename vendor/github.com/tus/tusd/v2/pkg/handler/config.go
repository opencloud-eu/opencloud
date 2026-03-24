package handler

import (
	"errors"
	"net/url"
	"regexp"
	"time"

	"golang.org/x/exp/slog"
)

// Config provides a way to configure the Handler depending on your needs.
type Config struct {
	// StoreComposer points to the store composer from which the core data store
	// and optional dependencies should be taken. May only be nil if DataStore is
	// set.
	StoreComposer *StoreComposer
	// MaxSize defines how many bytes may be stored in one single upload. If its
	// value is is 0 or smaller no limit will be enforced.
	MaxSize int64
	// BasePath defines the URL path used for handling uploads, e.g. "/files/".
	// If no trailing slash is presented it will be added. You may specify an
	// absolute URL containing a scheme, e.g. "http://tus.io"
	BasePath string
	isAbs    bool
	// EnableExperimentalProtocol controls whether the new resumable upload protocol draft
	// from the IETF's HTTP working group is accepted next to the current tus v1 protocol.
	// See https://datatracker.ietf.org/doc/draft-ietf-httpbis-resumable-upload/
	EnableExperimentalProtocol bool
	// DisableDownload indicates whether the server will refuse downloads of the
	// uploaded file, by not mounting the GET handler.
	DisableDownload bool
	// DisableTermination indicates whether the server will refuse termination
	// requests of the uploaded file, by not mounting the DELETE handler.
	DisableTermination bool
	// DisableConcatenation indicates whether the server will refuse POST requests
	// for creating uploads that use the concatenation extension.
	DisableConcatenation bool
	// Cors can be used to customize the handling of Cross-Origin Resource Sharing (CORS).
	// See the CorsConfig struct for more details.
	// Defaults to DefaultCorsConfig.
	Cors *CorsConfig
	// NotifyCompleteUploads indicates whether sending notifications about
	// completed uploads using the CompleteUploads channel should be enabled.
	NotifyCompleteUploads bool
	// NotifyTerminatedUploads indicates whether sending notifications about
	// terminated uploads using the TerminatedUploads channel should be enabled.
	NotifyTerminatedUploads bool
	// NotifyUploadProgress indicates whether sending notifications about
	// the upload progress using the UploadProgress channel should be enabled.
	NotifyUploadProgress bool
	// NotifyCreatedUploads indicates whether sending notifications about
	// the upload having been created using the CreatedUploads channel should be enabled.
	NotifyCreatedUploads bool
	// UploadProgressInterval specifies the interval at which the upload progress
	// notifications are sent to the UploadProgress channel, if enabled.
	// Defaults to 1s.
	UploadProgressInterval time.Duration
	// Logger is the logger to use internally, mostly for printing requests.
	Logger *slog.Logger
	// Respect the X-Forwarded-Host, X-Forwarded-Proto and Forwarded headers
	// potentially set by proxies when generating an absolute URL in the
	// response to POST requests.
	RespectForwardedHeaders bool
	// PreUploadCreateCallback will be invoked before a new upload is created, if the
	// property is supplied. If the callback returns no error, the upload will be created
	// and optional values from HTTPResponse will be contained in the HTTP response.
	// If the error is non-nil, the upload will not be created. This can be used to implement
	// validation of upload metadata etc. Furthermore, HTTPResponse will be ignored and
	// the error value can contain values for the HTTP response.
	// If the error is nil, FileInfoChanges can be filled out to specify individual properties
	// that should be overwriten before the upload is create. See its type definition for
	// more details on its behavior. If you do not want to make any changes, return an empty struct.
	PreUploadCreateCallback func(hook HookEvent) (HTTPResponse, FileInfoChanges, error)
	// PreFinishResponseCallback will be invoked after an upload is completed but before
	// a response is returned to the client. This can be used to implement post-processing validation.
	// If the callback returns no error, optional values from HTTPResponse will be contained in the HTTP response.
	// If the error is non-nil, the error will be forwarded to the client. Furthermore,
	// HTTPResponse will be ignored and the error value can contain values for the HTTP response.
	PreFinishResponseCallback func(hook HookEvent) (HTTPResponse, error)
	// PreUploadTerminateCallback will be invoked on DELETE requests before an upload is terminated,
	// giving the application the opportunity to reject the termination. For example, to ensure resources
	// used by other services are not deleted.
	// If the callback returns no error, optional values from HTTPResponse will be contained in the HTTP response.
	// If the error is non-nil, the error will be forwarded to the client. Furthermore,
	// HTTPResponse will be ignored and the error value can contain values for the HTTP response.
	PreUploadTerminateCallback func(hook HookEvent) (HTTPResponse, error)
	// GracefulRequestCompletionTimeout is the timeout for operations to complete after an HTTP
	// request has ended (successfully or by error). For example, if an HTTP request is interrupted,
	// instead of stopping immediately, the handler and data store will be given some additional
	// time to wrap up their operations and save any uploaded data. GracefulRequestCompletionTimeout
	// controls this time.
	// See HookEvent.Context for more details.
	// Defaults to 10s.
	GracefulRequestCompletionTimeout time.Duration
	// AcquireLockTimeout is the duration that a request handler will wait to acquire a lock for
	// an upload. If the timeout is reached, it will stop waiting and send an error response to the
	// client.
	// Defaults to 20s.
	AcquireLockTimeout time.Duration
	// NetworkTimeout is the timeout for individual read operations on the request body. If the
	// read operation succeeds in this time window, the handler will continue consuming the body.
	// If a read operation times out, the handler will stop reading and close the request.
	// This ensures that an upload is consumed while data is being transmitted, while also closing
	// dead connections.
	// Under the hood, this is passed to ResponseController.SetReadDeadline
	// Defaults to 60s
	NetworkTimeout time.Duration
}

// CorsConfig provides a way to customize the the handling of Cross-Origin Resource Sharing (CORS).
// More details about CORS are available at https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS.
type CorsConfig struct {
	// Disable instructs the handler to ignore all CORS-related headers and never set a
	// CORS-related header in a response. This is useful if CORS is already handled by a proxy.
	Disable bool
	// AllowOrigin is a regular expression used to check if a request is allowed to participate in the
	// CORS protocol. If the request's Origin header matches the regular expression, CORS is allowed.
	// If not, a 403 Forbidden response is sent, rejecting the CORS request.
	AllowOrigin *regexp.Regexp
	// AllowCredentials defines whether the `Access-Control-Allow-Credentials: true` header should be
	// included in CORS responses. This allows clients to share credentials using the Cookie and
	// Authorization header
	AllowCredentials bool
	// AllowMethods defines the value for the `Access-Control-Allow-Methods` header in the response to
	// preflight requests. You can add custom methods here, but make sure that all tus-specific methods
	// from DefaultConfig.AllowMethods are included as well.
	AllowMethods string
	// AllowHeaders defines the value for the `Access-Control-Allow-Headers` header in the response to
	// preflight requests. You can add custom headers here, but make sure that all tus-specific header
	// from DefaultConfig.AllowHeaders are included as well.
	AllowHeaders string
	// MaxAge defines the value for the `Access-Control-Max-Age` header in the response to preflight
	// requests.
	MaxAge string
	// ExposeHeaders defines the value for the `Access-Control-Expose-Headers` header in the response to
	// actual requests. You can add custom headers here, but make sure that all tus-specific header
	// from DefaultConfig.ExposeHeaders are included as well.
	ExposeHeaders string
}

// DefaultCorsConfig is the configuration that will be used in none is provided.
var DefaultCorsConfig = CorsConfig{
	Disable:          false,
	AllowOrigin:      regexp.MustCompile(".*"),
	AllowCredentials: false,
	AllowMethods:     "POST, HEAD, PATCH, OPTIONS, GET, DELETE",
	AllowHeaders:     "Authorization, Origin, X-Requested-With, X-Request-ID, X-HTTP-Method-Override, Content-Type, Upload-Length, Upload-Offset, Tus-Resumable, Upload-Metadata, Upload-Defer-Length, Upload-Concat, Upload-Incomplete, Upload-Complete, Upload-Draft-Interop-Version",
	MaxAge:           "86400",
	ExposeHeaders:    "Upload-Offset, Location, Upload-Length, Tus-Version, Tus-Resumable, Tus-Max-Size, Tus-Extension, Upload-Metadata, Upload-Defer-Length, Upload-Concat, Upload-Incomplete, Upload-Complete, Upload-Draft-Interop-Version",
}

func (config *Config) validate() error {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	base := config.BasePath
	uri, err := url.Parse(base)
	if err != nil {
		return err
	}

	// Ensure base path ends with slash to remove logic from absFileURL
	if base != "" && string(base[len(base)-1]) != "/" {
		base += "/"
	}

	// Ensure base path begins with slash if not absolute (starts with scheme)
	if !uri.IsAbs() && len(base) > 0 && string(base[0]) != "/" {
		base = "/" + base
	}
	config.BasePath = base
	config.isAbs = uri.IsAbs()

	if config.StoreComposer == nil {
		return errors.New("tusd: StoreComposer must no be nil")
	}

	if config.StoreComposer.Core == nil {
		return errors.New("tusd: StoreComposer in Config needs to contain a non-nil core")
	}

	if config.UploadProgressInterval <= 0 {
		config.UploadProgressInterval = 1 * time.Second
	}

	if config.GracefulRequestCompletionTimeout <= 0 {
		config.GracefulRequestCompletionTimeout = 10 * time.Second
	}

	if config.AcquireLockTimeout <= 0 {
		config.AcquireLockTimeout = 20 * time.Second
	}

	if config.NetworkTimeout <= 0 {
		config.NetworkTimeout = 60 * time.Second
	}

	if config.Cors == nil {
		config.Cors = &DefaultCorsConfig
	}

	return nil
}
