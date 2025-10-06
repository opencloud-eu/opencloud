package jmap

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/version"
)

// Implementation of ApiClient, SessionClient and BlobClient that uses
// HTTP to perform JMAP operations.
type HttpJmapClient struct {
	client         *http.Client
	masterUser     string
	masterPassword string
	userAgent      string
	listener       HttpJmapApiClientEventListener
}

var (
	_ ApiClient     = &HttpJmapClient{}
	_ SessionClient = &HttpJmapClient{}
	_ BlobClient    = &HttpJmapClient{}
)

const (
	logEndpoint       = "endpoint"
	logHttpStatus     = "status"
	logHttpStatusCode = "status-code"
	logHttpUrl        = "url"
)

/*
func bearer(req *http.Request, token string) {
	req.Header.Add("Authorization", "Bearer "+base64.StdEncoding.EncodeToString([]byte(token)))
}
*/

// Record JMAP HTTP execution events that may occur, e.g. using metrics.
type HttpJmapApiClientEventListener interface {
	OnSuccessfulRequest(endpoint string, status int)
	OnFailedRequest(endpoint string, err error)
	OnFailedRequestWithStatus(endpoint string, status int)
	OnResponseBodyReadingError(endpoint string, err error)
	OnResponseBodyUnmarshallingError(endpoint string, err error)
}

type nullHttpJmapApiClientEventListener struct {
}

func (l nullHttpJmapApiClientEventListener) OnSuccessfulRequest(endpoint string, status int) {
}
func (l nullHttpJmapApiClientEventListener) OnFailedRequest(endpoint string, err error) {
}
func (l nullHttpJmapApiClientEventListener) OnFailedRequestWithStatus(endpoint string, status int) {
}
func (l nullHttpJmapApiClientEventListener) OnResponseBodyReadingError(endpoint string, err error) {
}
func (l nullHttpJmapApiClientEventListener) OnResponseBodyUnmarshallingError(endpoint string, err error) {
}

var _ HttpJmapApiClientEventListener = nullHttpJmapApiClientEventListener{}

// An implementation of HttpJmapApiClientMetricsRecorder that does nothing.
func NullHttpJmapApiClientEventListener() HttpJmapApiClientEventListener {
	return nullHttpJmapApiClientEventListener{}
}

func NewHttpJmapClient(client *http.Client, masterUser string, masterPassword string, listener HttpJmapApiClientEventListener) *HttpJmapClient {
	return &HttpJmapClient{
		client:         client,
		masterUser:     masterUser,
		masterPassword: masterPassword,
		userAgent:      "OpenCloud/" + version.GetString(),
		listener:       listener,
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

func (h *HttpJmapClient) auth(username string, _ *log.Logger, req *http.Request) error {
	masterUsername := username + "%" + h.masterUser
	req.SetBasicAuth(masterUsername, h.masterPassword)
	return nil
}

var (
	errNilBaseUrl = errors.New("sessionUrl is nil")
)

func (h *HttpJmapClient) GetSession(sessionUrl *url.URL, username string, logger *log.Logger) (SessionResponse, Error) {
	if sessionUrl == nil {
		logger.Error().Msg("sessionUrl is nil")
		return SessionResponse{}, SimpleError{code: JmapErrorInvalidHttpRequest, err: errNilBaseUrl}
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
		return SessionResponse{}, SimpleError{code: JmapErrorInvalidHttpRequest, err: err}
	}
	h.auth(username, logger, req)
	req.Header.Add("Cache-Control", "no-cache, no-store, must-revalidate") // spec recommendation

	res, err := h.client.Do(req)
	if err != nil {
		h.listener.OnFailedRequest(endpoint, err)
		logger.Error().Err(err).Msgf("failed to perform GET %v", sessionUrl)
		return SessionResponse{}, SimpleError{code: JmapErrorInvalidHttpRequest, err: err}
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		h.listener.OnFailedRequestWithStatus(endpoint, res.StatusCode)
		logger.Error().Str(logHttpStatus, res.Status).Int(logHttpStatusCode, res.StatusCode).Msg("HTTP response status code is not 200")
		return SessionResponse{}, SimpleError{code: JmapErrorServerResponse, err: fmt.Errorf("JMAP API response status is %v", res.Status)}
	}
	h.listener.OnSuccessfulRequest(endpoint, res.StatusCode)

	if res.Body != nil {
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				logger.Error().Err(err).Msg("failed to close response body")
			}
		}(res.Body)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Error().Err(err).Msg("failed to read response body")
		h.listener.OnResponseBodyReadingError(endpoint, err)
		return SessionResponse{}, SimpleError{code: JmapErrorReadingResponseBody, err: err}
	}

	var data SessionResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		logger.Error().Str(logHttpUrl, sessionUrlStr).Err(err).Msg("failed to decode JSON payload from .well-known/jmap response")
		h.listener.OnResponseBodyUnmarshallingError(endpoint, err)
		return SessionResponse{}, SimpleError{code: JmapErrorDecodingResponseBody, err: err}
	}

	return data, nil
}

