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
	Name string `yaml:"name" env:"SEARCH_ENGINE_OPEN_SEARCH_RESOURCE_INDEX_NAME" desc:"The name of the OpenSearch index for resources." introductionVersion:"4.0.0"`
}

// EngineOpenSearchClient configures the OpenSearch client
type EngineOpenSearchClient struct {
	Addresses             []string      `yaml:"addresses" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_ADDRESSES" desc:"The addresses of the OpenSearch nodes.." introductionVersion:"4.0.0"`
	Username              string        `yaml:"username" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_USERNAME" desc:"Username for HTTP Basic Authentication." introductionVersion:"4.0.0"`
	Password              string        `yaml:"password" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_PASSWORD" desc:"Password for HTTP Basic Authentication." introductionVersion:"4.0.0"`
	Header                http.Header   `yaml:"header" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_HEADER" desc:"HTTP headers to include in requests." introductionVersion:"4.0.0"`
	CACert                string        `yaml:"ca_cert" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_CA_CERT" desc:"Path/File name for the root CA certificate (in PEM format) used to validate TLS server certificates of the opensearch server." introductionVersion:"4.0.0"`
	RetryOnStatus         []int         `yaml:"retry_on_status" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_RETRY_ON_STATUS" desc:"HTTP status codes that trigger a retry." introductionVersion:"4.0.0"`
	DisableRetry          bool          `yaml:"disable_retry" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_DISABLE_RETRY" desc:"Disable retries on errors." introductionVersion:"4.0.0"`
	EnableRetryOnTimeout  bool          `yaml:"enable_retry_on_timeout" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_ENABLE_RETRY_ON_TIMEOUT" desc:"Enable retries on timeout." introductionVersion:"4.0.0"`
	MaxRetries            int           `yaml:"max_retries" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_MAX_RETRIES" desc:"Maximum number of retries for requests." introductionVersion:"4.0.0"`
	CompressRequestBody   bool          `yaml:"compress_request_body" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_COMPRESS_REQUEST_BODY" desc:"Compress request bodies." introductionVersion:"4.0.0"`
	DiscoverNodesOnStart  bool          `yaml:"discover_nodes_on_start" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_DISCOVER_NODES_ON_START" desc:"Discover nodes on service start." introductionVersion:"4.0.0"`
	DiscoverNodesInterval time.Duration `yaml:"discover_nodes_interval" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_DISCOVER_NODES_INTERVAL" desc:"Interval for discovering nodes." introductionVersion:"4.0.0"`
	EnableMetrics         bool          `yaml:"enable_metrics" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_ENABLE_METRICS" desc:"Enable metrics collection." introductionVersion:"4.0.0"`
	EnableDebugLogger     bool          `yaml:"enable_debug_logger" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_ENABLE_DEBUG_LOGGER" desc:"Enable debug logging." introductionVersion:"4.0.0"`
	Insecure              bool          `yaml:"insecure" env:"SEARCH_ENGINE_OPEN_SEARCH_CLIENT_INSECURE" desc:"Skip TLS certificate verification." introductionVersion:"4.0.0"`
}
