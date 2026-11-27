package config

import (
	"context"

	"go-micro.dev/v4/client"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	LogLevel string `yaml:"loglevel" env:"OC_LOG_LEVEL;SEARCH_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Debug    Debug  `yaml:"debug"`

	GRPC       GRPCConfig    `yaml:"grpc"`
	GrpcClient client.Client `yaml:"-"`

	TokenManager *TokenManager `yaml:"token_manager"`

	Reva                       *shared.Reva          `yaml:"reva"`
	GRPCClientTLS              *shared.GRPCClientTLS `yaml:"grpc_client_tls"`
	Events                     Events                `yaml:"events"`
	Engine                     Engine                `yaml:"engine"`
	Extractor                  Extractor             `yaml:"extractor"`
	ContentExtractionSizeLimit uint64                `yaml:"content_extraction_size_limit" env:"SEARCH_CONTENT_EXTRACTION_SIZE_LIMIT" desc:"Maximum file size in bytes that is allowed for content extraction. Of a larger file only the metadata is extracted, from its first bytes up to this size." introductionVersion:"1.0.0"`
	ContentExtractionTextLimit uint64                `yaml:"content_extraction_text_limit" env:"SEARCH_CONTENT_EXTRACTION_TEXT_LIMIT" desc:"Maximum size in bytes of the text extracted from one file that is indexed. Longer text is cut off at this size. 0 indexes all of it." introductionVersion:"%%NEXT%%"`
	BatchSize                  int                   `yaml:"batch_size" env:"SEARCH_BATCH_SIZE" desc:"The number of documents to process in a single batch. Defaults to 500." introductionVersion:"1.0.0"`
	ReindexMaxConcurrency      int                   `yaml:"reindex_concurrency" env:"SEARCH_REINDEX_MAX_CONCURRENCY" desc:"The maximum number of spaces that are reindexed concurrently when reindexing all spaces." introductionVersion:"7.4.0"`
	QueryTemplate              string                `yaml:"query_template" env:"SEARCH_QUERY_TEMPLATE" desc:"A KQL template every search runs through, see the search service documentation for the available placeholders." introductionVersion:"9.0.0"`

	ServiceAccount ServiceAccount `yaml:"service_account"`

	Context context.Context `yaml:"-"`
}

// ServiceAccount is the configuration for the used service account
type ServiceAccount struct {
	ServiceAccountID     string `yaml:"service_account_id" env:"OC_SERVICE_ACCOUNT_ID;SEARCH_SERVICE_ACCOUNT_ID" desc:"The ID of the service account the service should use. See the 'auth-service' service description for more details." introductionVersion:"1.0.0"`
	ServiceAccountSecret string `yaml:"service_account_secret" env:"OC_SERVICE_ACCOUNT_SECRET;SEARCH_SERVICE_ACCOUNT_SECRET" desc:"The service account secret." introductionVersion:"1.0.0"`
}
