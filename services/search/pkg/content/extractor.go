package content

import (
	"context"
	"fmt"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
)

// Extractor is responsible to extract content and meta information from documents.
type Extractor interface {
	Extract(ctx context.Context, ri *provider.ResourceInfo) (Document, error)
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
