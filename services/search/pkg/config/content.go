package config

// Extractor defines which extractor to use
type Extractor struct {
	Type             string        `yaml:"type" env:"SEARCH_EXTRACTOR_TYPE" desc:"Defines the content extraction engine. Defaults to 'basic'. Supported values are: 'basic' and 'tika'." introductionVersion:"1.0.0"`
	CS3AllowInsecure bool          `yaml:"cs3_allow_insecure" env:"OC_INSECURE;SEARCH_EXTRACTOR_CS3SOURCE_INSECURE" desc:"Ignore untrusted SSL certificates when connecting to the CS3 source." introductionVersion:"1.0.0"`
	Tika             ExtractorTika `yaml:"tika"`
}

// ExtractorTika configures the Tika extractor
type ExtractorTika struct {
	TikaURL        string `yaml:"tika_url" env:"OC_TIKA_URL;SEARCH_EXTRACTOR_TIKA_TIKA_URL" desc:"URL of the tika server." introductionVersion:"1.0.0"`
	CleanStopWords bool   `yaml:"clean_stop_words" env:"SEARCH_EXTRACTOR_TIKA_CLEAN_STOP_WORDS" desc:"Defines if stop words should be cleaned or not. See the documentation for more details." introductionVersion:"1.0.0"`
}
