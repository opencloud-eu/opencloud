package config

import (
	"net/http"
	"time"
)

// Engine defines which search engine to use
type Engine struct {
	Type       string           `yaml:"type" env:"SEARCH_ENGINE_TYPE" desc:"Defines which search engine to use. Defaults to 'bleve'. Supported values are: 'bleve'." introductionVersion:"1.0.0"`
	Bleve      EngineBleve      `yaml:"bleve"`
	OpenSearch EngineOpenSearch `yaml:"open_search"`
}

// EngineBleve configures the bleve engine
type EngineBleve struct {
	Datapath string `yaml:"data_path" env:"SEARCH_ENGINE_BLEVE_DATA_PATH" desc:"The directory where the filesystem will store search data. If not defined, the root directory derives from $OC_BASE_DATA_PATH/search." introductionVersion:"1.0.0"`
}

// EngineOpenSearch configures the OpenSearch engine
type EngineOpenSearch struct {
	Client        EngineOpenSearchClient        `yaml:"client"`
	ResourceIndex EngineOpenSearchResourceIndex `yaml:"resource_index"`
}

// EngineOpenSearchResourceIndex defines the OpenSearch index for resources
type EngineOpenSearchResourceIndex struct {
	Name string `yaml:"name" env:"SEARCH_ENGINE_OPEN_SEARCH_RESOURCE_INDEX_NAME" desc:"The name of the OpenSearch index for resources." introductionVersion:"%%NEXT%%"`
}

// EngineOpenSearchClient configures the OpenSearch client
type EngineOpenSearchClient struct {
	Addresses             []string      `yaml:"addresses" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_ADDRESSES" desc:"The addresses of the OpenSearch nodes.." introductionVersion:"%%NEXT%%"`
	Username              string        `yaml:"username" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_USERNAME" desc:"Username for HTTP Basic Authentication." introductionVersion:"%%NEXT%%"`
	Password              string        `yaml:"password" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_PASSWORD" desc:"Password for HTTP Basic Authentication." introductionVersion:"%%NEXT%%"`
	Header                http.Header   `yaml:"header" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_HEADER" desc:"HTTP headers to include in requests." introductionVersion:"%%NEXT%%"`
	CACert                []byte        `yaml:"ca_cert" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_CA_CERT" desc:"CA certificate for TLS connections." introductionVersion:"%%NEXT%%"`
	RetryOnStatus         []int         `yaml:"retry_on_status" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_RETRY_ON_STATUS" desc:"HTTP status codes that trigger a retry." introductionVersion:"%%NEXT%%"`
	DisableRetry          bool          `yaml:"disable_retry" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_DISABLE_RETRY" desc:"Disable retries on errors." introductionVersion:"%%NEXT%%"`
	EnableRetryOnTimeout  bool          `yaml:"enable_retry_on_timeout" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_ENABLE_RETRY_ON_TIMEOUT" desc:"Enable retries on timeout." introductionVersion:"%%NEXT%%"`
	MaxRetries            int           `yaml:"max_retries" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_MAX_RETRIES" desc:"Maximum number of retries for requests." introductionVersion:"%%NEXT%%"`
	CompressRequestBody   bool          `yaml:"compress_request_body" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_COMPRESS_REQUEST_BODY" desc:"Compress request bodies." introductionVersion:"%%NEXT%%"`
	DiscoverNodesOnStart  bool          `yaml:"discover_nodes_on_start" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_DISCOVER_NODES_ON_START" desc:"Discover nodes on service start." introductionVersion:"%%NEXT%%"`
	DiscoverNodesInterval time.Duration `yaml:"discover_nodes_interval" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_DISCOVER_NODES_INTERVAL" desc:"Interval for discovering nodes." introductionVersion:"%%NEXT%%"`
	EnableMetrics         bool          `yaml:"enable_metrics" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_ENABLE_METRICS" desc:"Enable metrics collection." introductionVersion:"%%NEXT%%"`
	EnableDebugLogger     bool          `yaml:"enable_debug_logger" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_ENABLE_DEBUG_LOGGER" desc:"Enable debug logging." introductionVersion:"%%NEXT%%"`
	Insecure              bool          `yaml:"insecure" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_INSECURE" desc:"Skip TLS certificate verification." introductionVersion:"%%NEXT%%"`
}
