package jmap

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/version"
)

// Implementation of ApiClient, SessionClient and BlobClient that uses
// HTTP to perform JMAP operations.
type HttpJmapClient struct {
	client                   *http.Client
	userAgent                string
	authenticator            HttpJmapClientAuthenticator
	listener                 HttpJmapApiClientEventListener
	traceRequests            bool
	traceMaxRequestBodySize  int64
	traceResponses           bool
	traceMaxResponseBodySize int64
}

var (
	_ ApiClient     = &HttpJmapClient{}
	_ SessionClient = &HttpJmapClient{}
	_ BlobClient    = &HttpJmapClient{}
)

const (
	logEndpoint       = "endpoint"
	logUri            = "uri"
	logMethod         = "method"
	logHttpStatus     = "status"
	logHttpStatusCode = "status-code"
	logHttpUrl        = "url"
	logProto          = "proto"
	logProtoJmap      = "jmap"
	logProtoJmapWs    = "jmapws"
	logType           = "type"
	logTypeRequest    = "request"
	logTypeResponse   = "response"
	logTypePush       = "push"
	logDuration       = "duration"
	logBodyTruncated  = "truncated"
)

// Record JMAP HTTP execution events that may occur, e.g. using metrics.
type HttpJmapApiClientEventListener interface {
	OnSuccessfulRequest(endpoint string, status int)
	OnFailedRequest(endpoint string, err error)
	OnFailedRequestWithStatus(endpoint string, status int)
	OnResponseBodyReadingError(endpoint string, err error)
	OnResponseBodyUnmarshallingError(endpoint string, err error)
	OnSuccessfulWsRequest(endpoint string, status int)
	OnFailedWsHandshakeRequestWithStatus(endpoint string, status int)
}

type nullHttpJmapApiClientEventListener struct {
}

func (l nullHttpJmapApiClientEventListener) OnSuccessfulRequest(endpoint string, status int) {
	// null implementation does nothing
}
func (l nullHttpJmapApiClientEventListener) OnFailedRequest(endpoint string, err error) {
	// null implementation does nothing
}
func (l nullHttpJmapApiClientEventListener) OnFailedRequestWithStatus(endpoint string, status int) {
	// null implementation does nothing
}
func (l nullHttpJmapApiClientEventListener) OnResponseBodyReadingError(endpoint string, err error) {
	// null implementation does nothing
}
func (l nullHttpJmapApiClientEventListener) OnResponseBodyUnmarshallingError(endpoint string, err error) {
	// null implementation does nothing
}
func (l nullHttpJmapApiClientEventListener) OnSuccessfulWsRequest(endpoint string, status int) {
	// null implementation does nothing
}
func (l nullHttpJmapApiClientEventListener) OnFailedWsHandshakeRequestWithStatus(endpoint string, status int) {
	// null implementation does nothing
}

var _ HttpJmapApiClientEventListener = nullHttpJmapApiClientEventListener{}

type HttpJmapClientAuthenticator interface {
	Authenticate(ctx context.Context, username string, logger *log.Logger, req *http.Request) Error
	AuthenticateWS(ctx context.Context, username string, logger *log.Logger, headers http.Header) Error
}

type MasterAuthHttpJmapClientAuthenticator struct {
	masterUser     string
	masterPassword string
}

func NewMasterAuthHttpJmapClientAuthenticator(masterUser string, masterPassword string) HttpJmapClientAuthenticator {
	return &MasterAuthHttpJmapClientAuthenticator{masterUser: masterUser, masterPassword: masterPassword}
}

var _ HttpJmapClientAuthenticator = &MasterAuthHttpJmapClientAuthenticator{}

