package config

// Reva defines all available REVA configuration.
type Reva struct {
	Address string `yaml:"address" env:"OC_REVA_GATEWAY" desc:"The CS3 gateway endpoint." introductionVersion:"1.0.0"`
}

// TokenManager is the config for using the reva token manager
type TokenManager struct {
	JWTSecret string `yaml:"jwt_secret" env:"OC_JWT_SECRET;SEARCH_JWT_SECRET,file" desc:"The secret to mint and validate jwt tokens." introductionVersion:"1.0.0"`
}
