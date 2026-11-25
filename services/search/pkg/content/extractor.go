package content

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/go-tika/tika"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrUnreachable marks a failure to reach tika or the file, a failed connection
// or a server error, the next attempt may well get through
var ErrUnreachable = errors.New("tika or the file is unreachable")

// Extractor is responsible to extract content and meta information from documents.
type Extractor interface {
	Extract(ctx context.Context, ri *provider.ResourceInfo) (Document, error)
}

// RemoteExtractor is an Extractor that hands the content to a remote service
type RemoteExtractor interface {
	// Probe tells cheaply whether the remote answers, without a full extraction
	Probe(ctx context.Context) error
	// Remote tells whether extracting ri goes to the remote at all, a folder or
	// an empty file does not and says nothing about its state
	Remote(ri *provider.ResourceInfo) bool
}

// unreachable wraps the error of a failed call in ErrUnreachable, unless the
// other side answered that the request itself is wrong (not found, a 4xx)
func unreachable(err error) error {
	var tikaErr tika.ClientError
	if errors.As(err, &tikaErr) && !transientStatus(tikaErr.StatusCode) {
		return err
	}
	if s, ok := status.FromError(err); ok {
		switch s.Code() {
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
		default:
			return err
		}
	}
	return fmt.Errorf("%w: %w", ErrUnreachable, err)
}

// transientStatus tells an http status the next attempt may get past, a server
// error or too many requests, from an answer about the request itself
func transientStatus(code int) bool {
	return code >= http.StatusInternalServerError || code == http.StatusTooManyRequests
}

// getFirstValue returns the first metadata value present among keys, trying them
// in order. It errors when the map is nil or none of the keys holds a value.
func getFirstValue(m map[string][]string, keys ...string) (string, error) {
	for _, key := range keys {
		if v, ok := m[key]; ok && len(v) > 0 {
			return v[0], nil
		}
	}
	return "", fmt.Errorf("no value for keys: %v", keys)
}