func (h *MasterAuthHttpJmapClientAuthenticator) auth(username string, headers http.Header) {
	// not nice to read, but heavily optimized to prevent needless memory allocations, since
	// this hot path will be used all the time
	var sb strings.Builder
	sb.WriteString("Basic ")
	enc := base64.NewEncoder(base64.StdEncoding, &sb)
	enc.Write([]byte(username))
	if username != h.masterUser {
		enc.Write([]byte("%"))
		enc.Write([]byte(h.masterUser))
	}
	enc.Write([]byte(":"))
	enc.Write([]byte(h.masterPassword))
	enc.Close()
	headers.Set("Authorization", sb.String())
}

func (h *MasterAuthHttpJmapClientAuthenticator) Authenticate(_ context.Context, username string, _ *log.Logger, req *http.Request) Error {
	h.auth(username, req.Header)
	return nil
}

func (h *MasterAuthHttpJmapClientAuthenticator) AuthenticateWS(_ context.Context, username string, _ *log.Logger, headers http.Header) Error {
	h.auth(username, headers)
	return nil
}

// An implementation of HttpJmapApiClientMetricsRecorder that does nothing.
func NullHttpJmapApiClientEventListener() HttpJmapApiClientEventListener {
	return nullHttpJmapApiClientEventListener{}
}

func NewHttpJmapClient(client *http.Client, authenticator HttpJmapClientAuthenticator, listener HttpJmapApiClientEventListener,
	traceRequests bool, traceMaxRequestBodySize int64,
	traceResponses bool, traceMaxResponseBodySize int64,
) *HttpJmapClient {
	return &HttpJmapClient{
		client:                   client,
		authenticator:            authenticator,
		userAgent:                "OpenCloud/" + version.GetString(),
		listener:                 listener,
		traceRequests:            traceRequests,
		traceMaxRequestBodySize:  traceMaxRequestBodySize,
		traceResponses:           traceResponses,
		traceMaxResponseBodySize: traceMaxResponseBodySize,
	}
}

func (h *HttpJmapClient) Close() error {
	h.client.CloseIdleConnections()
	return nil
}

type AuthenticationError struct {
	Err error
}

func (e AuthenticationError) Error() string {
	return fmt.Sprintf("failed to find user for authentication: %v", e.Err.Error())
}
func (e AuthenticationError) Unwrap() error {
	return e.Err
}

func (h *HttpJmapClient) auth(ctx context.Context, username string, logger *log.Logger, req *http.Request) Error {
	return h.authenticator.Authenticate(ctx, username, logger, req)
}

var (
	errNilBaseUrl = errors.New("sessionUrl is nil")
)

func (h *HttpJmapClient) beforeRequest(_ context.Context, logger *log.Logger, _ string, endpoint string, req *http.Request) Error {
	l := logger.Trace()
	if h.traceRequests && l.Enabled() {
		var logBuilder bytes.Buffer
		for key, values := range req.Header {
			if key == "Authorization" {
				continue
			}
			for _, value := range values {
				fmt.Fprintf(&logBuilder, "%s: %s\n", key, value)
			}
		}
		if h.traceMaxRequestBodySize >= 0 && req.Body != nil && req.Body != http.NoBody {
			peekBuffer := new(bytes.Buffer)
			limitedReader := io.LimitReader(req.Body, h.traceMaxRequestBodySize)
			tee := io.TeeReader(limitedReader, peekBuffer)
			if _, err := io.Copy(io.Discard, tee); err != nil {
				return jmapError(fmt.Errorf("failed to peek at the request body for tracing: %v", err), JmapErrorRequestTracing)
			} else {
				logBuilder.Write(peekBuffer.Bytes())
				if int64(peekBuffer.Len()) >= h.traceMaxRequestBodySize {
					l = l.Bool(logBodyTruncated, true)
				}
				fullBodyReader := io.MultiReader(peekBuffer, req.Body)
				req.Body = io.NopCloser(fullBodyReader)
			}
		}
		l.Str(logMethod, req.Method).Str(logUri, req.URL.String()).
			Str(logEndpoint, endpoint).Str(logProto, logProtoJmap).Str(logType, logTypeRequest).
			Msg(logBuilder.String())
	}
	return nil
}

type httpResponseReadCloser struct {
	io.Reader
	body io.ReadCloser
}

