package opensearch

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	opensearchgo "github.com/opensearch-project/opensearch-go/v4"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
)

// NewClient builds an OpenSearch API client from the engine client config.
func NewClient(cfg config.EngineOpenSearchClient) (*opensearchgoAPI.Client, error) {
	clientConfig := opensearchgo.Config{
		Addresses:             cfg.Addresses,
		Username:              cfg.Username,
		Password:              cfg.Password,
		Header:                cfg.Header,
		RetryOnStatus:         cfg.RetryOnStatus,
		DisableRetry:          cfg.DisableRetry,
		EnableRetryOnTimeout:  cfg.EnableRetryOnTimeout,
		MaxRetries:            cfg.MaxRetries,
		CompressRequestBody:   cfg.CompressRequestBody,
		DiscoverNodesOnStart:  cfg.DiscoverNodesOnStart,
		DiscoverNodesInterval: cfg.DiscoverNodesInterval,
		EnableMetrics:         cfg.EnableMetrics,
		EnableDebugLogger:     cfg.EnableDebugLogger,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: cfg.Insecure,
			},
		},
	}

	if cfg.CACert != "" {
		certBytes, err := os.ReadFile(cfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}
		clientConfig.CACert = certBytes
	}

	client, err := opensearchgoAPI.NewClient(opensearchgoAPI.Config{Client: clientConfig})
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenSearch client: %w", err)
	}
	return client, nil
}
