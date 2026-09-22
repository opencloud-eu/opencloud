package config

type Font struct {
	AssetPath   string `yaml:"asset_path" env:"COLLABORATION_FONT_ASSET_PATH" desc:"Serve fonts from a path on the filesystem instead of the builtin assets. If not defined, the root directory derives from $OC_BASE_DATA_PATH/collaboration/fonts" introductionVersion:"7.3.0"`
	PreviewText string `yaml:"preview_text" env:"COLLABORATION_FONT_PREVIEW_TEXT" desc:"The text that will be displayed in the font preview." introductionVersion:"7.3.0"`
	BaseURL     string `yaml:"base_url" env:"COLLABORATION_FONT_BASE_URL" desc:"The base URL under which the font files are served. It must match the URL configured in the remote_font_config of the office suite. If not set, it defaults to $OC_URL/collaboration/fonts" introductionVersion:"7.3.0"`
}
