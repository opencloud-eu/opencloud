package config

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"go-micro.dev/v4/client"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	LogLevel string `yaml:"loglevel" env:"OC_LOG_LEVEL;WEBDAV_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Debug    Debug  `yaml:"debug"`

	GRPCClientTLS *shared.GRPCClientTLS `yaml:"grpc_client_tls"`
	GrpcClient    client.Client         `yaml:"-"`

	HTTP HTTP `yaml:"http"`

	OpenCloudPublicURL string `yaml:"opencloud_public_url" env:"OC_URL;OC_PUBLIC_URL" desc:"URL, where OpenCloud is reachable for users." introductionVersion:"1.0.0"`
	WebdavNamespace    string `yaml:"webdav_namespace" env:"WEBDAV_WEBDAV_NAMESPACE" desc:"CS3 path layout to use when forwarding /webdav requests" introductionVersion:"1.0.0"`
	RevaGateway        string `yaml:"reva_gateway" env:"OC_REVA_GATEWAY" desc:"CS3 gateway used to look up user metadata" introductionVersion:"1.0.0"`

	ThumbnailGeneratorURL        string `yaml:"thumbnail_generator_url" env:"WEBDAV_THUMBNAIL_GENERATOR_URL" desc:"Base URL of the thumbnail generator service, e.g. http://thumbnails:9130" introductionVersion:"1.0.0"`
	ThumbnailGeneratorTimeout    string `yaml:"thumbnail_generator_timeout" env:"WEBDAV_THUMBNAIL_GENERATOR_TIMEOUT" desc:"HTTP timeout for requests to the thumbnail generator, e.g. 30s" introductionVersion:"1.0.0"`
	ThumbnailGeneratorAuthHeader string `yaml:"thumbnail_generator_auth_header" env:"WEBDAV_THUMBNAIL_GENERATOR_AUTH_HEADER" desc:"Optional auth header value sent to the generator (for external generators behind a reverse proxy)" introductionVersion:"1.0.0"`
	MaxInputFileSize             string `yaml:"max_input_file_size" env:"WEBDAV_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE;THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE" desc:"Maximum file size of an input image for thumbnail generation. Usable common abbreviations: [KB, KiB, MB, MiB, GB, GiB], example: 50MB" introductionVersion:"1.0.0"`

	ThumbnailCacheBackend     string   `yaml:"thumbnail_cache_backend" env:"WEBDAV_THUMBNAIL_CACHE_BACKEND" 	desc:"Cache backend for imagor thumbnails: 'none', 'memory', 'file', or 's3'. Default: none." introductionVersion:"1.0.0"`
	ThumbnailCacheDir         string   `yaml:"thumbnail_cache_dir" env:"WEBDAV_THUMBNAIL_CACHE_DIR" desc:"Directory for file-based thumbnail cache (Default: $OC_BASE_DATA_PATH/thumbnails/files). Only used when ThumbnailCacheBackend is 'file'." introductionVersion:"1.0.0"`
	ThumbnailResolutions      []string `yaml:"thumbnail_resolutions" env:"THUMBNAILS_RESOLUTIONS;WEBDAV_THUMBNAIL_RESOLUTIONS" desc:"Supported target resolutions in the format WidthxHeight like 32x32. The requested size is snapped onto one of these (orientation-aware) before being sent to the generator." introductionVersion:"1.0.0"`
	ThumbnailCacheS3Bucket    string   `yaml:"thumbnail_cache_s3_bucket" env:"WEBDAV_THUMBNAIL_CACHE_S3_BUCKET" desc:"S3 bucket name for thumbnail cache when ThumbnailCacheBackend is 's3'." introductionVersion:"1.0.0"`
	ThumbnailCacheS3Region    string   `yaml:"thumbnail_cache_s3_region" env:"WEBDAV_THUMBNAIL_CACHE_S3_REGION" desc:"S3 region for thumbnail cache when ThumbnailCacheBackend is 's3'." introductionVersion:"1.0.0"`
	ThumbnailCacheS3Endpoint  string   `yaml:"thumbnail_cache_s3_endpoint" env:"WEBDAV_THUMBNAIL_CACHE_S3_ENDPOINT" desc:"S3 endpoint URL (for MinIO or compatible services) when ThumbnailCacheBackend is 's3'." introductionVersion:"1.0.0"`
	ThumbnailCacheS3AccessKey string   `yaml:"thumbnail_cache_s3_access_key" env:"WEBDAV_THUMBNAIL_CACHE_S3_ACCESS_KEY" desc:"S3 access key for thumbnail cache authentication." introductionVersion:"1.0.0"`
	ThumbnailCacheS3SecretKey string   `yaml:"thumbnail_cache_s3_secret_key" env:"WEBDAV_THUMBNAIL_CACHE_S3_SECRET_KEY" desc:"S3 secret key for thumbnail cache authentication." introductionVersion:"1.0.0"`
	FontMapFile               string   `yaml:"font_map_file" env:"WEBDAV_THUMBNAILS_TXT_FONTMAP_FILE;THUMBNAILS_TXT_FONTMAP_FILE" desc:"The path to a font map file for txt thumbnails." introductionVersion:"1.0.0"`

	Context context.Context `yaml:"-"`
}