func (h *HttpJmapClient) Command(ctx context.Context, logger *log.Logger, session *Session, request Request, acceptLanguage string) ([]byte, Language, Error) {
	jmapUrl := session.JmapUrl.String()
	endpoint := session.JmapEndpoint
	logger = log.From(logger.With().Str(logEndpoint, endpoint))

	bodyBytes, err := json.Marshal(request)
	if err != nil {
		logger.Error().Err(err).Msg("failed to marshall JSON payload")
		return nil, "", SimpleError{code: JmapErrorEncodingRequestBody, err: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, jmapUrl, bytes.NewBuffer(bodyBytes))
	if err != nil {
		logger.Error().Err(err).Msgf("failed to create POST request for %v", jmapUrl)
		return nil, "", SimpleError{code: JmapErrorCreatingRequest, err: err}
	}

	// Some JMAP APIs use the Accept-Language header to determine which language to use to translate
	// texts in attributes.
	if acceptLanguage != "" {
		req.Header.Add("Accept-Language", acceptLanguage)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", h.userAgent)
	h.auth(session.Username, logger, req)

	res, err := h.client.Do(req)
	if err != nil {
		h.listener.OnFailedRequest(endpoint, err)
		logger.Error().Err(err).Msgf("failed to perform POST %v", jmapUrl)
		return nil, "", SimpleError{code: JmapErrorSendingRequest, err: err}
	}
	language := Language(res.Header.Get("Content-Language"))
	if res.StatusCode < 200 || res.StatusCode > 299 {
		h.listener.OnFailedRequestWithStatus(endpoint, res.StatusCode)
		logger.Error().Str(logEndpoint, endpoint).Str(logHttpStatus, res.Status).Msg("HTTP response status code is not 2xx")
		return nil, language, SimpleError{code: JmapErrorServerResponse, err: err}
	}
	if res.Body != nil {
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				logger.Error().Err(err).Msg("failed to close response body")
			}
		}(res.Body)
	}
	h.listener.OnSuccessfulRequest(endpoint, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Error().Err(err).Msg("failed to read response body")
		h.listener.OnResponseBodyReadingError(endpoint, err)
		return nil, language, SimpleError{code: JmapErrorServerResponse, err: err}
	}

	return body, language, nil
}

