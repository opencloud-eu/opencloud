package config

// App defines the available app configuration.
type App struct {
	Name        string `yaml:"name" env:"COLLABORATION_APP_NAME" desc:"The name of the app which is shown to the user. You can chose freely but you are limited to a single word without special characters or whitespaces. We recommend to use pascalCase like 'CollaboraOnline'." introductionVersion:"1.0.0"`
	Product     string `yaml:"product" env:"COLLABORATION_APP_PRODUCT" desc:"The WebOffice app, either Collabora, OnlyOffice, Microsoft365 or MicrosoftOfficeOnline." introductionVersion:"1.0.0"`
	Description string `yaml:"description" env:"COLLABORATION_APP_DESCRIPTION" desc:"App description" introductionVersion:"1.0.0"`
	Icon        string `yaml:"icon" env:"COLLABORATION_APP_ICON" desc:"Icon for the app" introductionVersion:"1.0.0"`

	Addr     string `yaml:"addr" env:"COLLABORATION_APP_ADDR" desc:"The URL where the WOPI app is located, such as https://127.0.0.1:8080." introductionVersion:"1.0.0"`
	Insecure bool   `yaml:"insecure" env:"COLLABORATION_APP_INSECURE" desc:"Skip TLS certificate verification when connecting to the WOPI app" introductionVersion:"1.0.0"`

	ProofKeys          ProofKeys `yaml:"proofkeys"`
	LicenseCheckEnable bool      `yaml:"licensecheckenable" env:"COLLABORATION_APP_LICENSE_CHECK_ENABLE" desc:"Enable license checking to edit files. Needs to be enabled when using Microsoft365 with the business flow." introductionVersion:"1.0.0"`

	ProductEdition string `yaml:"product_edition" env:"COLLABORATION_APP_PRODUCT_EDITION" desc:"The edition of the WebOffice app, it decides which features the app offers. Only used for EuroOffice, where 'ce', 'de' and 'ee' are supported and an empty value is the same as 'ce'." introductionVersion:"%%NEXT%%"`
}

type ProofKeys struct {
	Disable  bool   `yaml:"disable" env:"COLLABORATION_APP_PROOF_DISABLE" desc:"Disable the proof keys verification" introductionVersion:"1.0.0"`
	Duration string `yaml:"duration" env:"COLLABORATION_APP_PROOF_DURATION" desc:"Duration for the proof keys to be cached in memory, using time.ParseDuration format. If the duration can't be parsed, we'll use the default 12h as duration" introductionVersion:"1.0.0"`
}
