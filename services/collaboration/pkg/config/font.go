package config

type Font struct {
	AssetPath   string `yaml:"asset_path" env:"COLLABORATION_FONT_ASSET_PATH" desc:"Serve fonts from a path on the filesystem instead of the builtin assets. If not defined, the root directory derives from $OC_BASE_DATA_PATH/collaboration/fonts" introductionVersion:"7.3.0"`
	PreviewText string `yaml:"preview_text" env:"COLLABORATION_FONT_PREVIEW_TEXT" desc:"The text that will be displayed in the font preview." introductionVersion:"7.3.0"`
}
