package icapclient

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strconv"
	"strings"
)

// the icap request methods.
const (
	MethodOPTIONS = "OPTIONS"
	MethodRESPMOD = "RESPMOD"
	MethodREQMOD  = "REQMOD"
)

// shared errors.
var (
	// ErrNoContext is used when no context is provided.
	ErrNoContext = errors.New("no context provided")

	// ErrInvalidScheme is used when the url scheme is not icap://.
	ErrInvalidScheme = errors.New("the url scheme must be icap://")

	// ErrMethodNotAllowed is used when the method is not allowed.
	ErrMethodNotAllowed = errors.New("the requested method is not registered")

	// ErrInvalidHost is used when the host is invalid.
	ErrInvalidHost = errors.New("the requested host is invalid")

	// ErrInvalidMsg is used when the tcp message is invalid.
	ErrInvalidMsg = errors.New("invalid message")

	// ErrInvalidConnection is used when the connection is invalid.
	ErrInvalidConnection = errors.New("invalid connection")

	// ErrREQMODWithoutReq is used when the request is nil for REQMOD method.
	ErrREQMODWithoutReq = errors.New("http request cannot be nil for method REQMOD")

	// ErrREQMODWithResp is used when the response is not nil for REQMOD method.
	ErrREQMODWithResp = errors.New("http response must be nil for method REQMOD")

	// ErrRESPMODWithoutResp is used when the response is nil for RESPMOD method.
	ErrRESPMODWithoutResp = errors.New("http response cannot be nil for method RESPMOD")
)

// general constants required for the package.
const (
	schemeICAP                      = "icap"
	icapVersion                     = "ICAP/1.0"
	httpVersion                     = "HTTP/1.1"
	schemeHTTPReq                   = "http_request"
	schemeHTTPResp                  = "http_response"
	crlf                            = "\r\n"
	doubleCRLF                      = crlf + crlf
	lf                              = "\n"
	bodyEndIndicator                = crlf + "0" + crlf
	fullBodyEndIndicatorPreviewMode = "; ieof" + doubleCRLF
	icap100ContinueMsg              = "ICAP/1.0 100 Continue" + doubleCRLF
	icap204NoModsMsg                = "ICAP/1.0 204 Unmodified"
)

// Common ICAP headers.
const (
	previewHeader      = "Preview"
	encapsulatedHeader = "Encapsulated"
)

// Conn represents the connection to the icap server.
type Conn interface {
	io.Closer
	Connect(ctx context.Context, address string) error
	Send(in []byte) ([]byte, error)
}

// Response represents the icap server response data.
type Response struct {
	StatusCode      int
	Status          string
	PreviewBytes    int
	Header          http.Header
	ContentRequest  *http.Request
	ContentResponse *http.Response
}

// getStatusWithCode prepares the status code and status text from two given strings.
func getStatusWithCode(str1, str2 string) (int, string, error) {
	statusCode, err := strconv.Atoi(str1)
	if err != nil {
		return 0, "", err
	}

	status := strings.TrimSpace(str2)

	return statusCode, status, nil
}

// getHeaderValue parses the header and its value from a tcp message string.
func getHeaderValue(str string) (string, string) {
	headerValues := strings.SplitN(str, ":", 2)
	header := headerValues[0]

	if len(headerValues) >= 2 {
		return header, strings.TrimSpace(headerValues[1])
	}

	return header, ""
}

// isRequestLine determines if the tcp message string is a request line, i.e., the first line of the message or not.
func isRequestLine(str string) bool {
	return strings.Contains(str, icapVersion) || strings.Contains(str, httpVersion)
}

// setEncapsulatedHeaderValue generates the Encapsulated values and assigns to the ICAP request string.
func setEncapsulatedHeaderValue(icapReqStr, httpReqStr, httpRespStr string) string {
	encVal := " "

	if strings.HasPrefix(icapReqStr, MethodOPTIONS) {
		switch {
		// the most common case for OPTIONS method, no Encapsulated body
		case httpReqStr == "" && httpRespStr == "":
			encVal += "null-body=0"
			// if there is an Encapsulated body
		default:
			encVal += "opt-body=0"
		}
	}

	if strings.HasPrefix(icapReqStr, MethodREQMOD) || strings.HasPrefix(icapReqStr, MethodRESPMOD) {
		// looking for the match of the string \r\n\r\n,
		// as that is the expression that separates each block, i.e., headers and bodies
		re := regexp.MustCompile(doubleCRLF)

		// getting the offsets of the matches, tells us the starting/ending point of headers and bodies
		reqIndices := re.FindAllStringIndex(httpReqStr, -1)

		// is needed to calculate the response headers by adding the last offset of the request block
		reqEndsAt := 0

		if reqIndices != nil {
			encVal += "req-hdr=0"
			reqEndsAt = reqIndices[0][1]

			switch {
			// indicating there is a body present for the request block, as length would have been 1 for a single match of \r\n\r\n
			case len(reqIndices) > 1:
				encVal += fmt.Sprintf(", req-body=%d", reqIndices[0][1]) // assigning the starting point of the body
				reqEndsAt = reqIndices[1][1]
			case httpRespStr == "":
				encVal += fmt.Sprintf(", null-body=%d", reqIndices[0][1])
			}

			if httpRespStr != "" {
				encVal += ", "
			}
		}

		respIndices := re.FindAllStringIndex(httpRespStr, -1)

		if respIndices != nil {
			encVal += fmt.Sprintf("res-hdr=%d", reqEndsAt)

			switch {
			case len(respIndices) > 1:
				encVal += fmt.Sprintf(", res-body=%d", reqEndsAt+respIndices[0][1])
			default:
				encVal += fmt.Sprintf(", null-body=%d", reqEndsAt+respIndices[0][1])
			}
		}
	}

	// formatting the ICAP request Encapsulated header with the value
	return fmt.Sprintf(icapReqStr, encVal)
}

