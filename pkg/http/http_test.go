package http

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPeekResponse(t *testing.T) {
	require := require.New(t)
	n := 6
	text := "hello, world"

	require.Less(n, len(text))
	body, peek, truncated, err := PeekResponse(io.NopCloser(strings.NewReader(text)), int64(n))
	require.NoError(err)
	require.True(truncated)
	require.Equal(text[:n], string(peek))
	buf := new(strings.Builder)
	_, err = io.Copy(buf, body)
	require.NoError(err)
	require.Equal(text, buf.String())
}

func TestDumpHttpResponse(t *testing.T) {
	require := require.New(t)
	n := 6
	text := "hello, world"

	h := http.Header{}
	h.Set("testing", "true")
	h.Add("aaa", "1")
	h.Add("aaa", "2")

	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     h,
		Body:       nil,
	}

	body, err := DumpHttpResponse(resp, io.NopCloser(strings.NewReader(text)), int64(n), func(method, uri, content string, truncated bool) {
		require.True(truncated)
		require.Equal(fmt.Sprintf(`200 OK
Testing: true
Aaa: 1
Aaa: 2
%s`, text[:n]), content)
	})
	require.NoError(err)
	buf := new(strings.Builder)
	_, err = io.Copy(buf, body)
	require.NoError(err)
	require.Equal(text, buf.String())
}

func TestDumpHttpRequest(t *testing.T) {
	require := require.New(t)
	n := 6
	text := "hello, world"

	h := http.Header{}
	h.Set("testing", "true")
	h.Add("aaa", "1")
	h.Add("aaa", "2")

	req := &http.Request{
		Method: "GET",
		Header: h,
		URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/testing"},
		Body:   io.NopCloser(strings.NewReader(text)),
	}

	err := DumpHttpRequest(req, int64(n), func(method, uri, content string, truncated bool) {
		require.True(truncated)
		require.Equal(fmt.Sprintf(`GET https://example.com/testing
Testing: true
Aaa: 1
Aaa: 2
%s`, text[:n]), content)
	})
	require.NoError(err)
	buf := new(strings.Builder)
	_, err = io.Copy(buf, req.Body)
	require.NoError(err)
	require.Equal(text, buf.String())
}