func (c httpResponseReadCloser) Close() error {
	if c.body != nil {
		return c.body.Close()
	}
	return nil
}

func (h *HttpJmapClient) response(_ context.Context, logger *log.Logger, _ string, endpoint string, duration time.Duration, req *http.Request, resp *http.Response) (io.ReadCloser, error) {
	l := logger.Trace()
	if h.traceResponses && l.Enabled() {
		var logBuilder bytes.Buffer
		for key, values := range resp.Header {
			for _, value := range values {
				fmt.Fprintf(&logBuilder, "%s: %s\n", key, value)
			}
		}

		var peekBuffer *bytes.Buffer = nil
		if h.traceMaxResponseBodySize > 0 {
			peekBuffer = new(bytes.Buffer)
			limitedReader := io.LimitReader(resp.Body, h.traceMaxResponseBodySize)
			tee := io.TeeReader(limitedReader, peekBuffer)
			if _, err := io.Copy(io.Discard, tee); err != nil {
				if resp.Body != nil {
					if err := resp.Body.Close(); err != nil {
						logger.Error().Err(err).Msg("failed to close response body") //NOSONAR
					}
				}
				return nil, err
			}

			if int64(peekBuffer.Len()) >= h.traceMaxResponseBodySize {
				l = l.Bool(logBodyTruncated, true)
			}

			logBuilder.Write(peekBuffer.Bytes())
		}

		l.Str(logProto, logProtoJmap).Str(logType, logTypeResponse).Str(logEndpoint, endpoint).
			Str(logMethod, req.Method).Str(logUri, req.URL.String()).
			Str(logHttpStatus, log.SafeString(resp.Status)).Int(logHttpStatusCode, resp.StatusCode).
			Dur(logDuration, duration).
			Msg(logBuilder.String())

		if peekBuffer != nil {
			return httpResponseReadCloser{
				body:   resp.Body,
				Reader: io.MultiReader(peekBuffer, resp.Body),
			}, nil
		} else {
			return resp.Body, nil
		}
	} else {
		return resp.Body, nil
	}
}

func (h *HttpJmapClient) GetSession(ctx context.Context, sessionUrl *url.URL, username string, logger *log.Logger) (SessionResponse, Error) {
	if sessionUrl == nil {
		logger.Error().Msg("sessionUrl is nil")
		return SessionResponse{}, jmapError(errNilBaseUrl, JmapErrorInvalidHttpRequest)
	}
	// See the JMAP specification on Service Autodiscovery: https://jmap.io/spec-core.html#service-autodiscovery
	// There are two standardised autodiscovery methods in use for Internet protocols:
	// - DNS SRV (see [@!RFC2782], [@!RFC6186], and [@!RFC6764])
	// - .well-known/servicename (see [@!RFC8615])
	// We are currently only supporting RFC8615, using the baseurl that was configured in this HttpJmapApiClient.
	//sessionUrl := baseurl.JoinPath(".well-known", "jmap")
	sessionUrlStr := sessionUrl.String()
	endpoint := endpointOf(sessionUrl)
	logger = log.From(logger.With().Str(logEndpoint, endpoint))

	req, err := http.NewRequest(http.MethodGet, sessionUrlStr, nil)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to create GET request for %v", sessionUrl)
		return SessionResponse{}, jmapError(err, JmapErrorInvalidHttpRequest)
	}
	req.Header.Add("Cache-Control", "no-cache, no-store, must-revalidate") // spec recommendation
	req.Header.Add("Content-Type", "application/json")                     //NOSONAR
	req.Header.Add("User-Agent", h.userAgent)                              //NOSONAR

	if err := h.beforeRequest(ctx, logger, username, endpoint, req); err != nil {
		return SessionResponse{}, err
	}

	if err := h.auth(ctx, username, logger, req); err != nil {
		return SessionResponse{}, err
	}

	before := time.Now()
	res, err := h.client.Do(req)
	duration := time.Since(before)
	if err != nil {
		h.listener.OnFailedRequest(endpoint, err)
		logger.Error().Err(err).Msgf("failed to perform GET %v", sessionUrl)
		return SessionResponse{}, jmapError(err, JmapErrorInvalidHttpRequest)
	}

	// dump the response regardless of the status code
	body, err := h.response(ctx, logger, username, endpoint, duration, req, res)

	// since we are not returning a stream from this function, we have to close the response body
	// before leaving the scope of the function
	defer func() {
		if err := body.Close(); err != nil {
			logger.Error().Err(err).Msg("failed to close response body") //NOSONAR
		}
	}()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		h.listener.OnFailedRequestWithStatus(endpoint, res.StatusCode)
		logger.Error().Str(logHttpStatus, log.SafeString(res.Status)).Int(logHttpStatusCode, res.StatusCode).Msg("HTTP response status code is not 200")
		return SessionResponse{}, jmapError(fmt.Errorf("JMAP API response status is %v", res.Status), JmapErrorServerResponse)
	}
	h.listener.OnSuccessfulRequest(endpoint, res.StatusCode)

	if err != nil {
		logger.Error().Err(err).Msg("failed to read response body") //NOSONAR
		h.listener.OnResponseBodyReadingError(endpoint, err)
		return SessionResponse{}, jmapError(err, JmapErrorReadingResponseBody)
	}

	var data SessionResponse
	if err := json.NewDecoder(body).Decode(&data); err != nil {
		logger.Error().Str(logHttpUrl, log.SafeString(sessionUrlStr)).Err(err).Msg("failed to decode JSON payload from .well-known/jmap response")
		h.listener.OnResponseBodyUnmarshallingError(endpoint, err)
		return SessionResponse{}, jmapError(err, JmapErrorDecodingResponseBody)
	}

	return data, nil
}