// replaceRequestURIWithActualURL replaces just the escaped portion of the url with the entire URL in the dumped request message.
func replaceRequestURIWithActualURL(str, uri, url string) string {
	if uri == "" {
		uri = "/"
	}

	return strings.Replace(str, uri, url, 1)
}

// addFullBodyInPreviewIndicator adds 0; ieof\r\n\r\n which indicates the entire body fitted in the preview.
func addFullBodyInPreviewIndicator(str string) string {
	return strings.TrimSuffix(str, doubleCRLF) + fullBodyEndIndicatorPreviewMode
}

// splitBodyAndHeader separates header and body from a http message.
func splitBodyAndHeader(str string) (string, string, bool) {
	ss := strings.SplitN(str, doubleCRLF, 2)

	if len(ss) < 2 || ss[1] == "" {
		return "", "", false
	}

	headerStr := ss[0]
	bodyStr := ss[1]

	return headerStr, bodyStr, true
}

// bodyIsChunked determines if the http body is already chunked from the origin server or not.
func bodyIsChunked(str string) bool {
	_, bodyStr, ok := splitBodyAndHeader(str)

	if !ok {
		return false
	}

	return regexp.MustCompile(`\r\n0(\r\n)+$`).MatchString(bodyStr)
}

// parsePreviewBodyBytes parses the preview portion of the body and only keeps that in the message.
func parsePreviewBodyBytes(str string, pb int) string {
	headerStr, bodyStr, ok := splitBodyAndHeader(str)
	if !ok {
		return str
	}

	return headerStr + doubleCRLF + bodyStr[:pb]
}

// 0.
func addHexBodyByteNotations(str string) string {
	return fmt.Sprintf("%x%s%s%s", len([]byte(str)), crlf, str, bodyEndIndicator)
}

// addHeaderAndBody merges the header and body of the http message.
func addHeaderAndBody(headerStr, bodyStr string) string {
	return headerStr + doubleCRLF + bodyStr
}

