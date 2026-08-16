package config

// Extractor defines which extractor to use
type Extractor struct {
	Type             string        `yaml:"type" env:"SEARCH_EXTRACTOR_TYPE" desc:"Defines the content extraction engine. Defaults to 'basic'. Supported values are: 'basic' and 'tika'." introductionVersion:"1.0.0"`
	CS3AllowInsecure bool          `yaml:"cs3_allow_insecure" env:"OC_INSECURE;SEARCH_EXTRACTOR_CS3SOURCE_INSECURE" desc:"Ignore untrusted SSL certificates when connecting to the CS3 source." introductionVersion:"1.0.0"`
	Tika             ExtractorTika `yaml:"tika"`
	Clip             ExtractorClip `yaml:"clip"`
}

// ExtractorTika configures the Tika extractor
type ExtractorTika struct {
	TikaURL        string `yaml:"tika_url" env:"SEARCH_EXTRACTOR_TIKA_TIKA_URL" desc:"URL of the tika server." introductionVersion:"1.0.0"`
	CleanStopWords bool   `yaml:"clean_stop_words" env:"SEARCH_EXTRACTOR_TIKA_CLEAN_STOP_WORDS" desc:"Defines if stop words should be cleaned or not. See the documentation for more details." introductionVersion:"1.0.0"`
}

// ExtractorClip configures the CLIP inference service used for semantic image
// search. It decorates the configured extractor, so it combines with 'basic'
// and 'tika'. The vector dimensionality is part of the index schema and
// therefore not configurable; the service verifies at startup that the
// configured model matches.
type ExtractorClip struct {
	URL      string `yaml:"url" env:"SEARCH_EXTRACTOR_CLIP_URL" desc:"URL of the CLIP inference service (immich machine-learning API). When set, image files are embedded during indexing to enable semantic search." introductionVersion:"7.5.0"`
	Model    string `yaml:"model" env:"SEARCH_EXTRACTOR_CLIP_MODEL" desc:"Name of the CLIP model to use. Must be a multilingual model producing 512-dimensional embeddings. Changing the model requires a full reindex." introductionVersion:"7.5.0"`
	MaxBytes uint64 `yaml:"max_bytes" env:"SEARCH_EXTRACTOR_CLIP_MAX_BYTES" desc:"Maximum image size in bytes to send to the inference service. Larger images are skipped." introductionVersion:"7.5.0"`
	Timeout  int    `yaml:"timeout" env:"SEARCH_EXTRACTOR_CLIP_TIMEOUT" desc:"Timeout in seconds for requests to the inference service." introductionVersion:"7.5.0"`
}
