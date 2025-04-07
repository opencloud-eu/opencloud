package config

import "github.com/opencloud-eu/reva/v2/pkg/sharedconf"

// Config holds the config options that need to be passed down to all ocdav handlers
type Config struct {
	Prefix string `mapstructure:"prefix"`
	// FilesNamespace prefixes the namespace, optionally with user information.
	// Example: if FilesNamespace is /users/{{substr 0 1 .Username}}/{{.Username}}
	// and received path is /docs the internal path will be:
	// /users/<first char of username>/<username>/docs
	FilesNamespace string `mapstructure:"files_namespace"`
	// WebdavNamespace prefixes the namespace, optionally with user information.
	// Example: if WebdavNamespace is /users/{{substr 0 1 .Username}}/{{.Username}}
	// and received path is /docs the internal path will be:
	// /users/<first char of username>/<username>/docs
	WebdavNamespace string `mapstructure:"webdav_namespace"`
	SharesNamespace string `mapstructure:"shares_namespace"`
	OCMNamespace    string `mapstructure:"ocm_namespace"`
	GatewaySvc      string `mapstructure:"gatewaysvc"`
	Timeout         int64  `mapstructure:"timeout"`
	Insecure        bool   `mapstructure:"insecure"`
	// If true, HTTP COPY will expect the HTTP-TPC (third-party copy) headers
	EnableHTTPTpc               bool                              `mapstructure:"enable_http_tpc"`
	PublicURL                   string                            `mapstructure:"public_url"`
	FavoriteStorageDriver       string                            `mapstructure:"favorite_storage_driver"`
	FavoriteStorageDrivers      map[string]map[string]interface{} `mapstructure:"favorite_storage_drivers"`
	Version                     string                            `mapstructure:"version"`
	VersionString               string                            `mapstructure:"version_string"`
	Edition                     string                            `mapstructure:"edition"`
	Product                     string                            `mapstructure:"product"`
	ProductName                 string                            `mapstructure:"product_name"`
	ProductVersion              string                            `mapstructure:"product_version"`
	AllowPropfindDepthInfinitiy bool                              `mapstructure:"allow_depth_infinity"`

	TransferSharedSecret string `mapstructure:"transfer_shared_secret"`

	NameValidation NameValidation `mapstructure:"validation"`

	MachineAuthAPIKey string `mapstructure:"machine_auth_apikey"`
}

// NameValidation is the validation configuration for file and folder names
type NameValidation struct {
	InvalidChars []string `mapstructure:"invalid_chars"`
	MaxLength    int      `mapstructure:"max_length"`
}

// Init initializes the configuration
func (c *Config) Init() {
	// note: default c.Prefix is an empty string
	c.GatewaySvc = sharedconf.GetGatewaySVC(c.GatewaySvc)

	if c.FavoriteStorageDriver == "" {
		c.FavoriteStorageDriver = "memory"
	}

	if c.Version == "" {
		c.Version = "10.0.11.5"
	}

	if c.VersionString == "" {
		c.VersionString = "10.0.11"
	}

	if c.Product == "" {
		c.Product = "reva"
	}

	if c.ProductName == "" {
		c.ProductName = "reva"
	}

	if c.ProductVersion == "" {
		c.ProductVersion = "10.0.11"
	}

	if c.NameValidation.InvalidChars == nil {
		c.NameValidation.InvalidChars = []string{"\f", "\r", "\n", "\\"}
	}

	if c.NameValidation.MaxLength == 0 {
		c.NameValidation.MaxLength = 255
	}
}