// toICAPRequest returns the given request in its ICAP/1.x wire.
func toICAPRequest(req Request) ([]byte, error) {
	// Making the ICAP message block
	reqStr := fmt.Sprintf("%s %s %s%s", req.Method, req.URL.String(), icapVersion, crlf)

	for headerName, values := range req.Header {
		for _, value := range values {
			reqStr += fmt.Sprintf("%s: %s%s", headerName, value, crlf)
		}
	}

	// will populate the Encapsulated header value after making the http Request & Response messages
	reqStr += "Encapsulated: %s" + crlf
	reqStr += crlf

	// build the HTTP Request message block
	httpReqStr := ""
	if req.HTTPRequest != nil {
		b, err := httputil.DumpRequestOut(req.HTTPRequest, true)
		if err != nil {
			return nil, err
		}

		httpReqStr += string(b)
		httpReqStr = replaceRequestURIWithActualURL(httpReqStr, req.HTTPRequest.URL.EscapedPath(), req.HTTPRequest.URL.String())

		if req.Method == MethodREQMOD {
			if req.previewSet {
				httpReqStr = parsePreviewBodyBytes(httpReqStr, req.PreviewBytes)
			}

			if !bodyIsChunked(httpReqStr) {
				headerStr, bodyStr, ok := splitBodyAndHeader(httpReqStr)
				if ok {
					bodyStr = addHexBodyByteNotations(bodyStr)
					httpReqStr = addHeaderAndBody(headerStr, bodyStr)
				}
			}
		}

		// if the HTTP Request message block doesn't end with a \r\n\r\n,
		// then going to add one by force for better calculation of byte offsets
		if httpReqStr != "" {
			for !strings.HasSuffix(httpReqStr, doubleCRLF) {
				httpReqStr += crlf
			}
		}
	}

	// build the HTTP Response message block
	httpRespStr := ""
	if req.HTTPResponse != nil {
		b, err := httputil.DumpResponse(req.HTTPResponse, true)
		if err != nil {
			return nil, err
		}

		httpRespStr += string(b)

		if req.previewSet {
			httpRespStr = parsePreviewBodyBytes(httpRespStr, req.PreviewBytes)
		}

		if !bodyIsChunked(httpRespStr) {
			headerStr, bodyStr, ok := splitBodyAndHeader(httpRespStr)
			if ok {
				bodyStr = addHexBodyByteNotations(bodyStr)
				httpRespStr = addHeaderAndBody(headerStr, bodyStr)
			}
		}

		if httpRespStr != "" && !strings.HasSuffix(httpRespStr, doubleCRLF) { // if the HTTP Response message block doesn't end with a \r\n\r\n, then going to add one by force for better calculation of byte offsets
			httpRespStr += crlf
		}
	}

	if encVal := req.Header.Get(encapsulatedHeader); encVal != "" {
		reqStr = fmt.Sprintf(reqStr, encVal)
	} else {
		// populating the Encapsulated header of the ICAP message portion
		reqStr = setEncapsulatedHeaderValue(reqStr, httpReqStr, httpRespStr)
	}

	// determining if the http message needs the full body fitted in the preview portion indicator or not
	if httpRespStr != "" && req.previewSet && req.bodyFittedInPreview {
		httpRespStr = addFullBodyInPreviewIndicator(httpRespStr)
	}

	if req.Method == MethodREQMOD && req.previewSet && req.bodyFittedInPreview {
		httpReqStr = addFullBodyInPreviewIndicator(httpReqStr)
	}

	data := []byte(reqStr + httpReqStr + httpRespStr)

	return data, nil
}

// toClientResponse reads an ICAP message and returns a Response.
func toClientResponse(b *bufio.Reader) (Response, error) {
	resp := Response{
		Header: make(map[string][]string),
	}

	scheme := ""
	httpMsg := ""
	for currentMsg, err := b.ReadString('\n'); err == nil || currentMsg != ""; currentMsg, err = b.ReadString('\n') { // keep reading the buffer message which is the http response message
		// if the current message line if the first line of the message portion(request line)
		if isRequestLine(currentMsg) {
			ss := strings.Split(currentMsg, " ")

			// must contain 3 words, for example, "ICAP/1.0 200 OK" or "GET /something HTTP/1.1"
			if len(ss) < 3 {
				return Response{}, fmt.Errorf("%w: %s", ErrInvalidMsg, currentMsg)
			}

			// preparing the scheme below
			if ss[0] == icapVersion {
				scheme = schemeICAP

				resp.StatusCode, resp.Status, err = getStatusWithCode(ss[1], strings.Join(ss[2:], " "))
				if err != nil {
					return Response{}, err
				}

				continue
			}

			if ss[0] == httpVersion {
				scheme = schemeHTTPResp
				httpMsg = ""
			}

			// http request message scheme version should always be at the end,
			// for example, GET /something HTTP/1.1
			if strings.TrimSpace(ss[2]) == httpVersion {
				scheme = schemeHTTPReq
				httpMsg = ""
			}
		}

		// preparing the header for ICAP & contents for the HTTP messages below
		if scheme == schemeICAP {
			// ignore the CRLF and the LF, shouldn't be counted
			if currentMsg == lf || currentMsg == crlf {
				continue
			}

			header, val := getHeaderValue(currentMsg)
			if header == previewHeader {
				pb, _ := strconv.Atoi(val)
				resp.PreviewBytes = pb
			}

			resp.Header.Add(header, val)
		}

		if scheme == schemeHTTPReq {
			httpMsg += strings.TrimSpace(currentMsg) + crlf
			bufferEmpty := b.Buffered() == 0

			// a crlf indicates the end of the HTTP message,
			// and the buffer check is just in case the buffer ended with one last message instead of a crlf
			if currentMsg == crlf || bufferEmpty {
				request, err := http.ReadRequest(bufio.NewReader(strings.NewReader(httpMsg)))
				if err != nil {
					return Response{}, err
				}
				resp.ContentRequest = request

				continue
			}
		}

		if scheme == schemeHTTPResp {
			httpMsg += strings.TrimSpace(currentMsg) + crlf
			bufferEmpty := b.Buffered() == 0

			if currentMsg == crlf || bufferEmpty {
				response, err := http.ReadResponse(bufio.NewReader(strings.NewReader(httpMsg)), resp.ContentRequest)
				if err != nil {
					return Response{}, err
				}
				resp.ContentResponse = response

				continue
			}
		}
	}

	return resp, nil
}
