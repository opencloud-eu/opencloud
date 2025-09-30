package icapclient

import (
	"bufio"
	"bytes"
	"errors"
	"net/http"
	"strings"
)

// Client represents the ICAP client who makes the ICAP server calls.
type Client struct {
	config Config // Store config for connection parameters
}

// NewClient creates a new ICAP client (no persistent connection).
func NewClient(options ...ConfigOption) (Client, error) {
	config := DefaultConfig()
	for _, option := range options {
		option(&config)
	}
	return Client{config: config}, nil
}

// Do make the ICAP request, creating and dropping a connection each time.
func (c Client) Do(req Request) (res Response, err error) {
	conn, err := NewICAPConn(c.config.ICAPConn)
	if err != nil {
		return Response{}, err
	}

	if err := conn.Connect(req.ctx, req.URL.Host); err != nil {
		return Response{}, err
	}
	defer func() {
		err = errors.Join(err, conn.Close())
	}()

	req.setDefaultRequestHeaders()

	message, err := toICAPRequest(req)
	if err != nil {
		return Response{}, err
	}

	// send the ICAP message to the server
	dataRes, err := conn.Send(message)
	if err != nil {
		return Response{}, err
	}

	res, err = toClientResponse(bufio.NewReader(strings.NewReader(string(dataRes))))
	if err != nil {
		return Response{}, err
	}

	// check if the message is fully done scanning or if it needs to be sent another chunk.
	done := !(res.StatusCode == http.StatusContinue && !req.bodyFittedInPreview && req.previewSet)
	if done {
		return res, nil
	}

	// get the remaining body bytes.
	data := req.remainingPreviewBytes
	if !bodyIsChunked(string(data)) {
		data = []byte(addHexBodyByteNotations(string(data)))
	}

	// hydrate the ICAP message with closing doubleCRLF suffix.
	if !bytes.HasSuffix(data, []byte(doubleCRLF)) {
		data = append(data, []byte(crlf)...)
	}

	// send the remaining body bytes to the server.
	dataRes, err = conn.Send(data)
	if err != nil {
		return Response{}, err
	}

	return toClientResponse(bufio.NewReader(strings.NewReader(string(dataRes))))
}