func (h *HttpJmapClient) Command(request Request, ctx Context) (io.ReadCloser, Language, Error) { //NOSONAR
	session := ctx.Session
	logger := ctx.Logger
	acceptLanguage := ctx.AcceptLanguage
	cotx := ctx.Context

	jmapUrl := session.JmapUrl.String()
	endpoint := session.JmapEndpoint
	logger = log.From(logger.With().Str(logEndpoint, endpoint))

	bodyBytes, err := json.Marshal(request)
	if err != nil {
		logger.Error().Err(err).Msg("failed to marshall JSON payload")
		return nil, "", jmapError(err, JmapErrorEncodingRequestBody)
	}

	req, err := http.NewRequestWithContext(cotx, http.MethodPost, jmapUrl, bytes.NewBuffer(bodyBytes))
	if err != nil {
		logger.Error().Err(err).Msgf("failed to create POST request for %v", jmapUrl)
		return nil, "", jmapError(err, JmapErrorCreatingRequest)
	}

	// Some JMAP APIs use the Accept-Language header to determine which language to use to translate
	// texts in attributes.
	if acceptLanguage != "" {
		req.Header.Add("Accept-Language", acceptLanguage) //NOSONAR
	}

	req.Header.Add("Content-Type", "application/json") //NOSONAR
	req.Header.Add("User-Agent", h.userAgent)          //NOSONAR

	h.beforeRequest(cotx, logger, session.Username, endpoint, req)

	if err := h.auth(cotx, session.Username, logger, req); err != nil {
		return nil, "", err
	}

	before := time.Now()
	res, err := h.client.Do(req)
	duration := time.Since(before)
	if err != nil {
		h.listener.OnFailedRequest(endpoint, err)
		logger.Error().Err(err).Msgf("failed to perform POST %v", jmapUrl)
		return nil, "", jmapError(err, JmapErrorSendingRequest)
	}

	// note that we are omitting the usual deferred response body closer here since we are returning
	// an io.ReadCloser to the body and if we close it when leaving the scope of this function, the
	// caller won't be able to read the response; instead, it is up to the caller to close the
	// ReadCloser we are returning from here

	body, err := h.response(cotx, logger, session.Username, endpoint, duration, req, res)
	language := Language(res.Header.Get("Content-Language")) //NOSONAR
	if err != nil {
		logger.Error().Err(err).Msg("failed to read response body")
		h.listener.OnResponseBodyReadingError(endpoint, err)
		return nil, language, jmapError(err, JmapErrorServerResponse)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		h.listener.OnFailedRequestWithStatus(endpoint, res.StatusCode)
		logger.Error().Str(logEndpoint, endpoint).Str(logHttpStatus, log.SafeString(res.Status)).Msg("HTTP response status code is not 2xx") //NOSONAR
		return nil, language, jmapError(err, JmapErrorServerResponse)
	}

	h.listener.OnSuccessfulRequest(endpoint, res.StatusCode)

	return body, language, nil
}

