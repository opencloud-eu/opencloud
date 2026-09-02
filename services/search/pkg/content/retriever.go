package content

import (
	"context"
	"io"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"google.golang.org/grpc/metadata"
)

// Retriever is the interface that wraps the basic Retrieve method. 🐕
// It requests and then returns a resource from the underlying storage.
type Retriever interface {
	Retrieve(ctx context.Context, rID *provider.ResourceId) (io.ReadCloser, error)
}

func contextGet(ctx context.Context, k string) (string, bool) {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return "", false
	}

	token, ok := md[k]
	if len(token) == 0 || !ok {
		return "", false
	}

	return token[0], ok
}
