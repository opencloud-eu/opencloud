package config

// Settings defines the storage of the per-user and shared office settings (Collabora "Options" dialog).
type Settings struct {
	Disabled bool   `yaml:"disabled" env:"COLLABORATION_SETTINGS_DISABLED" desc:"Disable the office settings endpoints. If disabled, Collabora does not offer the Options dialog." introductionVersion:"%%NEXT%%"`
	DataPath string `yaml:"data_path" env:"COLLABORATION_SETTINGS_DATA_PATH" desc:"Path on the filesystem where the office settings (dictionaries, autotext, preferences) are stored. It must be shared by all instances of the collaboration service. If not defined, the root directory derives from $OC_BASE_DATA_PATH/collaboration/settings" introductionVersion:"%%NEXT%%"`
}