func (h *HttpJmapClient) UploadBinary(uploadUrl string, endpoint string, contentType string, body io.Reader, ctx Context) (UploadedBlob, Language, Error) { //NOSONAR
	session := ctx.Session
	logger := ctx.Logger
	acceptLanguage := ctx.AcceptLanguage
	cotx := ctx.Context

	logger = log.From(logger.With().Str(logEndpoint, endpoint))

	req, err := http.NewRequestWithContext(cotx, http.MethodPost, uploadUrl, body)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to create POST request for %v", uploadUrl)
		return UploadedBlob{}, "", jmapError(err, JmapErrorCreatingRequest)
	}
	req.Header.Add("Content-Type", contentType)
	req.Header.Add("User-Agent", h.userAgent)
	if acceptLanguage != "" {
		req.Header.Add("Accept-Language", acceptLanguage)
	}
	h.beforeRequest(cotx, logger, session.Username, endpoint, req)

	if err := h.auth(cotx, session.Username, logger, req); err != nil {
		return UploadedBlob{}, "", err
	}

	before := time.Now()
	res, err := h.client.Do(req)
	duration := time.Since(before)
	if err != nil {
		h.listener.OnFailedRequest(endpoint, err)
		logger.Error().Err(err).Msgf("failed to perform POST %v", uploadUrl)
		return UploadedBlob{}, "", jmapError(err, JmapErrorSendingRequest)
	}

	// note that we are omitting the usual deferred response body closer here since we are returning
	// an io.ReadCloser to the body and if we close it when leaving the scope of this function, the
	// caller won't be able to read the response; instead, it is up to the caller to close the
	// ReadCloser we are returning from here

	responseBody, err := h.response(cotx, logger, session.Username, endpoint, duration, req, res)

	// since we are not returning a stream from this function, we have to close the response body
	// before leaving the scope of the function
	defer func() {
		if err := responseBody.Close(); err != nil {
			logger.Error().Err(err).Msg("failed to close response body") //NOSONAR
		}
	}()
	language := Language(res.Header.Get("Content-Language"))
	if err != nil {
		logger.Error().Err(err).Msg("failed to read response body")
		h.listener.OnResponseBodyReadingError(endpoint, err)
		return UploadedBlob{}, language, jmapError(err, JmapErrorServerResponse)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		h.listener.OnFailedRequestWithStatus(endpoint, res.StatusCode)
		logger.Error().Str(logHttpStatus, log.SafeString(res.Status)).Int(logHttpStatusCode, res.StatusCode).Msg("HTTP response status code is not 2xx")
		return UploadedBlob{}, language, jmapError(err, JmapErrorServerResponse)
	}
	h.listener.OnSuccessfulRequest(endpoint, res.StatusCode)

	var result UploadedBlob
	if err := json.NewDecoder(responseBody).Decode(&result); err != nil {
		logger.Error().Str(logHttpUrl, log.SafeString(uploadUrl)).Err(err).Msg("failed to decode JSON payload from the upload response")
		h.listener.OnResponseBodyUnmarshallingError(endpoint, err)
		return UploadedBlob{}, language, jmapError(err, JmapErrorDecodingResponseBody)
	}

	return result, language, nil
}

