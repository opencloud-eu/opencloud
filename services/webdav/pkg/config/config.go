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

	Thumbnails Thumbnails `yaml:"thumbnails"`

	TikaURL                          string   `yaml:"tika_url" env:"OC_TIKA_URL;WEBDAV_TIKA_URL" desc:"URL of an Apache Tika server. When set, thumbnails are generated for documents, raw photos, audio cover art and other formats Tika can extract a preview from. Needs Tika 4.2.0 or newer, started with TIKA_PRESETS=thumbnail. A 4.1.x server works as well, with the preset activated in its config file instead." introductionVersion:"%%NEXT%%"`
	TikaThumbnailMimeTypes           []string `yaml:"tika_thumbnail_mime_types" env:"OC_TIKA_THUMBNAIL_MIME_TYPES;WEBDAV_TIKA_THUMBNAIL_MIME_TYPES" desc:"The mime types sent to Tika for their thumbnail, which replaces the built-in list of the formats a Tika release serves one for. The camera raws are in that list because reva types them as images and an image is handed to the generator as it is unless something claims it. A single '*' sends every type that nothing else claims, and an empty list sends nothing at all, which leaves a Tika shared with the search service to the search service alone. An entry can map a type to the one Tika knows the format by, e.g. image/x-nikon-nef:image/x-raw-nikon. Tika takes a declared type only when it refines what it detected itself, so an entry pins the leaf type its detector left at the parent, a raw photo it only read as a plain tiff. The type decides which parser reads the file, so a raw read as a tiff loses its maker notes and with them the embedded preview; the formats shipped here are detected down to the leaf type, so no entry is needed for them." introductionVersion:"%%NEXT%%"`
	AdditionalTikaThumbnailMimeTypes []string `yaml:"additional_tika_thumbnail_mime_types" env:"OC_TIKA_ADDITIONAL_THUMBNAIL_MIME_TYPES;WEBDAV_TIKA_ADDITIONAL_THUMBNAIL_MIME_TYPES" desc:"Additional mime types sent to Tika for their thumbnail, on top of the built-in list or of whatever OC_TIKA_THUMBNAIL_MIME_TYPES names instead. An entry carries a type hint the same way." introductionVersion:"%%NEXT%%"`

	Context context.Context `yaml:"-"`
}

