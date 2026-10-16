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
	DisablePreviews    bool   `yaml:"disablePreviews" env:"OC_DISABLE_PREVIEWS;WEBDAV_DISABLE_PREVIEWS" desc:"Set this option to 'true' to disable rendering of thumbnails triggered via webdav access. Note that when disabled, all access to preview related webdav paths will return a 404." introductionVersion:"1.0.0"`

	ThumbnailGeneratorURL     string `yaml:"thumbnail_generator_url" env:"WEBDAV_THUMBNAIL_GENERATOR_URL" desc:"Base URL of the thumbnail generator service, e.g. http://thumbnails:9186" introductionVersion:"%%NEXT%%"`
	ThumbnailGeneratorTimeout string `yaml:"thumbnail_generator_timeout" env:"WEBDAV_THUMBNAIL_GENERATOR_TIMEOUT" desc:"HTTP timeout for requests to the thumbnail generator, e.g. 30s" introductionVersion:"%%NEXT%%"`
	// the yaml key stays unprefixed, a config file written against it has to keep working
	ThumbnailMaxInputFileSize string `yaml:"max_input_file_size" env:"WEBDAV_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE;THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE" desc:"Maximum file size of an input image for thumbnail generation. Usable common abbreviations: [KB, KiB, MB, MiB, GB, GiB], example: 50MB" introductionVersion:"%%NEXT%%"`

	ThumbnailCacheBackend string `yaml:"thumbnail_cache_backend" env:"WEBDAV_THUMBNAIL_CACHE_BACKEND" desc:"Cache backend for imagor thumbnails: 'none', 'memory', 'file', or 's3'. Default: file." introductionVersion:"%%NEXT%%"`
	ThumbnailCacheDir     string `yaml:"thumbnail_cache_dir" env:"WEBDAV_THUMBNAIL_CACHE_DIR" desc:"Directory for file-based thumbnail cache (Default: $OC_BASE_DATA_PATH/thumbnails/files). Only used when ThumbnailCacheBackend is 'file'." introductionVersion:"%%NEXT%%"`
	// kept so a deployment that configured the thumbnails service keeps its cache
	ThumbnailFilesystemStorageRoot string   `yaml:"thumbnail_filesystemstorage_root" env:"THUMBNAILS_FILESYSTEMSTORAGE_ROOT" desc:"Directory the thumbnails service stored its files in. Read as a fallback for WEBDAV_THUMBNAIL_CACHE_DIR." introductionVersion:"1.0.0" deprecationVersion:"%%NEXT%%" removalVersion:"%%NEXT_PRODUCTION_VERSION%%" deprecationInfo:"THUMBNAILS_FILESYSTEMSTORAGE_ROOT is deprecated. The webdav service owns the thumbnail cache now, so use WEBDAV_THUMBNAIL_CACHE_DIR instead." deprecationReplacement:"WEBDAV_THUMBNAIL_CACHE_DIR"`
	ThumbnailResolutions           []string `yaml:"thumbnail_resolutions" env:"THUMBNAILS_RESOLUTIONS;WEBDAV_THUMBNAIL_RESOLUTIONS" desc:"Allowed thumbnail edge lengths, like 512. Each axis of a requested size is rounded up to the next one of these before the box is sent to the generator. Legacy WidthxHeight entries are still accepted and contribute both of their sides." introductionVersion:"%%NEXT%%"`
	ThumbnailCacheS3Bucket         string   `yaml:"thumbnail_cache_s3_bucket" env:"WEBDAV_THUMBNAIL_CACHE_S3_BUCKET" desc:"S3 bucket name for thumbnail cache when ThumbnailCacheBackend is 's3'." introductionVersion:"%%NEXT%%"`
	ThumbnailCacheS3Region         string   `yaml:"thumbnail_cache_s3_region" env:"WEBDAV_THUMBNAIL_CACHE_S3_REGION" desc:"S3 region for thumbnail cache when ThumbnailCacheBackend is 's3'." introductionVersion:"%%NEXT%%"`
	ThumbnailCacheS3Endpoint       string   `yaml:"thumbnail_cache_s3_endpoint" env:"WEBDAV_THUMBNAIL_CACHE_S3_ENDPOINT" desc:"S3 endpoint URL (for MinIO or compatible services) when ThumbnailCacheBackend is 's3'." introductionVersion:"%%NEXT%%"`
	ThumbnailCacheS3AccessKey      string   `yaml:"thumbnail_cache_s3_access_key" env:"WEBDAV_THUMBNAIL_CACHE_S3_ACCESS_KEY" desc:"S3 access key for thumbnail cache authentication." introductionVersion:"%%NEXT%%"`
	ThumbnailCacheS3SecretKey      string   `yaml:"thumbnail_cache_s3_secret_key" env:"WEBDAV_THUMBNAIL_CACHE_S3_SECRET_KEY" desc:"S3 secret key for thumbnail cache authentication." introductionVersion:"%%NEXT%%"`
	ThumbnailSourceTypes           []string `yaml:"thumbnail_source_types" env:"OC_THUMBNAIL_SOURCE_TYPES;WEBDAV_THUMBNAIL_SOURCE_TYPES" desc:"Additional mime types a preview is announced and attempted for, e.g. image/vnd.microsoft.icon for a generator that reads icons. The types the build ships are included; this only adds to them. A type the running generator cannot read answers 404." introductionVersion:"%%NEXT%%"`

	ThumbnailGeneratorFormats []string `yaml:"thumbnail_generator_formats" env:"OC_THUMBNAIL_GENERATOR_FORMATS;WEBDAV_THUMBNAIL_GENERATOR_FORMATS" desc:"The formats the thumbnail generator writes, as mime:format entries, e.g. image/webp:webp. The format is the name the generator's own format filter uses. A type that is not listed is one the generator cannot write, and a thumbnail mapped to it becomes a JPEG instead." introductionVersion:"%%NEXT%%"`
	ThumbnailTypeMapping      []string `yaml:"thumbnail_type_mapping" env:"OC_THUMBNAIL_TYPE_MAPPING;WEBDAV_THUMBNAIL_TYPE_MAPPING" desc:"The type a thumbnail is returned in, per type of the image it is made from, as mime:mime entries, e.g. image/vnd.microsoft.icon:image/png. The target is asked for by the name WEBDAV_THUMBNAIL_GENERATOR_FORMATS gives it. A type that is not listed, and one whose target the generator cannot write, becomes a JPEG." introductionVersion:"%%NEXT%%"`

	// the yaml key stays unprefixed, a config file written against it has to keep working
	ThumbnailFontMapFile string `yaml:"font_map_file" env:"WEBDAV_THUMBNAILS_TXT_FONTMAP_FILE;THUMBNAILS_TXT_FONTMAP_FILE" desc:"The path to a font map file for txt thumbnails." introductionVersion:"%%NEXT%%"`

	Context context.Context `yaml:"-"`
}