func (h *HttpJmapClient) DownloadBinary(downloadUrl string, endpoint string, ctx Context) (*BlobDownload, Language, Error) { //NOSONAR
	session := ctx.Session
	logger := ctx.Logger
	acceptLanguage := ctx.AcceptLanguage
	cotx := ctx.Context

	logger = log.From(logger.With().Str(logEndpoint, endpoint))

	req, err := http.NewRequestWithContext(cotx, http.MethodGet, downloadUrl, nil)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to create GET request for %v", downloadUrl)
		return nil, "", jmapError(err, JmapErrorCreatingRequest)
	}
	req.Header.Add("User-Agent", h.userAgent)
	if acceptLanguage != "" {
		req.Header.Add("Accept-Language", acceptLanguage)
	}
	h.beforeRequest(cotx, logger, session.Username, endpoint, req)

	if err := h.auth(cotx, session.Username, logger, req); err != nil {
		return nil, "", err
	}

	before := time.Now()
	res, err := h.client.Do(req)
	duration := time.Since(before)
	if err != nil {
		h.listener.OnFailedRequest(endpoint, err)
		logger.Error().Err(err).Msgf("failed to perform GET %v", downloadUrl)
		return nil, "", jmapError(err, JmapErrorSendingRequest)
	}

	// note that we are omitting the usual deferred response body closer here since we are returning
	// an io.ReadCloser to the body and if we close it when leaving the scope of this function, the
	// caller won't be able to read the response; instead, it is up to the caller to close the
	// ReadCloser we are returning from here

	responseBody, err := h.response(cotx, logger, session.Username, endpoint, duration, req, res)

	language := Language(res.Header.Get("Content-Language"))
	if err != nil {
		logger.Error().Err(err).Msg("failed to read response body")
		h.listener.OnResponseBodyReadingError(endpoint, err)
		return nil, language, jmapError(err, JmapErrorServerResponse)
	}

	if res.StatusCode == http.StatusNotFound {
		return nil, language, nil
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		h.listener.OnFailedRequestWithStatus(endpoint, res.StatusCode)
		logger.Error().Str(logHttpStatus, log.SafeString(res.Status)).Int(logHttpStatusCode, res.StatusCode).Msg("HTTP response status code is not 2xx")
		return nil, language, jmapError(err, JmapErrorServerResponse)
	}
	h.listener.OnSuccessfulRequest(endpoint, res.StatusCode)

	sizeStr := res.Header.Get("Content-Length")
	size := -1
	if sizeStr != "" {
		size, err = strconv.Atoi(sizeStr)
		if err != nil {
			logger.Warn().Err(err).Msgf("failed to parse Content-Length blob download response header value '%v'", sizeStr)
			size = -1
		}
	}

	return &BlobDownload{
		Body:               responseBody,
		Size:               size,
		Type:               res.Header.Get("Content-Type"),
		ContentDisposition: res.Header.Get("Content-Disposition"),
		CacheControl:       res.Header.Get("Cache-Control"),
	}, language, nil
}

type WebSocketPushEnableType string
type WebSocketPushDisableType string

const (
	WebSocketPushTypeEnable  = WebSocketPushEnableType("WebSocketPushEnable")
	WebSocketPushTypeDisable = WebSocketPushDisableType("WebSocketPushDisable")
)

type WebSocketPushEnable struct {
	// This MUST be the string "WebSocketPushEnable".
	Type WebSocketPushEnableType `json:"@type"`

	// A list of data type names (e.g., "Mailbox" or "Email") that the client is interested in.
	//
	// A StateChange notification will only be sent if the data for one of these types changes.
	// Other types are omitted from the TypeState object.
	//
	// If null, changes will be pushed for all supported data types.
	DataTypes *[]string `json:"dataTypes"`

	// The last "pushState" token that the client received from the server.

	// Upon receipt of a "pushState" token, the server SHOULD immediately send all changes since that state token.
	PushState State `json:"pushState,omitempty"`
}