// Thumbnails is how this service makes previews and where it keeps them. Each
// setting is named the way the thumbnails service names the same one, so
// WEBDAV_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE and the yaml key below it read as
// one name.
type Thumbnails struct {
	GeneratorURL     string `yaml:"generator_url" env:"WEBDAV_THUMBNAILS_GENERATOR_URL" desc:"Base URL of the thumbnail generator service, e.g. http://thumbnails:9186" introductionVersion:"%%NEXT%%"`
	GeneratorTimeout string `yaml:"generator_timeout" env:"WEBDAV_THUMBNAILS_GENERATOR_TIMEOUT" desc:"HTTP timeout for requests to the thumbnail generator, e.g. 30s" introductionVersion:"%%NEXT%%"`
	MaxInputFileSize string `yaml:"max_input_image_file_size" env:"THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE;OC_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE;WEBDAV_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE" desc:"Maximum file size of an input image a thumbnail is made of. The generator refuses a larger body, so both read the same name by default. Usable common abbreviations: [KB, KiB, MB, MiB, GB, GiB], example: 50MB" introductionVersion:"%%NEXT%%"`

	CacheBackend string `yaml:"cache_backend" env:"WEBDAV_THUMBNAILS_CACHE_BACKEND" desc:"Cache backend for imagor thumbnails: 'none', 'memory', 'file', or 's3'. Default: file." introductionVersion:"%%NEXT%%"`
	CacheDir     string `yaml:"cache_dir" env:"WEBDAV_THUMBNAILS_CACHE_DIR" desc:"Directory for file-based thumbnail cache (Default: $OC_BASE_DATA_PATH/thumbnails/files). Only used when WEBDAV_THUMBNAILS_CACHE_BACKEND is 'file'." introductionVersion:"%%NEXT%%"`
	// kept so a deployment that configured the thumbnails service keeps its cache
	FilesystemStorageRoot string   `yaml:"filesystemstorage_root" env:"THUMBNAILS_FILESYSTEMSTORAGE_ROOT" desc:"Directory the thumbnails service stored its files in. Read as a fallback for WEBDAV_THUMBNAILS_CACHE_DIR." introductionVersion:"1.0.0" deprecationVersion:"%%NEXT%%" removalVersion:"%%NEXT_PRODUCTION_VERSION%%" deprecationInfo:"THUMBNAILS_FILESYSTEMSTORAGE_ROOT is deprecated. The webdav service owns the thumbnail cache now, so use WEBDAV_THUMBNAILS_CACHE_DIR instead." deprecationReplacement:"WEBDAV_THUMBNAILS_CACHE_DIR"`
	Resolutions           []string `yaml:"resolutions" env:"THUMBNAILS_RESOLUTIONS;WEBDAV_THUMBNAILS_RESOLUTIONS" desc:"Allowed thumbnail edge lengths, like 512. Each axis of a requested size is rounded up to the next one of these before the box is sent to the generator. Legacy WidthxHeight entries are still accepted and contribute both of their sides." introductionVersion:"%%NEXT%%"`
	CacheS3Bucket         string   `yaml:"cache_s3_bucket" env:"WEBDAV_THUMBNAILS_CACHE_S3_BUCKET" desc:"S3 bucket name for thumbnail cache when WEBDAV_THUMBNAILS_CACHE_BACKEND is 's3'." introductionVersion:"%%NEXT%%"`
	CacheS3Region         string   `yaml:"cache_s3_region" env:"WEBDAV_THUMBNAILS_CACHE_S3_REGION" desc:"S3 region for thumbnail cache when WEBDAV_THUMBNAILS_CACHE_BACKEND is 's3'." introductionVersion:"%%NEXT%%"`
	CacheS3Endpoint       string   `yaml:"cache_s3_endpoint" env:"WEBDAV_THUMBNAILS_CACHE_S3_ENDPOINT" desc:"S3 endpoint URL (for MinIO or compatible services) when WEBDAV_THUMBNAILS_CACHE_BACKEND is 's3'." introductionVersion:"%%NEXT%%"`
	CacheS3AccessKey      string   `yaml:"cache_s3_access_key" env:"WEBDAV_THUMBNAILS_CACHE_S3_ACCESS_KEY" desc:"S3 access key for thumbnail cache authentication." introductionVersion:"%%NEXT%%"`
	CacheS3SecretKey      string   `yaml:"cache_s3_secret_key" env:"WEBDAV_THUMBNAILS_CACHE_S3_SECRET_KEY" desc:"S3 secret key for thumbnail cache authentication." introductionVersion:"%%NEXT%%"`
	AdditionalSourceTypes []string `yaml:"additional_source_types" env:"OC_THUMBNAILS_ADDITIONAL_SOURCE_TYPES;WEBDAV_THUMBNAILS_ADDITIONAL_SOURCE_TYPES" desc:"Additional mime types a preview is announced and attempted for, e.g. image/vnd.microsoft.icon for a generator that reads icons. The built-in list is included; this only adds to them. A type the running generator cannot read answers 404." introductionVersion:"%%NEXT%%"`

	AdditionalGeneratorFormats []string `yaml:"additional_generator_formats" env:"OC_THUMBNAILS_ADDITIONAL_GENERATOR_FORMATS;WEBDAV_THUMBNAILS_ADDITIONAL_GENERATOR_FORMATS" desc:"Additional formats the thumbnail generator writes, one per type it writes, as type:format entries, e.g. image/webp:webp. The format is the name the generator's own format filter uses. The built-in list is included; an entry for a type that is already listed replaces it. A type that is not listed is asked for as a JPEG instead, because what the generator behind the URL really writes is its own business." introductionVersion:"%%NEXT%%"`
	AdditionalOutputTypes      []string `yaml:"additional_output_types" env:"OC_THUMBNAILS_ADDITIONAL_OUTPUT_TYPES;WEBDAV_THUMBNAILS_ADDITIONAL_OUTPUT_TYPES" desc:"Additional types a thumbnail is returned in, one per type of the image it is made from, as source:target entries, e.g. image/vnd.microsoft.icon:image/png. The built-in list is included; an entry for a source type that is already listed replaces it. The target is asked for by the name WEBDAV_THUMBNAILS_ADDITIONAL_GENERATOR_FORMATS gives it. A type that is not listed becomes a JPEG; a target this service has no format name for is asked for as a JPEG as well." introductionVersion:"%%NEXT%%"`

	FontMapFile string `yaml:"txt_fontmap_file" env:"THUMBNAILS_TXT_FONTMAP_FILE;WEBDAV_THUMBNAILS_TXT_FONTMAP_FILE" desc:"The path to a font map file for txt thumbnails." introductionVersion:"%%NEXT%%"`
}
