package content

import (
	"context"
	"errors"
	"fmt"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
)

// Extractor is responsible to extract content and meta information from documents.
type Extractor interface {
	Extract(ctx context.Context, ri *provider.ResourceInfo) (Document, error)
}

func getFirstValue(m map[string][]string, key string) (string, error) {
	if m == nil {
		return "", errors.New("undefined map")
	}

	v, ok := m[key]
	if !ok {
		return "", fmt.Errorf("unknown key: %v", key)
	}

	if len(v) == 0 {
		return "", fmt.Errorf("no values for: %v", key)
	}

	return v[0], nil
}