func (h *HttpJmapClient) UploadBinary(ctx context.Context, logger *log.Logger, session *Session, uploadUrl string, endpoint string, contentType string, acceptLanguage string, body io.Reader) (UploadedBlob, Language, Error) {
	logger = log.From(logger.With().Str(logEndpoint, endpoint))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadUrl, body)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to create POST request for %v", uploadUrl)
		return UploadedBlob{}, "", SimpleError{code: JmapErrorCreatingRequest, err: err}
	}
	req.Header.Add("Content-Type", contentType)
	req.Header.Add("User-Agent", h.userAgent)
	h.auth(session.Username, logger, req)
	if acceptLanguage != "" {
		req.Header.Add("Accept-Language", acceptLanguage)
	}

	res, err := h.client.Do(req)
	if err != nil {
		h.listener.OnFailedRequest(endpoint, err)
		logger.Error().Err(err).Msgf("failed to perform POST %v", uploadUrl)
		return UploadedBlob{}, "", SimpleError{code: JmapErrorSendingRequest, err: err}
	}
	language := Language(res.Header.Get("Content-Language"))
	if res.StatusCode < 200 || res.StatusCode > 299 {
		h.listener.OnFailedRequestWithStatus(endpoint, res.StatusCode)
		logger.Error().Str(logHttpStatus, res.Status).Int(logHttpStatusCode, res.StatusCode).Msg("HTTP response status code is not 2xx")
		return UploadedBlob{}, language, SimpleError{code: JmapErrorServerResponse, err: err}
	}
	if res.Body != nil {
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				logger.Error().Err(err).Msg("failed to close response body")
			}
		}(res.Body)
	}
	h.listener.OnSuccessfulRequest(endpoint, res.StatusCode)

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Error().Err(err).Msg("failed to read response body")
		h.listener.OnResponseBodyReadingError(endpoint, err)
		return UploadedBlob{}, language, SimpleError{code: JmapErrorServerResponse, err: err}
	}

	var result UploadedBlob
	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		logger.Error().Str(logHttpUrl, uploadUrl).Err(err).Msg("failed to decode JSON payload from the upload response")
		h.listener.OnResponseBodyUnmarshallingError(endpoint, err)
		return UploadedBlob{}, language, SimpleError{code: JmapErrorDecodingResponseBody, err: err}
	}

	return result, language, nil
}

func (h *HttpJmapClient) DownloadBinary(ctx context.Context, logger *log.Logger, session *Session, downloadUrl string, endpoint string, acceptLanguage string) (*BlobDownload, Language, Error) {
	logger = log.From(logger.With().Str(logEndpoint, endpoint))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadUrl, nil)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to create GET request for %v", downloadUrl)
		return nil, "", SimpleError{code: JmapErrorCreatingRequest, err: err}
	}
	req.Header.Add("User-Agent", h.userAgent)
	h.auth(session.Username, logger, req)
	if acceptLanguage != "" {
		req.Header.Add("Accept-Language", acceptLanguage)
	}

	res, err := h.client.Do(req)
	if err != nil {
		h.listener.OnFailedRequest(endpoint, err)
		logger.Error().Err(err).Msgf("failed to perform GET %v", downloadUrl)
		return nil, "", SimpleError{code: JmapErrorSendingRequest, err: err}
	}
	language := Language(res.Header.Get("Content-Language"))
	if res.StatusCode == http.StatusNotFound {
		return nil, language, nil
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		h.listener.OnFailedRequestWithStatus(endpoint, res.StatusCode)
		logger.Error().Str(logHttpStatus, res.Status).Int(logHttpStatusCode, res.StatusCode).Msg("HTTP response status code is not 2xx")
		return nil, language, SimpleError{code: JmapErrorServerResponse, err: err}
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
		Body:               res.Body,
		Size:               size,
		Type:               res.Header.Get("Content-Type"),
		ContentDisposition: res.Header.Get("Content-Disposition"),
		CacheControl:       res.Header.Get("Cache-Control"),
	}, language, nil
}

type WebSocketPushEnable struct {
	// This MUST be the string "WebSocketPushEnable".
	Type string `json:"@type"`

	// A list of data type names (e.g., "Mailbox" or "Email") that the client is interested in.
	//
	// A StateChange notification will only be sent if the data for one of these types changes.
	// Other types are omitted from the TypeState object.
	//
	// If null, changes will be pushed for all supported data types.
	DataTypes *[]string `json:"dataTypes"`

	// The last "pushState" token that the client received from the server.

	// Upon receipt of a "pushState" token, the server SHOULD immediately send all changes since that state token.
	PushState string `json:"pushState,omitempty"`
}

type WebSocketPushDisable struct {
	// This MUST be the string "WebSocketPushDisable".
	Type string `json:"@type"`
}

type HttpWsClientFactory struct {
	dialer         *websocket.Dialer
	masterUser     string
	masterPassword string
}

var _ WsClientFactory = &HttpWsClientFactory{}