type WebSocketPushDisable struct {
	// This MUST be the string "WebSocketPushDisable".
	Type WebSocketPushDisableType `json:"@type"`
}

type HttpWsClientFactory struct {
	dialer        *websocket.Dialer
	authenticator HttpJmapClientAuthenticator
	logger        *log.Logger
	eventListener HttpJmapApiClientEventListener
}

var _ WsClientFactory = &HttpWsClientFactory{}

func NewHttpWsClientFactory(dialer *websocket.Dialer, authenticator HttpJmapClientAuthenticator, logger *log.Logger,
	eventListener HttpJmapApiClientEventListener) (*HttpWsClientFactory, error) {
	// RFC 8887: Section 4.2:
	// Otherwise, the client MUST make an authenticated HTTP request [RFC7235] on the encrypted connection
	// and MUST include the value "jmap" in the list of protocols for the "Sec-WebSocket-Protocol" header
	// field.
	dialer.Subprotocols = []string{"jmap"}

	return &HttpWsClientFactory{
		dialer:        dialer,
		authenticator: authenticator,
		logger:        logger,
		eventListener: eventListener,
	}, nil
}

func (w *HttpWsClientFactory) auth(ctx context.Context, username string, logger *log.Logger, h http.Header) Error {
	return w.authenticator.AuthenticateWS(ctx, username, logger, h)
}

func (w *HttpWsClientFactory) connect(ctx context.Context, sessionProvider func() (*Session, error)) (*websocket.Conn, string, string, Error) {
	logger := w.logger

	session, err := sessionProvider()
	if err != nil {
		return nil, "", "", jmapError(err, JmapErrorWssFailedToRetrieveSession)
	}
	if session == nil {
		return nil, "", "", jmapError(fmt.Errorf("WSS connection failed to retrieve JMAP session"), JmapErrorWssFailedToRetrieveSession)
	}

	if !session.SupportsWebsocketPush {
		return nil, "", "", jmapError(fmt.Errorf("WSS connection returned a session that does not support websocket push"), JmapErrorSocketPushUnsupported)
	}

	username := session.Username
	u := session.WebsocketUrl
	endpoint := session.WebsocketEndpoint

	h := http.Header{}
	w.auth(ctx, username, logger, h)
	w.logger.Trace().Str("username", log.SafeString(username)).Str("url", log.SafeString(u.String())).Msgf("connecting")
	c, res, err := w.dialer.DialContext(ctx, u.String(), h)
	if err != nil {
		return nil, "", endpoint, jmapError(err, JmapErrorFailedToEstablishWssConnection)
	}

	if w.logger.Trace().Enabled() {
		responseBytes, err := httputil.DumpResponse(res, true)
		if err == nil {
			logger.Trace().Str(logEndpoint, endpoint).Str(logProto, logProtoJmapWs).Str(logType, logTypeResponse).
				Str(logHttpStatus, log.SafeString(res.Status)).Int(logHttpStatusCode, res.StatusCode).
				Msg(string(responseBytes))
		}
	}

	if res.StatusCode != 101 {
		w.eventListener.OnFailedRequestWithStatus(endpoint, res.StatusCode)
		logger.Error().Str(logHttpStatus, log.SafeString(res.Status)).Int(logHttpStatusCode, res.StatusCode).Msg("HTTP response status code is not 101")
		return nil, "", endpoint, jmapError(fmt.Errorf("JMAP WS API response status is %v", res.Status), JmapErrorServerResponse)
	} else {
		w.eventListener.OnSuccessfulWsRequest(endpoint, res.StatusCode)
	}

	// RFC 8887: Section 4.2:
	// The reply from the server MUST also contain a corresponding "Sec-WebSocket-Protocol" header
	// field with a value of "jmap" in order for a JMAP subprotocol connection to be established.
	if !slices.Contains(res.Header.Values("Sec-WebSocket-Protocol"), "jmap") {
		return nil, "", endpoint, jmapError(fmt.Errorf("WSS connection header does not contain Sec-WebSocket-Protocol:jmap"), JmapErrorWssConnectionResponseMissingJmapSubprotocol)
	}

	return c, username, endpoint, nil
}

