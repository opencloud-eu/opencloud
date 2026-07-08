package jmap

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/stretchr/testify/require"
)

type testNopReadCloser struct {
	io.Reader
	closed bool
}

func (c *testNopReadCloser) Close() error {
	c.closed = true
	return nil
}

func TestResponse(t *testing.T) {
	text := "hello, world"

	logger := log.NewLogger(log.Level("trace"))

	for _, tt := range []struct {
		name               string
		peekSize           int
		expectedTruncation bool
		expectedPeek       string
	}{
		{"without peek", 0, false, ""},
		{"with truncated peek", 4, true, "hell"},
		{"with full peek", 1024, false, text},
	} {
		for _, tr := range []struct {
			name      string
			traceSize int
		}{
			{"with tracing", 256},
			{"without tracing", 0},
		} {
			for _, b := range []struct {
				name     string
				withBody bool
			}{
				{"with a body", true},
				{"without a body", false},
			} {
				t.Run(fmt.Sprintf("%s %s %s %s", t.Name(), tt.name, tr.name, b.name), func(t *testing.T) {
					require := require.New(t)

					var closer *testNopReadCloser
					var source io.ReadCloser
					if b.withBody {
						closer = &testNopReadCloser{closed: false, Reader: strings.NewReader(text)}
						source = closer
					} else {
						closer = nil
						source = http.NoBody
					}

					resp := &http.Response{
						Status:     "200 OK",
						StatusCode: 200,
						Header:     http.Header{},
						Body:       source,
					}

					client := HttpJmapClient{
						traceResponses:           true,
						traceMaxResponseBodySize: int64(tr.traceSize),
					}
					body, peek, truncated, err := client.response(int64(tt.peekSize), context.TODO(), &logger, "username", "https://stalwart", time.Duration(1*time.Second), resp)
					require.NoError(err)

					if b.withBody {
						require.Equal(tt.expectedPeek, string(peek))
						require.Equal(tt.expectedTruncation, truncated)
						buf := new(strings.Builder)
						_, err = io.Copy(buf, body)
						require.NoError(err)
						require.Equal(text, buf.String())
					} else {
						require.Empty(peek)
						require.False(truncated)
					}

					body.Close()

					if closer != nil {
						require.True(closer.closed)
					}
				})
			}
		}
	}
}