func NewHttpWsClientFactory(dialer *websocket.Dialer, masterUser string, masterPassword string, logger *log.Logger) (*HttpWsClientFactory, error) {
	/*
		d := websocket.Dialer{
			TLSClientConfig:  &tls.Config{InsecureSkipVerify: true}, // TODO configurable
			HandshakeTimeout: 5 * time.Second,
			// RFC 8887: Section 4.2:
			// Otherwise, the client MUST make an authenticated HTTP request [RFC7235] on the encrypted connection
			// and MUST include the value "jmap" in the list of protocols for the "Sec-WebSocket-Protocol" header
			// field.
			Subprotocols: []string{"jmap"},
		}
	*/

	// RFC 8887: Section 4.2:
	// Otherwise, the client MUST make an authenticated HTTP request [RFC7235] on the encrypted connection
	// and MUST include the value "jmap" in the list of protocols for the "Sec-WebSocket-Protocol" header
	// field.
	dialer.Subprotocols = []string{"jmap"}

	return &HttpWsClientFactory{
		dialer:         dialer,
		masterUser:     masterUser,
		masterPassword: masterPassword,
	}, nil
}

func (w *HttpWsClientFactory) auth(username string, h http.Header) error {
	masterUsername := username + "%" + w.masterUser
	h.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(masterUsername+":"+w.masterPassword)))
	return nil
}

func (w *HttpWsClientFactory) connect(sessionProvider func() (*Session, error)) (*websocket.Conn, string, Error) {
	session, err := sessionProvider()
	if err != nil {
		return nil, "", SimpleError{code: JmapErrorWssFailedToRetrieveSession, err: err}
	}
	if session == nil {
		return nil, "", SimpleError{code: JmapErrorWssFailedToRetrieveSession, err: nil}
	}
	username := session.Username
	u := session.WebsocketUrl

	h := http.Header{}
	w.auth(username, h)
	c, resp, err := w.dialer.Dial(u.String(), h)
	if err != nil {
		return nil, "", SimpleError{code: JmapErrorFailedToEstablishWssConnection, err: err}
	}

	// RFC 8887: Section 4.2:
	// The reply from the server MUST also contain a corresponding "Sec-WebSocket-Protocol" header
	// field with a value of "jmap" in order for a JMAP subprotocol connection to be established.
	if !slices.Contains(resp.Header.Values("Sec-WebSocket-Protocol"), "jmap") {
		return nil, "", SimpleError{code: JmapErrorWssConnectionResponseMissingJmapSubprotocol}
	}

	return c, username, nil
}

type HttpWsClient struct {
	client          *HttpWsClientFactory
	username        string
	sessionProvider func() (*Session, error)
	c               *websocket.Conn
	WsClient
}

func (w *HttpWsClientFactory) EnableNotifications(pushState string, sessionProvider func() (*Session, error), listener WsPushListener) (WsClient, Error) {
	c, username, jerr := w.connect(sessionProvider)
	if jerr != nil {
		return nil, jerr
	}

	err := c.WriteJSON(WebSocketPushEnable{
		Type:      "WebSocketPushEnable",
		DataTypes: nil,       // = all datatypes
		PushState: pushState, // will be omitted if empty string
	})
	if err != nil {
		return nil, SimpleError{code: JmapErrorWssFailedToSendWebSocketPushEnable, err: err}
	}

	return &HttpWsClient{
		client:          w,
		username:        username,
		sessionProvider: sessionProvider,
		c:               c,
	}, nil
}

func (w *HttpWsClientFactory) Close() error {
	return nil
}

func (c *HttpWsClient) DisableNotifications() Error {
	if c.c == nil {
		return nil
	}

	err := c.c.WriteJSON(WebSocketPushDisable{
		Type: "WebSocketPushDisable",
	})
	if err != nil {
		return SimpleError{code: JmapErrorWssFailedToSendWebSocketPushDisable, err: err}
	}

	err = c.c.Close()
	if err != nil {
		return SimpleError{code: JmapErrorWssFailedToClose, err: err}
	}

	return nil
}

func (c *HttpWsClient) Close() error {
	return c.DisableNotifications()
}