type HttpWsClient struct {
	client          *HttpWsClientFactory
	username        string
	sessionProvider func() (*Session, error)
	c               *websocket.Conn
	logger          *log.Logger
	endpoint        string
	listener        WsPushListener
	WsClient
}

func (w *HttpWsClient) readPump() { //NOSONAR
	logger := log.From(w.logger.With().Str("username", w.username))
	defer func() {
		if err := w.c.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Warn().Err(err).Msg("failed to close websocket connection")
		}
	}()
	//w.c.SetReadLimit(maxMessageSize)
	//c.conn.SetReadDeadline(time.Now().Add(pongWait))
	//c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		if _, message, err := w.c.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error().Err(err).Msg("unexpected close")
			}
			break
		} else {
			if logger.Trace().Enabled() {
				logger.Trace().Str(logEndpoint, w.endpoint).Str(logProto, logProtoJmapWs).Str(logType, logTypePush).Msg(string(message))
			}

			var peek struct {
				Type string `json:"@type"`
			}
			if err := json.Unmarshal(message, &peek); err != nil {
				logger.Error().Err(err).Msg("failed to deserialized pushed WS message")
				continue
			}
			switch peek.Type {
			case string(TypeOfStateChange):
				var stateChange StateChange
				if err := json.Unmarshal(message, &stateChange); err != nil {
					logger.Error().Err(err).Msgf("failed to deserialized pushed WS message into a %T", stateChange)
					continue
				} else {
					if w.listener != nil {
						w.listener.OnNotification(w.username, stateChange)
					} else {
						logger.Warn().Msgf("no listener to be notified of %v", stateChange)
					}
				}
			default:
				logger.Warn().Msgf("unsupported pushed WS message JMAP @type: '%s'", peek.Type)
				continue
			}
		}
	}
}

func (w *HttpWsClientFactory) EnableNotifications(ctx context.Context, pushState State, sessionProvider func() (*Session, error), listener WsPushListener) (WsClient, Error) {
	c, username, endpoint, jerr := w.connect(ctx, sessionProvider)
	if jerr != nil {
		return nil, jerr
	}

	msg := WebSocketPushEnable{
		Type:      WebSocketPushTypeEnable,
		DataTypes: nil,       // = all datatypes
		PushState: pushState, // will be omitted if empty string
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, jmapError(err, JmapErrorWssFailedToSendWebSocketPushEnable)
	}

	if w.logger.Trace().Enabled() {
		w.logger.Trace().Str(logEndpoint, endpoint).Str(logProto, logProtoJmapWs).Str(logType, logTypeRequest).Msg(string(data))
	}
	if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
		return nil, jmapError(err, JmapErrorWssFailedToSendWebSocketPushEnable)
	}

	wsc := &HttpWsClient{
		client:          w,
		username:        username,
		sessionProvider: sessionProvider,
		c:               c,
		logger:          w.logger,
		endpoint:        endpoint,
		listener:        listener,
	}

	go wsc.readPump()

	return wsc, nil
}

func (w *HttpWsClientFactory) Close() error {
	return nil
}

func (c *HttpWsClient) DisableNotifications() Error {
	if c.c == nil {
		return nil
	}

	werr := c.c.WriteJSON(WebSocketPushDisable{Type: WebSocketPushTypeDisable})
	merr := c.c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	cerr := c.c.Close()

	if werr != nil {
		return jmapError(werr, JmapErrorWssFailedToClose)
	}
	if merr != nil {
		return jmapError(merr, JmapErrorWssFailedToClose)
	}
	if cerr != nil {
		return jmapError(cerr, JmapErrorWssFailedToClose)
	}
	return nil
}

func (c *HttpWsClient) Close() error {
	return c.DisableNotifications()
}
